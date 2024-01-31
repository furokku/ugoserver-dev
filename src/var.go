package main

import (
    "encoding/base64"
    "floc/ugoserver/ugo"

    "time"
)

// should be self-explanatory
type (
    authPostRequest struct {
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

)

const (
    // enable the NAS functionality in the server
    // recommended to use wiimmfi
    enableNas = true

    // sane default paths for commonly accessed things
    staticPath = "/srv/hatena/static"
    dataPath = "/srv/hatena/hatena_storage"
    serverUrl = "http://flipnote.hatena.com"

    // ip to allow connections from
    // by default set to allow every connection,
    // regardless of origin
    listen = "0.0.0.0"
)

var (
    // Map to store flipnote sessions in
    sessions = make(map[string]struct{fsid string; username string; issued time.Time})

    // only the regions that are listed can be accessed thru /ds/xxxxx/foo/bar
    // not necessary because i realized that half of these things
    // can just be done with a simple regex statement while assigning handlers
    // regions = []string{"v2-us", "v2-eu", "v2-jp", "v2"} // TODO: tv-jp (?)
    // txts := []string{"delete", "download", "eula", "upload"}

    // template/static ugomenus
    indexUGO = ugo.Ugomenu{
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
                    serverUrl + "/front/recent.uls",
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
    }

    frontBaseUGO = ugo.Ugomenu{
        Entries: []ugo.MenuEntry{
            {
                EntryType: 0,
                Data: []string{
                    "2",
                },
            },
        },
    }
)
