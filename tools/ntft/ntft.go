package ntft

import (
	"image"

	"errors"
	"fmt"

	"encoding/binary"
)

var (
    ErrInvalidSize = errors.New("wrong input size! image must be 32x32")
)

func ToNtft(img image.Image) ([]byte, error) {

    var next uint16
    var bytes []byte

    fmt.Println(img.Bounds().Max.X, img.Bounds().Max.Y)
    if img.Bounds().Max.X != 32 || img.Bounds().Max.Y != 32 {
        return nil, ErrInvalidSize
    }

    for y:=0;y<32;y++ {
        for x:=0;x<32;x++ {
            r, g, b, a := img.At(x, y).RGBA()

            nr := r & 0xFF * 0x1F / 0xFF
            ng := g & 0xFF * 0x1F / 0xFF
            nb := b & 0xFF * 0x1F / 0xFF
            na := uint32(0)
            if a & 0xFF >= 0x80 { na = uint32(1)}

            next = uint16((na << 15) | (nb << 10) | (ng << 5) | nr)
            bytes = binary.LittleEndian.AppendUint16(bytes, next)
        }
    }

    return bytes, nil
}
