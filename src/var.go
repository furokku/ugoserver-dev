package main

import (
    "encoding/base64"
    "floc/ugoserver/ugo"

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
    indexUGO = ugo.Ugomenu{}
    gridBaseUGO = ugo.Ugomenu{}

    prettyPageTypes = map[string]string{"recent":"Recent"}
)

// keep this here so that server.go doesn't get too messy
func setBaseUGO(ugo *ugo.Ugomenu, t ugo.Ugomenu) {
    *ugo = t
}

func ugoworkaroundinit() {

    setBaseUGO(&indexUGO, ugo.Ugomenu{
        Entries: []ugo.MenuEntry{
            {
                EntryType: 0,
                Data: []string{
                    "0",
                },
            },
            {
                EntryType: 4,
                Data: []string{
                    configuration.ServerUrl + "/front/recent.uls",
                    "100",
                    base64.StdEncoding.EncodeToString(encUTF16LE("Browse Flipnotes")),
                },
            },
            {
                EntryType: 4,
                Data: []string{
                    "ugomemo://postmemo",
                    "102",
                    base64.RawStdEncoding.EncodeToString(encUTF16LE("Post a Flipnote")),
                },
            },
        },
    })

    setBaseUGO(&gridBaseUGO, ugo.Ugomenu{
        Entries: []ugo.MenuEntry{
            {
                EntryType: 0,
                Data: []string{
                    "2",
                },
            },
        },
    })
}
