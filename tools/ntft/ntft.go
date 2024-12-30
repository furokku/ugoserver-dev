package ntft

import (
	"encoding/binary"
	"fmt"
	"image"
	_ "image/png"
	"os"
)

var next uint16
var bytes []byte

func nmain() {

    fi, err := os.Open(os.Args[1])
    if err != nil {
        panic(err)
    }

    decoded, _, err := image.Decode(fi)
    if err != nil {
        panic(err)
    }

    fmt.Println(decoded.Bounds().Max.X, decoded.Bounds().Max.Y)
    if decoded.Bounds().Max.X != 32 || decoded.Bounds().Max.Y != 32 {
        fmt.Println("wrong input size")
        os.Exit(1)
    }

    for y:=0;y<32;y++ {
        for x:=0;x<32;x++ {
            r, g, b, a := decoded.At(x, y).RGBA()

            nr := r & 0xFF * 0x1F / 0xFF
            ng := g & 0xFF * 0x1F / 0xFF
            nb := b & 0xFF * 0x1F / 0xFF
            na := uint32(0)
            if a & 0xFF >= 0x80 { na = uint32(1)}

            next = uint16((na << 15) | (nb << 10) | (ng << 5) | nr)
            bytes = binary.LittleEndian.AppendUint16(bytes, next)
        }
    }

    os.WriteFile(os.Args[2], bytes, 0644)
}
