package main

import (
	"floc/ugoserver/nxlib"

	"github.com/esimov/colorquant"

	"image"
	"image/color/palette"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"

	"fmt"
	"strings"

	"bufio"
	"io"
	"net"

	"os"
	"os/signal"
)

func print_usage() {
    fmt.Printf("usage: %v console [address] | img/imgq infile.jpg/png outfile.npf/nbf/ntft\n", os.Args[0])
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
            console("/tmp/ugoserver.sock")
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

func img(infile string, outfile string, quantize bool) {

    spl := strings.Split(outfile, ".")
    ext := spl[len(spl)-1]
    
    fi, err := os.Open(infile)
    if err != nil {
        panic(err)
    }
    defer fi.Close()
    
    fo, err := os.OpenFile(outfile, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
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
        err := nxlib.EncodeNpf(fo, img)
        if err != nil {
            panic(err)
        }
    case "nbf":
        if quantize { img = colorquant.NoDither.Quantize(src, dst, 256, false, true) }
        err := nxlib.EncodeNbf(fo, img)
        if err != nil {
            panic(err)
        }
    case "ntft":
        err := nxlib.EncodeNtft(fo, src)
        if err != nil {
            panic(err)
        }
    }
    
    fmt.Printf("encoded %v (%dx%d) to %v", infile, src.Bounds().Max.X, src.Bounds().Max.Y, outfile)
}