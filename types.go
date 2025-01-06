package main

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// should be self-explanatory
type (
    session struct {
        mac      string
        fsid     string
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

        ip     string // current ip
        issued time.Time

        is_unregistered bool
        is_logged_in bool
        userid int
    }
    
    Configuration struct {
        Listen   string `json:"listen"`
        URL      string `json:"url"`
        Dir      string `json:"dir"`
        StoreDir string `json:"store_dir"`

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

    flipnote struct {
        id int
        author_userid int
        author_fsid string
        author_name string
        author_filename string
        uploaded time.Time
        lock bool
        views int
        downloads int
        deleted bool
        channelid int
        ys int // stars
        gs int
        rs int
        bs int
        ps int
    }

    restriction struct {
        banid    int
        issuer   string
        issued   time.Time
        expires  time.Time
        message  string
        pardon   bool
        affected string
    }

    tmb []byte

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
        Items      []MenuItem `json:"items"`
        Embed      []string `json:"embed"`
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

    rwWrapper struct {
        http.ResponseWriter
        status int
        done   bool
    }
)