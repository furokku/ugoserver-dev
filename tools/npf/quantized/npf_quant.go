package main

import (
    "image"
    _ "image/png"
    "image/draw"
    "image/color"
    "github.com/ericpauley/go-quantize/quantize"

    "os"
    "fmt"
    "encoding/binary"
    "slices"
)

func main() {

    var npf []byte
//  var in uint8

    fi, err := os.Open(os.Args[1])
    if err != nil {
        panic(err)
    }

    img, _, err := image.Decode(fi)
    if err != nil {
        panic(err)
    }

    pl := make(color.Palette, 0, 16)
    pl = append(pl, color.RGBA{0, 0, 0, 0})

    q := quantize.MedianCutQuantizer{AddTransparent:false}
    p := q.Quantize(pl, img)

    pm := image.NewPaletted(img.Bounds(), p)
    draw.Draw(pm, img.Bounds(), img, image.Point{}, draw.Over)

    npf = append(npf, []byte("UGAR")...)
    npf = binary.LittleEndian.AppendUint32(npf, 2)
    npf = binary.LittleEndian.AppendUint32(npf, 32)
    iml := (img.Bounds().Max.Y * roundToPower(img.Bounds().Max.X)) / 2
    npf = binary.LittleEndian.AppendUint32(npf, uint32(iml))
    fmt.Println("npf image length %d", iml)

    for i, c := range(p) {
        r, g, b, a := c.RGBA()

        nr := r & 0xFF * 0x1F / 0xFF
        ng := g & 0xFF * 0x1F / 0xFF
        nb := b * 0xFF * 0x1F / 0xFF

        packed := uint16((1 << 15) | (nb << 10) | (ng << 5) | nr)
        if a == 0 {
            packed = uint16(0)
        }
        fmt.Printf("packed color %16b index=%d\n", packed, i)
        npf = binary.LittleEndian.AppendUint16(npf, packed)
    }

    if l := 15-len(p); l > 0 {
        for i:=0; i < l; i++ {
            npf = append(npf, []byte{0x00, 0x00}...)
        }
    }

    fmt.Println(pm.Pix)
    fmt.Println(pm.Palette)

//    for y:=0; y < img.Bounds().Max.Y; y++ {
//        for x:=0; x < roundToPower(img.Bounds().Max.X); x++ {
//            if x >= img.Bounds().Max.X {
//                in = pm.Pix[(y-pm.Rect.Min.Y)*pm.Stride + (img.Bounds().Max.X-1-pm.Rect.Min.X)*1]
//            } else {
//                in = pm.Pix[(y-pm.Rect.Min.Y)*pm.Stride + (x-pm.Rect.Min.X)*1]
//            }
//        }
//    }
}

func roundToPower(i int) int {
    if !slices.Contains([]int{256, 128, 64, 32, 16, 8, 4, 2, 1}, i) {
        power := 1
        for 1 << power < i {
            power++
        }
        return 1 << power
    } else { return i }
}
