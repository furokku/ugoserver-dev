package main

import (
	"encoding/binary"
	"fmt"
	"net/http"
	"os"
	"strings"
)

const MENU_MAGIC string = "UGAR"

// menubc() (menu byte convert) is a helper to convert arguments into a menu entry
func menubc(t int, a ...any) []byte {
    var entry []byte
    if t != 0 {
        entry = append(entry, 0x0A)
    }
    entry = append(entry, fmt.Sprint(t)...)
    for _, i := range a {
        entry = append(entry, 0x09)
        entry = append(entry, fmt.Sprint(i)...)
    }

    return entry
}

// newMenu() returns a fresh pointer to ugomenu
func newMenu() *Ugomenu {
    u := Ugomenu{}

    return &u
}

// setLayout() sets menu layout
// not sure what exactly the second numbers always mean, ie:
// index.ugo / big top button, rest small | 0;
// list of small buttons | 1;
// inbox.ugo / feed / 6 sq buttons side by side | 2 (1);
// channels.ugo | 3 4
func (u *Ugomenu) setLayout(types ...int) {
    u.Layout = types
}

// setTopScreenURL() sets the URL to an image(nbf) or animation(ppm) for the top screen
// If this is set, other options are ignored
func (u *Ugomenu) setTopScreenURL(url string) {
    u.TopScreenContents.URL = url
}
// setTopScreenText() sets values for text on the top screen
func (u *Ugomenu) setTopScreenText(title string, left string, right string, top string, bottom string) {
    u.TopScreenContents.Uppertitle = title
    u.TopScreenContents.Uppersubleft = left
    u.TopScreenContents.Uppersubright = right
    u.TopScreenContents.Uppersubtop = top
    u.TopScreenContents.Uppersubbottom = bottom
}

// addDropdown() adds a dropdown button at the top of the bottom screen
func (u *Ugomenu) addDropdown(url string, label string, selected bool) {
    u.Items = append(u.Items, MenuItem{Type:"dropdown", URL:url, Label:label, Selected:selected})
}

// addCorner() adds a corner button,
// there can be at most two on any one menu
func (u *Ugomenu) addCorner(url string, label string) {
    u.Items = append(u.Items, MenuItem{Type:"corner", URL:url, Label:label})
}

// addButton() adds a button to the menu
// unlimited number, for flipnote previews must have tmb embed
// icon index: index.ugo: 100 people, 101 tv, 102 globe, 103 search, 104 frog
// type 0,1: 101 tv, 104 frog, 113 6frog, 114 person, 115 rarrow, 116 larrow, 117 question
// type 4: 101 tv, 104 frog, 105 send, 106 heartspeech, 107 addstar, 108 frogspeech, 109 tvspeech
// (cont.) 110 diary, 111 profile, 112 4frog, 113 6frog
// type 2,tmb: 0 letterdefault, 1 readtext, 2 letterdefault, 3 thumbstar, 4 readtext
// type 2,url: 0 letterstamp, 1 readnostamp, 2 letterstamp, 3 letternostamp
// example feed.AddButton("http://foo/bar/baz.ppm", 3, "label", 69, 765, 573 (u1), 0 (u2))
// lock can be set to 765 to get from tmb
// unknown1 is usually 573, unsure what it does
// unknown2 is usually 0, unsure
func (u *Ugomenu) addButton(url string, icon int, label string, extra ...int) {
    if len(extra) == 4 {
        // non-mandatory attributes
        u.Items = append(u.Items, MenuItem{Type:"button", URL:url, Icon:icon, Label:label, Count:extra[0], Lock:extra[1], Unknown1:extra[2], Unknown2:extra[3]})
    } else {
        u.Items = append(u.Items, MenuItem{Type:"button", URL:url, Icon:icon, Label:label})
    }
}

func (u *Ugomenu) addMenuItem(m MenuItem) {
    u.Items = append(u.Items, m)
}

// addTest() is for testing menu items with types >4
func (u *Ugomenu) addTest(t int, v ...string) {
    u.Items = append(u.Items, MenuItem{Type:"test", TestType:t, TestValues:v})    
}

// addEmbed() takes in a slice of bytes to be added to the menu
func (u *Ugomenu) addEmbed(e []byte) {
    u.EmbedBytes = append(u.EmbedBytes, e)
}

// pack() builds the menu, converting the struct into something the DS can parse;
// "flipnote.hatena.com" and "v2-xx" will be replaced with the Root url and the correct region
// and labels will be converted automatically
func (e *env) pack(u Ugomenu, r string) []byte {

    var header, menus, embedded []byte
    sections := 1 // there is at least 1 section
    emb := false

    // workaround
    l := []any{}
    for _, i := range u.Layout {
        l = append(l, i)
    }
    menus = append(menus, menubc(0, l...)...)

    if u.TopScreenContents.URL != "" {
        menus = append(menus, menubc(1, 1, u.TopScreenContents.URL)...)
    } else {
        menus = append(menus, menubc(1, 0, q(u.TopScreenContents.Uppertitle), q(u.TopScreenContents.Uppersubleft), q(u.TopScreenContents.Uppersubright), q(u.TopScreenContents.Uppersubtop), q(u.TopScreenContents.Uppersubbottom))...)
    }

    for _, item := range u.Items {
        url := strings.Replace(item.URL, "flipnote.hatena.com", e.cnf.Root, 1)
        url = strings.Replace(url, "xx", r, 1)
        switch item.Type {
        case "dropdown":
            menus = append(menus, menubc(2, url, q(item.Label), btoi(item.Selected))...)
        case "corner":
            menus = append(menus, menubc(3, url, q(item.Label))...)
        case "button":
            menus = append(menus, menubc(4, url, item.Icon, q(item.Label), item.Count, item.Lock, item.Unknown1, item.Unknown2)...)
        case "test":
            // workaround
            v := []any{}
            for _, i := range item.TestValues {
                v = append(v, i)
            }
            menus = append(menus, menubc(item.TestType, v...)...)
        }
    }

    menus = append(menus, pad(len(menus))...)

    if len(u.Embed) != 0 || len(u.EmbedBytes) != 0 {
        for _, embed := range u.Embed {
            bytes, err := os.ReadFile(embed)
            if err != nil {
                errorlog.Printf("embedding %v failed: %v", embed, err)
            }
            embedded = append(embedded, bytes...)
        }
        for _, embed := range u.EmbedBytes {
            embedded = append(embedded, embed...)
        }

        emb = true
        sections = 2
        embedded = append(embedded, pad(len(embedded))...)
    }

    header = []byte(MENU_MAGIC)
    header = binary.LittleEndian.AppendUint32(header, uint32(sections))
    header = binary.LittleEndian.AppendUint32(header, uint32(len(menus)))
    if emb { header = binary.LittleEndian.AppendUint32(header, uint32(len(embedded))) }

    return append(header, append(menus, embedded...)...)
}

// handle() method will return an http.HandlerFunc which responds with a packed menu
func (e *env) handle(u Ugomenu) http.HandlerFunc {

    fn := func(w http.ResponseWriter, r *http.Request) {
        w.Write(e.pack(u, e.sessions[r.Header.Get("X-Dsi-Sid")].Region))
    }

    return fn
}

// pad() returns enough null bytes to pad a slice so that its length is divisible by 4
func pad(l int) []byte {
    var pad []byte

    if x := l % 4; x != 0 {
        for i := 0; i < (4-x); i++ {
            pad = append(pad, 0x00)
        }
    }
    return pad
}
