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


func ugoNew() *Ugomenu {
    u := Ugomenu{}

    return &u
}

// set ugomenu layout
// not sure what exactly the second numbers always mean
// examples:
// index.ugo, big top button, rest small | 0
// small top button, rest small | 1
// inbox.ugo, feed, 6 sq buttons sidebyside | 2 (1)
// channels.ugo | 3 4
func (u *Ugomenu) setType(types... int) {
    t := []string{}
    for i := range types {
        t = append(t, fmt.Sprint(i))
    }
    u.Entries = append(u.Entries, MenuEntry{Type:0, Data:t})
}

// adds a dropdown button at the top
// applicable to layout 2, others not sure
func (u *Ugomenu) addDropdown(url string, label string, selected bool) {
    u.Entries = append(u.Entries, MenuEntry{Type:2, Data:[]string{url, q(label), fmt.Sprint(btoi(selected)) }})
}

// adds a corner button
// there can be at most two on any
// one ugomenu
func (u *Ugomenu) addCorner(url string, label string) {
    u.Entries = append(u.Entries, MenuEntry{Type:3, Data:[]string{url, q(label) }})
}

// add a button
// unlimited number, for flipnote previews must have tmb embed
// icon index: index.ugo: 100 people, 101 tv, 102 globe, 103 search, 104 frog
// type 0,1: 101 tv, 104 frog, 113 6frog, 114 person, 115 rarrow, 116 larrow, 117 question
// type 4: 101 tv, 104 frog, 105 send, 106 heartspeech, 107 addstar, 108 frogspeech, 109 tvspeech
// (cont.) 110 diary, 111 profile, 112 4frog, 113 6frog
// type 2,tmb: 0 letterdefault, 1 readtext, 2 letterdefault, 3 thumbstar, 4 readtext
// type 2,url: 0 letterstamp, 1 readnostamp, 2 letterstamp, 3 letternostamp
// example feed.AddButton("http://foo/bar/baz.ppm", 3, "label", count, lock (765), 573 (u1), 0 (u2))
// lock can be set to 765 to get from tmb
// unknown1 is usually 573, unsure what it does
// unknown2 is usually 0, unsure
func (u *Ugomenu) addButton(url string, icon int, label string, extra... int) {
    e := []string{}
    for i := range extra {
        e = append(e, fmt.Sprint(i))
    }
    u.Entries = append(u.Entries, MenuEntry{Type:4, Data:append([]string{url, fmt.Sprint(icon), q(label)}, e...) })
}

func (ju JsonUgo) parse() Ugomenu {
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
            errorlog.Fatalf("%v", err)
        } else {
            u.Embed = append(u.Embed, bytes)
        }
    }

    return u
}

func (u Ugomenu) ugoHandle() http.HandlerFunc {

    fn := func(w http.ResponseWriter, r *http.Request) {
        w.Write(u.pack(mux.Vars(r)["reg"]))
        return
    }

    return fn
}

// compile ugomenu to array of bytes
func (u Ugomenu) pack(r string) []byte {

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
    var padded = d

    if x := len(d) % 4; x != 0 {
        for i := 0; i < (4-x); i++ {
            padded = append(padded, 0x00)
        }
    }
    return padded
}
