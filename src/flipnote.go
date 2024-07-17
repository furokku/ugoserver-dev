package main

import (
    "time"
    "os"
    "fmt"
)

type flipnote struct {
    id int
    author_id string
    author_name string
    parent_author_id string
    parent_author_name string
    author_filename string
    uploaded_at time.Time
    lock int
}

type tmb []byte
var tmbSize int = 0x6A0

func (f flipnote) TMB() tmb {
    buf := make([]byte, tmbSize)
    path := fmt.Sprintf(configuration.HatenaDir + "/hatena_storage/flipnotes/%d.ppm", f.id)

    file, err := os.Open(path)
    if err != nil {
        errorlog.Printf("failed to open %v: %v", path, err)
        return nil
    }

    defer file.Close()

    _, err = file.Read(buf)
    if err != nil {
        errorlog.Printf("failed to read %v: %v", path, err)
        return nil
    }

    return buf
}


// return whether a flipnote is locked
// 0 if not, 1 if it is
func (t tmb) flipnoteIsLocked() int {
    l := int( t[0x10] )

    if l != 0 && l != 1 {
        warnlog.Printf("invalid lock state")
        return 0
    } else {
        return l
    }
}
