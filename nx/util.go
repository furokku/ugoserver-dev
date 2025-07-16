package nx

import (
	"errors"
	"image/color"
	"slices"
)

const (
    image_magic string = "UGAR"
    animation_magic string = "PARA"
)

var (
    ErrNpfTooManyColors = errors.New("npf may not have more than 15 colors, excluding transparency")
    ErrNbfTooManyColors = errors.New("nbf may not have more than 256 colors")
    ErrNoColor = errors.New("[internal] color does not exist in map, this is an issue with ugoimg")
    ErrNbfTransparent = errors.New("nbf may not have transparent pixels")
    ErrNtftInvalidSize = errors.New("ntft cannot be larger than 128x128")
    ErrNbfInvalidSize = errors.New("nbf must be 256x192")
    ErrNotPpm = errors.New("data is missing ppm file magic. not a ppm")
    ErrOffsetBeyondData = errors.New("frame offset is beyond possible limits")
    ErrNotNx = errors.New("missing magic")
    ErrNot2Sects = errors.New("image has != 2 sections")

    // ppm colors
	black = color.NRGBA{ R: 0x0e, G: 0x0e, B: 0x0e, A: 0xff }
	white = color.NRGBA{ R: 0xff, G: 0xff, B: 0xff, A: 0xff }
	red = color.NRGBA{ R: 0xff, G: 0x2a, B: 0x2a, A: 0xff }
	blue = color.NRGBA{ R: 0x0a, G: 0x39, B: 0xff, A: 0xff }
)

// a single animation frame in a ppm
type frame struct {
	layer1 [192][256]uint8
	layer2 [192][256]uint8
	pen1 int
	pen2 int
	paper int
}


// round a number up to the nearest 2^x 
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

func packabgr(c color.Color, alpha bool) uint16 {
    r, g, b, a := c.RGBA()

    na := uint16(1)
    if alpha && a * 0xFF / 0xFFFF <= 0x80 { na = uint16(0) }

    return (na << 15) | (fast5(b) << 10) | (fast5(g) << 5) | fast5(r)
}

func unpackabgr(c uint16, alpha bool) color.NRGBA {
    
    a := uint8(c >> 15)
    if alpha && a == 0 { a = 0x00 } else { a = 0xFF }
    
    return color.NRGBA{
        R: fast8(c & 0x1F),
        G: fast8(c >> 5 & 0x1F),
        B: fast8(c >> 10 & 0x1F),
        A: a,
    }
}

// 5 bits -> 8 bits
func fast8(s uint16) uint8 {
    return uint8((s * 527 + 23) >> 6)
}

// 16 bits -> 5 bits
func fast5(s uint32) uint16 {
    return uint16((s * 31745 + 33538048) >> 26)
}

func framepen(pe int, pa int) (color.NRGBA, color.NRGBA) {
	var pec, pac color.NRGBA
	switch pa {
	case 0:
		pac = black
	case 1:
		pac = white
	}

	switch pe {
	case 1:
		pec = func() color.NRGBA { if pa == 1 { return black }; return white}() // simpler
	case 2:
		pec = red
	case 3:
		pec = blue
	}
	
	return pec, pac
}
