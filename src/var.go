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
        EnableNas bool `json:"enableNas"`

        Listen    string `json:"listen"`
        ServerUrl string `json:"serverUrl"`

        HatenaDir string `json:"hatenaDir"`

        DbHost    string `json:"dbHost"`
        DbPort    int    `json:"dbPort"`
        DbUser    string `json:"dbUser"`
        DbPass    string `json:"dbPass"`
        DbName    string `json:"dbName"`
    }
)

// added better config
// const (
    // enable the NAS functionality in the server
    // recommended to use wiimmfi
    // enableNas = true

    // sane default paths for commonly accessed things
    // staticPath = "/srv/hatena/static"
    // dataPath = "/srv/hatena/hatena_storage"
    // serverUrl = "http://flipnote.hatena.com"

    // ip to allow connections from
    // by default set to allow every connection,
    // regardless of origin
    // listen = "0.0.0.0"
// )

var (
    // Map to store flipnote sessions in
    sessions = make(map[string]struct{fsid string; username string; issued time.Time})

    // only the regions that are listed can be accessed thru /ds/xxxxx/foo/bar
    // not necessary because i realized that half of these things
    // can just be done with a simple regex statement while assigning handlers
    // regions = []string{"v2-us", "v2-eu", "v2-jp", "v2"} // TODO: tv-jp (?)
    // txts := []string{"delete", "download", "eula", "upload"}

    // template/static ugomenus
    indexUGO = ugo.Ugomenu{}
    gridBaseUGO = ugo.Ugomenu{}
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
