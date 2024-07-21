package main

import (
    "time"
)

// should be self-explanatory
type (
    AuthPostRequest struct {
        mac      string
        id       string
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
        ServerUrl string `json:"serverUrl"`

        HatenaDir string `json:"hatenaDir"`

        DbHost    string `json:"dbHost"`
        DbPort    int    `json:"dbPort"`
        DbUser    string `json:"dbUser"`
        DbPass    string `json:"dbPass"`
        DbName    string `json:"dbName"`
    }

    session struct {
        fsid string
        username string
        issued time.Time
        ip string
    }
)

var (
    // Map to store flipnote sessions in
    sessions = make(map[string]session)

    // template/static ugomenus
    indexUGO = Ugomenu{}
    gridBaseUGO = Ugomenu{}

    prettyPageTypes = map[string]string{"recent":"Recent"}

    loadedUgos = make(map[string]Ugomenu)
)

// keep this here so that server.go doesn't get too messy
func setBaseUGO(ugo *Ugomenu, t Ugomenu) {
    *ugo = t
}

func ugoworkaroundinit() {

    setBaseUGO(&indexUGO, Ugomenu{
        Entries: []MenuEntry{
            {
                Type: 0,
                Data: []string{
                    "0",
                },
            },
            {
                Type: 4,
                Data: []string{
                    configuration.ServerUrl + "/ds/v2-xx/feed.uls?mode=recent&page=1",
                    "100",
                    q("Browse Flipnotes"),
                },
            },
            {
                Type: 4,
                Data: []string{
                    "ugomemo://postmemo",
                    "102",
                    q("Post a Flipnote"),
                },
            },
        },
    })

    setBaseUGO(&gridBaseUGO, Ugomenu{
        Entries: []MenuEntry{
            {
                Type: 0,
                Data: []string{
                    "2",
                },
            },
        },
    })
}
