package main

import (
    "encoding/binary"
    "fmt"
    "os"
    "net/http"
    "github.com/gorilla/mux"
    "strings"
)

const magic string = "UGAR"


func (ju JsonUgo) Parse() Ugomenu {
    u := Ugomenu{}
    var temp []string

    for _, l := range ju.Layout {
        temp = append(temp, fmt.Sprint(l))
    }
    u.Entries = append(u.Entries, MenuEntry{Type:0, Data:temp})

    if ju.TopScreenContents.URL != "" {
        u.Entries = append(u.Entries, MenuEntry{Type:1, Data:[]string{"1", ju.TopScreenContents.URL}})
    } else {
        u.Entries = append(u.Entries, MenuEntry{Type:1, Data:[]string{"0", q(ju.TopScreenContents.Uppertitle), q(ju.TopScreenContents.Uppersubleft), q(ju.TopScreenContents.Uppersubright), q(ju.TopScreenContents.Uppersubtop), q(ju.TopScreenContents.Uppersubbottom) }})
    }

    for _, item := range ju.Items {
        switch item.Type {
        case "dropdown":
            u.Entries = append(u.Entries, MenuEntry{Type:2, Data:[]string{item.URL, q(item.Label), fmt.Sprint(btoi(item.Selected)) }})
        case "corner":
            u.Entries = append(u.Entries, MenuEntry{Type:3, Data:[]string{item.URL, q(item.Label) }})
        case "button":
            u.Entries = append(u.Entries, MenuEntry{Type:4, Data:[]string{item.URL, fmt.Sprint(item.Icon), q(item.Label), fmt.Sprint(item.Count), fmt.Sprint(item.Lock), fmt.Sprint(item.Unknown1), fmt.Sprint(item.Unknown2) }})
        }
    }

    for _, embed := range ju.Embed {
        if bytes, err := os.ReadFile(embed); err != nil {
            errorlog.Printf("%v", err)
        } else {
            u.Embed = append(u.Embed, bytes)
        }
    }

    return u
}

func (u Ugomenu) UgoHandle() http.HandlerFunc {

    fn := func(w http.ResponseWriter, r *http.Request) {
        w.Write(u.Pack(mux.Vars(r)["reg"]))
        return
    }

    return fn
}

// compile ugomenu to array of bytes
func (u Ugomenu) Pack(r string) []byte {

    var header, menus, embedded []byte
    sections := 1 // there is always at least one section
    emb := false

    // an ugomenu must have this section
    for _, item := range u.Entries {
        menus = append(menus, 0x0A)
        menus = append(menus, fmt.Sprint(item.Type)...)

        for _, data := range item.Data {
            menus = append(menus, 0x09)
            menus = append(menus, strings.Replace(data, "v2-xx", r, 1)...)
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
    header = []byte(magic)
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
