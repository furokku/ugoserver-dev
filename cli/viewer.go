package main

import (
	"fmt"
	"strings"

	"os"
	"time"

	"image"
	"image/draw"

	"floc/ugoserver/nx"

	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/size"
)

const (
    minw = 200
    minh = 200
)

func view(im string, w, h int) {

	spl := strings.Split(im, ".")
	ext := spl[len(spl)-1]

	fi, err := os.Open(im)
	if err != nil {
		panic(err)
	}

	var junk image.Image

	switch ext {
	case "npf":
		junk, err = nx.DecodeNpf(fi, w, h)
		if err != nil {
			panic(err)
		}
	case "nbf":
		junk, err = nx.DecodeNbf(fi, w, h)
		if err != nil {
			panic(err)
		}
	case "ntft":
		junk, err = nx.DecodeNtft(fi, w, h)
		if err != nil {
			panic(err)
		}
	default:
		print_usage()
	}

	wiw := func() int {
		if w > minw {
			return w
		}
		return minw
	}()
	wih := func() int {
		if h > minh {
			return h
		}
		return minh
	}()
	// start all of the gui stuff
	driver.Main(func(s screen.Screen) {
		wi, err := s.NewWindow(&screen.NewWindowOptions{
			Title:  fmt.Sprintf("ugotool: viewing %s", im),
			Width:  wiw,
			Height: wih,
		})
		if err != nil {
			panic(err)
		}
		defer wi.Release()

		sb, err := s.NewBuffer(image.Point{wiw, wih})
		if err != nil {
			panic(err)
		}
		defer sb.Release()
		pixbuf := sb.RGBA()

		for {
			draw.Draw(pixbuf, pixbuf.Bounds(), junk, image.Point{}, draw.Src)
			//wi.Send(paint.Event{External:true})
			wi.Upload(image.Point{0, 0}, sb, sb.Bounds())
			wi.Publish()

			switch e := wi.NextEvent().(type) {
			case key.Event:
				if e.Code == key.CodeEscape {
					return
				}

			case lifecycle.Event:
				if e.To == lifecycle.StageDead {
					return
				}

			case size.Event:
				// will crash if width/height == 0; workaround:
				es := image.Point{func() int {
					if e.WidthPx == 0 {
						return 1
					}
					return e.WidthPx
				}(), func() int {
					if e.HeightPx == 0 {
						return 1
					}
					return e.HeightPx
				}()}
				sb.Release()
				sb, err = s.NewBuffer(es)
				if err != nil {
					panic(err)
				}
				pixbuf = sb.RGBA()
			}
			time.Sleep(time.Millisecond * 5)
		}
	})
}
