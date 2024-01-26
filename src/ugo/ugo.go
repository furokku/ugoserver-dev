package ugo

import (
    "fmt"
    "encoding/binary"
)

const magic string = "UGAR"

type Ugomenu struct {
    Entries []MenuEntry
    Embed [][]byte
}

type MenuEntry struct {
    EntryType uint
    Data []string
}


// compile ugomenu to array of bytes
func (u Ugomenu) Pack() []byte {

    var header, menus, embedded []byte
    sections := 1 // there is always at least one section
    emb := false

    // an ugomenu must have this section
    for _, item := range u.Entries {
        menus = append(menus, 0x0A)
        menus = append(menus, fmt.Sprint(item.EntryType)...)

        for n := range item.Data {
            menus = append(menus, 0x09)
            menus = append(menus, item.Data[n]...)
        }
    }

    menus = padBytes(menus)

    // embedded content can be omitted, but is required
    // for things like custom icons or layout 2
    //
    // Should be ntft or tmb
    if len(u.Embed) > 0 {
        for _, embed := range u.Embed {
            embedded = append(embedded, embed...)
        }

        embedded = padBytes(embedded)

        emb = true
        sections = 2
    }

    // Has to be little endian byte order
    header = append(header, magic...)
    header = binary.LittleEndian.AppendUint32(header, uint32(sections))
    header = binary.LittleEndian.AppendUint32(header, uint32(len(menus)))
    if emb { header = binary.LittleEndian.AppendUint32(header, uint32(len(embedded))) }
    
    return append(header, append(menus, embedded...)...)
}


// 4 byte padding for ugomenus
func padBytes(d []byte) []byte {
    var padded []byte = d

    if x := len(d) % 4; x != 0 {
        for i := 0; i < (4-x); i++ {
            padded = append(padded, 0x00)
        }
    }
    return padded
}
