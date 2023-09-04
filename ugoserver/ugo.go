package main

import (
    "fmt"
    "encoding/binary"
)

type ugomenu struct {
    entries []menuEntry
    embed [][]byte
}

type menuEntry struct {
    entryType uint
    data []string
}


// build ugomenu to be able to return from w.Write
func (u ugomenu) Pack() []byte {

    var header, menus, embedded []byte
    sections := 1 // there is always at least one section
    emb := false
    magic := "UGAR"

    // an ugomenu must have this section
    for _, item := range u.entries {
        menus = append(menus, newline)
        menus = append(menus, fmt.Sprint(item.entryType)...)

        for n := range item.data {
            menus = append(menus, tab)
            menus = append(menus, item.data[n]...)
        }
    }

    menus = padBytes(menus)

    // embedded content can be omitted, but is required
    // for things like custom icons or layout 2
    if len(u.embed) != 0 {
        for _, embed := range u.embed {
            embedded = append(embedded, embed...)
        }

        embedded = padBytes(embedded)

        emb = true
        sections = 2
    }

    // needs to be little endian because dsi
    header = append(header, magic...)
    header = binary.LittleEndian.AppendUint32(header, uint32(sections))
    header = binary.LittleEndian.AppendUint32(header, uint32(len(menus)))
    if emb { header = binary.LittleEndian.AppendUint32(header, uint32(len(embedded))) }
    
    return append(header, append(menus, embedded...)...)
}
