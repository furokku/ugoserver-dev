package ugoimg

import (
	"encoding/binary"
	"image"
	"io"
)

// see comments for ToNtft
func FromNtft(ntft []byte, w, h int) (image.Image, error) {
	im := image.NewNRGBA(image.Rect(0, 0, w, h))
	
	// Rounded width
	wr := round(w)
	
	for y:=0; y<h; y++ {
		for x:=0; x<wr; x++ {
			if x >= w { // ignore the padding
				continue
			}
			
			n := (wr * y + x) * 2
			read := ntft[n:n+2]
			pix := binary.LittleEndian.Uint16(read)
			c := unpackabgr(pix, true)

			//fmt.Printf("RAW=%02x%02x B=%b x=%d y=%d n=%d, RGBA=%02x%02x%02x%02x\n", read[0], read[1], pix, x, y, n, c.R, c.G, c.B, c.A)
			im.SetNRGBA(x, y, c)
		}
	}
	
	return im.SubImage(im.Bounds()), nil
}

func DecodeNtft(r io.Reader, w, h int) (image.Image, error) {
	ntft, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	
	return FromNtft(ntft, w, h)
}

// ppm: flipnote studio animation format
// 0x0 - 0x6a0: tmb/thumbnail, contains author information, preview, other data
// 0x6a0 onwards: animation data, contains frames, sound, 128-byte rsa1024 sha1 signature + 16 bytes 0x00 padding
// 256x192, each frame has two toggleable layers with its own pen color
// (red, blue, or inverse of paper color)
// the signature is made using the flipnote studio private key
// key SHA256 (pem format): 87f45ee349077c27538a3c44f4347f5153e9b1554b29a3b3957f91afdb084d47
//
// this decoder returns a list of image.Images with each frame
func FromPpm(ppm []byte) []image.Image {
	// read animation header
	//
	// frame offset table size, 4 bytes for each offset as this is uint32
	fots := uint32(binary.LittleEndian.Uint16(ppm[0x6a0 : 0x6a1+1]))
	//unk1h := binary.LittleEndian.Uint32(ppm[0x6a2 : 0x6a5+1]) // unknown, always 0
	ahf := binary.LittleEndian.Uint16(ppm[0x6a6 : 0x6a7+1]) // animation header flags
	//fmt.Printf("fots=%02x unk=%02x flag=%08b\n", fots, unk1h, ahf)
	
	// animation header flags
	//ah1 := ahf & 0x1 // unknown
	//loop := ahf >> 1 & 0x1 // loop flipnote if 1
	//ah3 := ahf >> 2 & 0x1 // unknown
	//ah4 := ahf >> 3 & 0x1 // unknown
	hide1 := ahf >> 4 & 0x1 // hide layer 1 if 1
	hide2 := ahf >> 5 & 0x1 // hide layer 2 if 1
	//ah7 := ahf >> 6 & 0x1 // always set
	
	fn := int(fots / 4) // # of frames
	//fmt.Printf("reading %d frames\n", fn)
	
	offsets := make([]uint32, fn)
	frames := make([]frame, fn)
	
	// read frame offset table
	// offsets are relative to the start of the animation data
	for i:=0; i<fn; i++ {
		o := binary.LittleEndian.Uint32(ppm[0x6a8+i*4 : 0x6a8+i*4+4])
		offsets[i] = o
		//fmt.Printf("found offset %08x for frame %d\n", o, i)
	}
	
	// read each frame and its header
	for n:=0; n<fn; n++ {
		this := frame{
		  	layer1: make([][]uint8, 192),
		  	layer2: make([][]uint8, 192),
		}
		
		//fmt.Printf("reading frame at %08x\n", offsets[n])
		cur := 0x6a8 + fots + offsets[n] // The next byte(s) that will be read
		fh := ppm[cur]
		ftx := 0 // frame translation x
		fty := 0 // frame translation y
		
		this.paper = int(fh & 0x1) // paper color
		this.pen1 = int(fh >> 1 & 0x3) // layer 1 pen color
		this.pen2 = int(fh >> 3 & 0x3) // layer 2 pen color
		ft := fh >> 5 & 0x3 // translate, read two int8 values if this is set
		fd := fh >> 7 & 0x1 // uses frame diffing (if 0)
		
		if ft != 0 {
			ftx = int(ppm[cur+1])
			fty = int(ppm[cur+2])
			cur += 2
		}
		cur += 1
		
		// arrays of line encodings for 192 lines (h)
		le1 := make([]uint8, 192) // top layer
		le2 := make([]uint8, 192) // bottom layer
		li := 0 // index
		
		// unpack line encoding from 48 bytes to 192 2-bit values
		// This needs to be done for both layers
		for byo:=0; byo<96; byo++ {
			// Byte offset
			b := ppm[cur]
			cur++
			
			for bio:=0; bio<8; bio+=2 {
				// Bit offset
				if byo < 48 {
					le1[li] = (b >> bio) & 0x03
				} else {
					le2[li-192] = (b >> bio) & 0x03
				}
				//fmt.Printf("byte offset=%d bit offset=%d line encoding=%d\n", byo, bio, (b >> bio) & 0x03)
				li += 1
			}
		}

		// decode and read the lines
		for y:=0; y<384; y++ { // Read 192 lines * 2 layers
			line1 := make([]uint8, 256) // array of pixels in the line (w), layer 1
			line2 := make([]uint8, 256) // layer 2
			pix := 0
			
			yr := y
			line := &line1 // To make this simpler, the array to write to is referenced by a pointer
			le := &le1     // so that it can be switched on layer 2 with an if like this
			cl := &this.layer1
			if y >= 192 {
				yr = y-192
				line = &line2
				le = &le2
				cl = &this.layer2
			}
			
			//fmt.Printf("frame=%d line=%d(%d) encoding=%d\n", n, yr, y, (*le)[yr])
			switch (*le)[yr] {
			case 0: // line is empty
				//fmt.Printf("frame=%d line=%d(%d) empty\n", n, yr, y)
			case 1: // line is compressed
				chunk_flags := binary.BigEndian.Uint32(ppm[cur : cur + 4])
				//fmt.Printf("frame=%d line=%d(%d) chunk_flags=%032b offset=%x\n", n, yr, y, chunk_flags, cur)
				cur += 4

				for cfb:=0;cfb<32;cfb++ { // loop through chunk flag bits
					bit := (chunk_flags << cfb) & 0x80000000
					if bit == 0x80000000 { // check if the bit is set, starting at the highest bit
						chunk := ppm[cur]
						//fmt.Printf("frame=%d line=%d(%d) cfb=%d chunk=%08b offset=%x\n", n, yr, y, cfb, chunk, cur)
						cur++
						
						for cb:=0; cb<8; cb++ { // go through each bit of the chunk
							(*line)[pix] = chunk >> cb & 0x1
							pix++
						}
					} else {
						//fmt.Printf("cfb=%d not set, skipping\n", cfb)
						pix += 8
					}
				}
			case 2: // same as 1 but all pixels are set to 1 before decoding
				for i:=0;i<256;i++ {
					(*line)[i] = 1
				}
				chunk_flags := binary.BigEndian.Uint32(ppm[cur : cur + 4])
				//fmt.Printf("frame=%d line=%d(%d) chunk_flags=%032b offset=%x\n", n, yr, y, chunk_flags, cur)
				cur += 4

				for cfb:=0;cfb<32;cfb++ { // loop through chunk flag bits
					bit := (chunk_flags << cfb) & 0x80000000
					if bit == 0x80000000 { // check if the bit is set, starting at the highest bit
						chunk := ppm[cur]
						//fmt.Printf("frame=%d line=%d(%d) cfb=%d chunk=%08b offset=%x\n", n, yr, y, cfb, chunk, cur)
						cur++
						
						for cb:=0; cb<8; cb++ { // go through each bit of the chunk
							(*line)[pix] = chunk >> cb & 0x1
							pix++
						}
					} else {
						//fmt.Printf("cfb=%d not set, skipping\n", cfb)
						pix += 8
					}
				}
			case 3: // all chunks are used so no need for the flags
				for i:=0;i<32;i++ {
					chunk := ppm[cur]
					cur++

					for cb:=0;cb<8;cb++ { // go through each bit
						(*line)[pix] = chunk >> cb & 0x1
						pix++
					}
				}
			}
			//fmt.Printf("frame=%d line=%d(%d) data=%v\n", n, yr, y, *line)
			(*cl)[yr] = *line
		}
		
		// if the flag is set in the frame header, the frame uses frame diffing
		// so it needs to be XORed over the previous frame on both layers
		if fd == 0x0 {
			//fmt.Printf("frame=%d diffing\n", n)
			for y:=0;y<192;y++ {
				if y - fty < 0 {
					continue
				}
				if y - fty >= 192 {
					break
				}
				
				for x:=0;x<256;x++ {
					if x - ftx < 0 {
						continue
					}
					if x - ftx >= 256 {
						break
					}
					
					this.layer1[y][x] ^= frames[n-1].layer1[y-ftx][x-ftx]
					this.layer2[y][x] ^= frames[n-1].layer2[y-ftx][x-ftx]
				}
			}
		}
		
		frames[n] = this
	}
	
	//fmt.Printf("decoded %d frames\n", len(frames))
	
	m := make([]image.Image, fn)
	for n, frame := range frames {
		//fmt.Printf("frame=%d converting\n", n)
		//fmt.Println(frame)
		this := image.NewNRGBA(image.Rect(0, 0, 256, 192))
		
		// draw bottom layer first
		pen, paper := framepen(frame.pen2, frame.paper)
		for y:=0;y<192;y++ {
			for x:=0;x<256;x++ {
				pixel := frame.layer2[y][x]
				if pixel == 1 && hide2 != 1 {
					this.SetNRGBA(x, y, pen)
				} else {
					this.SetNRGBA(x, y, paper)
				}
			}
		}
		
		// then the top layer over it
		pen, _ = framepen(frame.pen1, frame.paper)
		for y:=0;y<192;y++ {
			for x:=0;x<256;x++ {
				pixel := frame.layer1[y][x]
				if pixel == 1 && hide1 != 1{
					this.SetNRGBA(x, y, pen)
				}
			}
		}
		m[n] = this.SubImage(this.Bounds())
	}
	
	return m
}