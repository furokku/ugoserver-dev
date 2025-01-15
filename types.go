package main

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type (
    
    // Internal types, don't need to be exported
    session struct {
        mac      string
        fsid     string
        auth     string // xor stuff
        ver      int // ugomemo version 0:rev1(jp release) 1:rev2(jp update 1) 2:rev3(us/eu release, jp update 2)
        username string
        region   int // 0:jp 1:us 2:eu
        lang     string
        country  string
        birthday string
        datetime string // ds supplied date/time
        color    string

        ip     string
        issued time.Time

        is_unregistered bool
        is_logged_in bool
        userid int
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


    // Json config
    Configuration struct {
        Listen   string `json:"listen"`
        Root     string `json:"root"`
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


    // Must be exported for html templates
    Movie struct {
        ID int
        Au_userid int // author info
        Au_fsid string
        Au_name string
        Au_fn string
        Posted time.Time
        Lock bool // This isn't really used
        Views int
        Downloads int
        Deleted bool
        Channelid int
        Ys int // stars
        Gs int
        Rs int
        Bs int
        Ps int
    }

    Comment struct {
        ID int
        Userid int
        Is_memo bool
        Content string
        Posted time.Time
    }


    // Json ugomenus
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

    

    // html template stuff
    Page struct {
        Root string // server domain to use everywhere
        Title string
        Region string
        LoggedIn bool
    }
    
    CommentPage struct {
        Page
        Comments []Comment
        CommentCount int
        MovieID int
    }
    
    MoviePage struct {
        Page
        Movie
        MovieAuthor bool
        CommentCount int
    }
)