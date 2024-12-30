package main

import (
	"net"
	"sync"
	"time"
)

// should be self-explanatory
type (
    AuthPostRequest struct {
        mac      string
        id       string
        ip       string
        auth     string
        sid      string
        ver      string
        username string
        region   string
        lang     string
        country  string
        birthday string
        datetime string
        color    string
    }

    Configuration struct {
        Listen    string `json:"listen"`
        URL string `json:"url"`
        Dir string `json:"dir"`

        DB struct {
            Type    string `json:"type"`
            File    string `json:"file"`
            Host    string `json:"host"`
            Port    int    `json:"port"`
            User    string `json:"user"`
            Pass    string `json:"pass"`
            Name    string `json:"name"`
        } `json:"db"`
        UseHosts  bool     `json:"use_allowed_hosts"`
        Hosts     []string `json:"hosts"`
    }

    session struct {
        fsid string
        username string
        issued time.Time
        ip string
        s2r AuthPostRequest 
    }

    flipnote struct {
        id int
        author_id string
        author_name string
        parent_author_id string
        parent_author_name string
        author_filename string
        uploaded_at time.Time
        lock bool
        views int
        downloads int
        stars map[string]int
        deleted bool
    }

    restriction struct {
        id int
        issuer string
        issued time.Time
        expires time.Time
        reason string
        message string
        pardon bool
        affected string
    }

    tmb []byte

    UgomenuDepr struct {
        Layout []uint
        Entries []MenuEntry
        Embed [][]byte
    }

    MenuEntry struct {
        Type uint
        Data []string
    }

    Ugomenu struct {
        Layout []int `json:"layout"`
        TopScreenContents struct {
            URL            string `json:"url,omitempty"`
            Uppertitle     string `json:"uppertitle,omitempty"`
            Uppersubleft   string `json:"uppersubleft,omitempty"`
            Uppersubright  string `json:"uppersubright,omitempty"`
            Uppersubtop    string `json:"uppersubtop,omitempty"`
            Uppersubbottom string `json:"uppersubbottom,omitempty"`
        } `json:"top"`
        Items []MenuItem `json:"items"`
        Embed []string `json:"embed"`
        EmbedBytes [][]byte
    }

    MenuItem struct {
        Type     string `json:"type"`
        URL      string `json:"url"`
        Label    string `json:"label"`
        Selected bool `json:"selected,omitempty"`
        Icon     int `json:"icon,omitempty"`
        Count    int `json:"count,omitempty"`
        Lock     int `json:"lock,omitempty"`
        Unknown1 int `json:"unknown1,omitempty"`
        Unknown2 int `json:"unknown2,omitempty"`
    }

    ipcListener struct {
        listener net.Listener
        quit     chan interface{}
        wg       sync.WaitGroup
    }
)
