package ugoimg

import (
	"errors"
	"image/color"
	"slices"
)

const (
    magic string = "UGAR"
    cm uint8 = (1<<8-1) / (1<<5-1) // conversion magic 5bit to 8bit
)

var (
    ErrNpfTooManyColors = errors.New("npf may not have more than 15 colors, excluding transparency")
    ErrNbfTooManyColors = errors.New("nbf may not have more than 256 colors")
    ErrNoColor = errors.New("[internal] color does not exist in map, this is an issue with ugoimg")
    ErrNbfTransparent = errors.New("nbf may not have transparent pixels")
    ErrNtftInvalidSize = errors.New("ntft cannot be larger than 128x128")
    ErrNbfInvalidSize = errors.New("nbf must be 256x192")
)


func round(i int) int {
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

func packargb(c color.Color, alpha bool) uint16 {
    r, g, b, a := c.RGBA()

    nr := r & 0xFF * 0x1F / 0xFF
    ng := g & 0xFF * 0x1F / 0xFF
    nb := b & 0xFF * 0x1F / 0xFF
    na := uint32(1)
    if alpha && a & 0xFF >= 0x80 { na = uint32(1) }

    return uint16((na << 15) | (nb << 10) | (ng << 5) | nr)
}

func unpackargb(c uint16, alpha bool) color.Color {
    
    r := cm * uint8(c & 0x1F)
    g := cm * uint8(c >> 5 & 0x1F)
    b := cm * uint8(c >> 10 & 0x1F)
    a := uint8(c >> 15 & 0x01)
    if alpha && a == 0 { a = 0x00 } else { a = 0xFF }
    
    return color.RGBA{R: r, G: g, B: b, }
}