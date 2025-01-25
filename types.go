package main

import (
	"net"
	"net/http"
	"sync"
	"time"
)

type (
    
    // Internal types, don't need to be exported (yet?)

    // restriction contains all information about a user's ban
    restriction struct {
        banid    int
        issuer   string // Who banned the user
        issued   time.Time
        expires  time.Time
        message  string
        pardon   bool // whether the ban was pardoned
        affected string // affected IP or FSID
    }

    // Unix ipc listener
    ipcListener struct {
        listener net.Listener
        quit     chan interface{}
        wg       sync.WaitGroup
    }

    // Wrap responsewriter in order to log http requests and reponses
    rwWrapper struct {
        http.ResponseWriter
        status int
        done   bool
    }

    // Json config format
    Configuration struct {
        Listen   string `json:"listen"`
        Root     string `json:"root"`
        Dir      string `json:"dir"`
        StoreDir string `json:"store_dir"`

        DB struct {
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
    
    // Session contains all information about a user's session, including data
    // from the initial authentication on /ds/v2-xx/auth
    Session struct {
        MAC      string // console MAC address
        FSID     string
        Auth     string // Auth Challenge Response, it can be checked, but i haven't figured it out yet
        Ver      int // ugomemo version 0:rev1(jp release) 2:rev2(jp update 1) and rev3(us/eu release, jp update 2)
        Username string
        Region   int // 0:jp 1:us 2:eu
        Lang     string // system language, as set in settings
        Country  string // system country, as set in settings
        Birthday string // user birthday as 20060102 date
        DateTime string // ds supplied date/time
        Color    string // system color, as set in settings

        IP     string
        Issued time.Time

        IsUnregistered bool
        IsLoggedIn bool
        UserID int // 0 if unregistered
    }
    
    // Movie contains all information about a movie, including the amount of comments it has
    // When building the feed, only ID and Ys are set, as nothing else is necessary
    // However, when fetching a singular movie by its ID, all fields are populated
    Movie struct {
        ID int
        AuUserID int // author user id
        AuFSID string // author user fsid
        AuName string // author user name
        AuFN string // filename when uploaded
        Posted time.Time
        Lock bool // whether movie is locked, in menus this is unused
        Views int 
        Downloads int
        Deleted bool
        ChannelID int
        Ys int // yellow stars
        Gs int // green
        Rs int // red
        Bs int // blue
        Ps int // purple
        Replies int // number of comments
    }

    // Comment contains all information about a reply made to a movie
    Comment struct {
        ID int
        UserID int
        MovieID int
        Username string
        IsMemo bool // whether the comment is a mini flipnote
        Content string // text, if the comment is a text comment
        Posted time.Time
    }

    // Ugomenu is a data type for parsing statically laid out menus from json, for convenience
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

    // MenuItem is a single item in a menu
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

    // html template data type, only the fields that are needed in a particular template should be set
    Page struct {
        Session
        Root string
        Region string
        
        Movie
        Comments []Comment
        
        SID string

        Return string
    }
)