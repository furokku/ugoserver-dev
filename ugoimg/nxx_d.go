package ugoimg

import (
	"image"
)

func FromNtft(ntft []byte, w int, h int) (image.Image, error) {
	im := image.NewRGBA(image.Rect(0, 0, w, h))
	
	// todo
	
	return im.SubImage(im.Bounds()), nil
}