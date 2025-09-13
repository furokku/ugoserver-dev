package main

import (
	"context"
	"html/template"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type (
    
    // Internal types, don't need to be exported (yet?)

    // Unix ipc listener
    ipcListener struct {
        listener net.Listener
        quit     chan interface{}
        wg       sync.WaitGroup
    }
    
    cmdHandlerFunc func([]string) string // Handler for IPC commands
    cmdHandler map[string]cmdHandlerFunc

    // Wrap responsewriter in order to log http requests and reponses
    rwWrapper struct {
        http.ResponseWriter
        status int
        done   bool
    }

    dbhandle interface {
        Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
        QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
        Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
    }
    
    env struct {
        sessions map[string]*Session
        pool *pgxpool.Pool // use when transactions aren't necessary
        cnf *Configuration // from config.json containing global options
        
        html *template.Template // html templates
        assets map[string][]byte // 
        menus map[string]Ugomenu
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
    
    // None of these need to be exported anymore
    
    // Session contains all information about a user's session, including data
    // from the initial authentication on /ds/v2-xx/auth
    Session struct {
        MAC      string // console MAC address
        FSID     string
        Auth     string // Auth Challenge Response, it can be checked, but i haven't figured it out yet
        Ver      int // ugomemo version 0:rev1(jp release) 2:rev2(jp update 1) and rev3(us/eu release, jp update 2)
        Username string
        RegionCode int // 0:jp 1:us 2:eu
        Region   string // simpler
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
    
    // User definition for other things
    User struct {
        ID int
        FSID string
        Username string
        Admin bool
        Deleted bool // why is this even here? I can;t bother to remove it
        
        ExpendableStars []int //1: green, 2: blue, etc
        
        LastLoginIP string
        LastLoginTime time.Time
    }
    
    UserPref struct {
        // stub: will contain user preferences for DS or web ui
    }
    
    // Movie contains all information about a movie, including the amount of comments it has
    // When building the feed, only ID and Ys are set, as nothing else is necessary
    // However, when fetching a singular movie by its ID, all fields are populated
    Movie struct {
        ID int
        ChannelID int // allow multiple channels?

        AuUserID int // author user id
        AuFSID string // author fsid
        AuName string // author username
        AuFN string // filename when uploaded

        OGAuFSID string
        OGAuName string
        OGAuFNFrag string // Fragment of original flipnote (useful?)

        Posted time.Time
        LastMod time.Time // when flipnote was last edited I think
        Deleted bool

        Lock bool // whether movie is locked, use value of 765 to have flipnote fill it in automatically
        Views int 
        Downloads int
        Replies int // # of comments

        Stars []int // 0: yellow, 1: green, etc
        JumpCode string
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
        Deleted bool
    }

    Channel struct {
        ID int
        Name string
        Description string
        Deleted bool
    }

    // Ban contains all information about a user's ban
    Ban struct {
        ID    int
        Issuer   string // Who banned the user
        Issued   time.Time
        Expires  time.Time
        Message  string
        Pardon   bool // whether the ban was pardoned
        Affected string // affected IP or FSID
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

    // MenuItem is a single item in a menu.
    // Supported Types are corner, dropdown, button and test.
    // User-facing text elements are automatically encoded when packed;
    // in original assets Unknown1 has always been set to 573 and Unknown2 to 0 on buttons
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
        TestType int `json:"test_type,omitempty"`
        TestValues []string `json:"test_values,omitempty"`
    }
    
    // html template data type, only the fields that are needed in a particular template should be set
    // realized this is horrible
    // use a map with an interface{} value instead
    // DSPage struct {
    //     Session
    //     Root string
    //     Region string
    //     User
    //     
    //     Movie
    //     Comments []Comment
    //     Channel
    //     
    //     // redirect after login for better ux
    //     Redirect string
    //     // whether the page should display anything extra
    //     // if you are pushed back to it;
    //     // example: [..]/ds/v2-eu/sa/register.htm?ret=error
    //     Return string
    // }
    // 
    // WebPage struct {
    //     User
    //     UserPref
    //     
    //     Movie
    //     Movies []Movie
    //     Comments []Comment
    //     Channel
    //     Channels []Channel

    //     Return string
    // }
)