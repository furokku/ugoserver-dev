package main

import (
    "github.com/cascax/colorthief-go"
    "image"
    _ "image/png"

    "os"
    "fmt"

//  "encoding/binary"
)

var magic string = "UGAR"

func main() {
    fi, err := os.Open(os.Args[1])
    if err != nil {
        panic(err)
    }

    decoded, _, err := image.Decode(fi)
    if err != nil {
        panic(err)
    }

    colors, err := colorthief.GetPalette(decoded, 6)
    if err != nil {
        panic(err)
    }

    fmt.Println(colors)
}
