package main

import (
    "time"
    "os"
    "fmt"
)

type flipnote struct {
    id int
    author string
    filename string
    uploaded_at time.Time
}

type tmb []byte
var tmbSize int = 0x6A0

// Get the TMB for a given flipnote
// Used for flipnote previews on menu type 2
// Returns nil if failed to read file
func (f flipnote) getTmb() tmb {
    buf := make([]byte, tmbSize)
    path := fmt.Sprintf(dataPath + "/flipnotes/%s.ppm", f.filename)

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
