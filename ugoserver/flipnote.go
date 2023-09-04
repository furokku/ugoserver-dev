package main

import (
    "time"
    "os"
    "fmt"
    "log"
)

type (
    flipnote struct {
        id int
        author string
        filename string
        uploaded_at time.Time
    }

    tmb []byte
)

var (
    tmbOffset int = 0x6A0
)

func getTmbData(au string, fn string) tmb {
    buf := make([]byte, tmbOffset)
    path := fmt.Sprintf("/srv/ugoserver/hatena/flipnotes/creators/%s/%s.ppm", au, fn)

    file, err := os.Open(path)
    if err != nil {
        log.Fatalf("getTmbData: %v", err)
    }

    defer file.Close()

    n, err := file.Read(buf)
    if err != nil {
        log.Fatalf("getTmbData: %v", err)
    } else if n != tmbOffset {
        log.Printf("getTmbData: WARNING: read %v bytes instead of 1696", n)
    }

    return buf
}


func (t tmb) flipnoteIsLocked() uint {
    return uint(t[0x10])
}
