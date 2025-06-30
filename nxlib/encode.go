package nxlib

// encode.go: encode nbf/npf/ntft images

import (
	"image"
	"image/color"

	"encoding/binary"
	"io"
)

// npf: ugomemo image format for html pages with 15 colors + transparency
// image width MUST be padded to the nearest 2^x, ie 50px -> 64px
// this library uses an edge padding method where the edge's color is taken and used
// as filler. the regular image width must then be provided to the dsi, not the padded one
// this image format does not have any dimension information, so it must
// be provided to the dsi when used
// Can be all sorts of sizes, but this is not checked, so be careful
func ToNpf(img image.Image) ([]byte, error) {
    var im []int
    var out []byte
    colors := make(map[color.Color]int, 15)

    xm := img.Bounds().Max.X
    xr := round(xm)

    for y:=0; y < img.Bounds().Max.Y; y++ {
        // Cycle through each pixel to get a palette
        // of colors in the image and an integer array representing
        // each pixel as an index in the palette
        //
        // We will then use this to encode the image
        for x:=0; x < xr; x++ {
            c := img.At(x, y)
            if x >= xm {
                c = img.At(xm-1, y)
            }
            _,_,_,a := c.RGBA()
            _, ok := colors[c]
            // add the pixel to the palette if it is opaque and is not already in the palette
            if !ok && a != 0 {
                colors[c] = len(colors)+1
            }
            // index 0 is transparent
            if a == 0 {
                im = append(im, 0)
            } else {
                im = append(im, colors[c])
            }
            if colors[c] > 15 {
                return nil, ErrNpfTooManyColors
            }
//          fmt.Printf("x=%d y=%d index=%d color=%d %d %d %d\n", x, y, colors[c], r, g, b, a)
        }
    }
//  fmt.Println(im)

    out = append(out, image_magic...)
    out = binary.LittleEndian.AppendUint32(out, 2) // # of sections
    out = binary.LittleEndian.AppendUint32(out, 32) // palette section length
    out = binary.LittleEndian.AppendUint32(out, uint32(len(im)/2))
//  fmt.Printf("image section length %v\n", len(im)/2)

    out = append(out, []byte{0x00, 0x00}...) //first color is ignored ()
    for i:=1; i <= len(colors); i++ {
        c, ok := mapkey(colors, i)
        if !ok {
            return nil, ErrNoColor
        }
        out = binary.LittleEndian.AppendUint16(out, packabgr(c, false))
    }
    if l := 15-len(colors); l > 0 {
        for i:=0; i < l; i++ {
            out = append(out, []byte{0x00, 0x00}...)
        }
    }
    // Nibbles are reversed, so in image data 12 34
    // pixel #1 is 2, pixel #2 is 1, pixel #3 is 4, pixel #4 is 3, etc...
    for i:=0; i < len(im); i+=2 {
        b := uint8((im[i+1] << 4) | im[i])
//      fmt.Printf("byte n=%d %2x\n", i/2, b)
        out = append(out, b)
    }

    return out, nil
}

// generic function to encode npf
func EncodeNpf(w io.Writer, m image.Image) error {
    npf, err := ToNpf(m)
    if err != nil {
        return err
    }
    
    w.Write(npf)
    return nil
}


// nbf: ugomemo image format, used mainly for top screen backgrounds in html/ugomenus;
// similar to NPF but uses 8-bit index for image data, so max 256 colors,
// apparently should always be 256x192, but idk, so size isn't checked;
// Has no support for transparency
func ToNbf(img image.Image) ([]byte, error) {
    var im []int
    var out []byte
    colors := make(map[color.Color]int, 256)
    
    xm := img.Bounds().Max.X
    xr := round(xm)

    for y := 0; y < img.Bounds().Max.Y; y++ {
        for x := 0; x < xr; x++ {
            c := img.At(x, y)
            if x >= xm {
                c = img.At(xm-1, y)
            }
            _,_,_,a := c.RGBA()
            _, ok := colors[c]
            
            if a == 0 {
                return nil, ErrNbfTransparent
            }

            if !ok {
                colors[c] = len(colors)+1
            }

            if colors[c] > 256 {
                return nil, ErrNbfTooManyColors
            }

            im = append(im, colors[c])
//          fmt.Printf("x=%d y=%d index=%d color=%d %d %d %d\n", x, y, colors[c], r, g, b, a)
        }
    }
//  fmt.Println(im)

    out = append(out, image_magic...)
    out = binary.LittleEndian.AppendUint32(out, 2) // # of sections
    out = binary.LittleEndian.AppendUint32(out, 512) // palette section length
    out = binary.LittleEndian.AppendUint32(out, uint32(len(im)))
//  fmt.Printf("image section length %v\n", len(im)/2)

    for i:=1; i <= len(colors); i++ {
        c, ok := mapkey(colors, i)
        if !ok {
            return nil, ErrNoColor
        }
        out = binary.LittleEndian.AppendUint16(out, packabgr(c, false))
    }
    if l := 256-len(colors); l > 0 {
        for i:=0; i < l; i++ {
            out = append(out, []byte{0x00, 0x00}...)
        }
    }
    for i:=0; i < len(im); i++ {
        // Oops
        //b := uint8((im[i+1] << 4) | im[i])
//      fmt.Printf("byte n=%d %2x\n", i/2, b)
        out = append(out, uint8(im[i]))
    }

    return out, nil
}

// same thing
func EncodeNbf(w io.Writer, m image.Image) error {
    nbf, err := ToNbf(m)
    if err != nil {
        return err
    }
    
    w.Write(nbf)
    return nil
}


// ntft: ugomemo image format, used mainly for icons in ugomenus but also in html pages
// in ugomenus, icon must be 32x32
// has the same padding quirks as npf
// apparently has a size limit of 128x128
func ToNtft(img image.Image) ([]byte, error) {
    var bytes []byte

//  fmt.Println(img.Bounds().Max.X, img.Bounds().Max.Y)
    if img.Bounds().Max.X > 128 || img.Bounds().Max.Y > 128 {
        return nil, ErrNtftInvalidSize
    }

    // Format is simple, just a bunch of abgr1555 bytes one after another
    // No header or dimension information, no limit on colors either
    xm := img.Bounds().Max.X
    for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
        for x := img.Bounds().Min.X; x < round(xm); x++ {
            c := img.At(x, y)

            if x >= xm {
                c = img.At(xm-1, y)
            }

            bytes = binary.LittleEndian.AppendUint16(bytes, packabgr(c, true))
        }
    }

    return bytes, nil
}

// generic function to encode ntfts
func EncodeNtft(w io.Writer, m image.Image) error {
    ntft, err := ToNtft(m)
    if err != nil {
        return err
    }
    
    w.Write(ntft)
    return nil
}