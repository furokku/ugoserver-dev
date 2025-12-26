package main

import (
	"fmt"
	"strconv"
	"strings"

	"bufio"
	"io"
	"net"

	"os"
	"os/signal"

	"image"
	"image/color/palette"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"

	"floc/ugoserver/nx"

	"github.com/esimov/colorquant"
)

func print_usage() {
    fmt.Printf("usage: %v console [address] | img/imgq im.jpg/png out.npf/nbf/ntft | view im.npf/nbf/ntft width height\n", os.Args[0])
    os.Exit(1)
}

func main() {
    if len(os.Args) < 2 {
        print_usage()
    }
    
    switch os.Args[1] {

    case "console":
        if len(os.Args) >= 3 {
            console(os.Args[2])
        } else {
            console(fmt.Sprint(os.TempDir(), "/ugoserver.sock")) // windows isnt a mythical being that has a /tmp
        }
        
    case "img":
        if len(os.Args) >= 4 {
            img(os.Args[2], os.Args[3], false)
        } else {
            print_usage()
        }

    case "imgq":
        if len(os.Args) >= 4 {
            img(os.Args[2], os.Args[3], true)
        } else {
            print_usage()
        }
        
    case "view":
        if len(os.Args) >= 5 {
            w, err := strconv.Atoi(os.Args[3])
            if err != nil {
                print_usage()
            }
            h, err := strconv.Atoi(os.Args[4]) // make this optional and try to guess height
            if err != nil {
                print_usage()
            }
            view(os.Args[2], w, h)
        }
    default:
        print_usage()
    }
}

func console(addr string) {
    fmt.Printf("trying %v\n", addr)

    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, os.Interrupt)

    conn, err := net.Dial("unix", addr)
    if err != nil {
        panic(err)
    }

    go func(c net.Conn) {
        sig := <- sigs
        c.Close()
        fmt.Printf("caught %v, exiting\n", sig)
        os.Exit(0)
    }(conn)
    
    fmt.Printf("connected to server @ %s\n", addr)
    
    for {
        fmt.Print("> ")
        s := bufio.NewScanner(os.Stdin)
        s.Scan()
        if err := s.Err(); err != nil {
            panic(err)
        }

        n, err := conn.Write(s.Bytes())
        if err != nil {
            panic(err)
        }
        if n == 0 {
            continue
        }

        buf := make([]byte, 1048576) // read at most 1MiB, this should never be too little
        n, err = conn.Read(buf)
        if err != nil && err != io.EOF {
            panic(err)
        }

        fmt.Printf("%s\n", string(buf[:n]))
    }
}

func img(im string, out string, quantize bool) {

    spl := strings.Split(out, ".")
    ext := spl[len(spl)-1]
    
    fi, err := os.Open(im)
    if err != nil {
        panic(err)
    }
    defer fi.Close()
    
    fo, err := os.OpenFile(out, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
    if err != nil {
        panic(err)
    }
    defer fo.Close()
    
    src, _, err := image.Decode(fi)
    if err != nil {
        panic(err)
    }
    
    var img image.Image
    var dst draw.Image
    
    if quantize {
        dst = image.NewPaletted(src.Bounds(), palette.WebSafe)
    } else {
        img = src
    }
    
    switch ext {
    case "npf":
        if quantize { img = colorquant.NoDither.Quantize(src, dst, 15, false, true) }
        err := nx.EncodeNpf(fo, img)
        if err != nil {
            panic(err)
        }
    case "nbf":
        if quantize { img = colorquant.NoDither.Quantize(src, dst, 256, false, true) }
        err := nx.EncodeNbf(fo, img)
        if err != nil {
            panic(err)
        }
    case "ntft":
        err := nx.EncodeNtft(fo, src)
        if err != nil {
            panic(err)
        }
    }
    
    fmt.Printf("encoded %v (%dx%d) to %v", im, src.Bounds().Max.X, src.Bounds().Max.Y, out)
}
