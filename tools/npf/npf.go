package main

import (
    "image"
    "image/color"
    _ "image/png"

    "os"
    "fmt"
    "slices"

    "encoding/binary"
)

const magic string = "UGAR"
var colors = make(map[color.Color]int, 15)
var im []int
var out []byte

func main() {
    fi, err := os.Open(os.Args[1])
    if err != nil {
        panic(err)
    }

    img, _, err := image.Decode(fi)
    if err != nil {
        panic(err)
    }

    for y:=0; y < img.Bounds().Max.Y; y++ {
        for x:=0; x < roundToPower(img.Bounds().Max.X); x++ {
            c := img.At(x, y)
            if x >= img.Bounds().Max.X {
                c = img.At(img.Bounds().Max.X-1, y)
            }
            r,g,b,a := c.RGBA()
            _, ok := colors[c]
            if !ok && a != 0 {
                colors[c] = len(colors)+1
            }
            if a == 0 {
                im = append(im, 0)
            } else {
                im = append(im, colors[c])
            }
            if colors[c] > 15 {
                panic("more than 15 colors in image")
            }
            fmt.Printf("x=%d y=%d index=%d color=%d %d %d %d\n", x, y, colors[c], r, g, b, a)
        }
    }
    fmt.Println(im)

    out = append(out, magic...)
    out = binary.LittleEndian.AppendUint32(out, 2)
    out = binary.LittleEndian.AppendUint32(out, 32)
    out = binary.LittleEndian.AppendUint32(out, uint32(len(im)/2))
    fmt.Printf("image section length %v\n", len(im)/2)

    out = append(out, []byte{0x00, 0x00}...) //first color is ignored
    for i:=1; i <= len(colors); i++ {
        c, ok := mapkey(colors, i)
        if !ok {
            panic("color does not exist in map")
        }
        r, g, b, _ := c.RGBA()

        nr := r & 0xFF * 0x1F / 0xFF
        ng := g & 0xFF * 0x1F / 0xFF
        nb := b & 0xFF * 0x1F / 0xFF

        next := uint16((1 << 15) | (nb << 10) | (ng << 5) | nr)
        fmt.Printf("packed color %16b index=%d\n", next, i)
        out = binary.LittleEndian.AppendUint16(out, next)
    }
    if l := 15-len(colors); l > 0 {
        for i:=0; i < l; i++ {
            out = append(out, []byte{0x00, 0x00}...)
        }
    }
    for i:=0; i < len(im); i+=2 {
        b := uint8(((im[i+1] & 0b1111) << 4) | (im[i] & 0b1111))
        fmt.Printf("byte n=%d %2x\n", i/2, b)
        out = append(out, b)
    }

    os.WriteFile(os.Args[2], out, 0644)
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

func mapkey(m map[color.Color]int, value int) (color.Color, bool) {
    for k, v := range m {
        if v == value {
            return k, true
        }
    }
    return nil, false
}
