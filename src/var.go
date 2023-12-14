package main

import (
    "encoding/base64"
    "floc/ugoserver/ugo"
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
    txtPath = "/srv/hatena/txt/"
    dataPath = "/srv/hatena/hatena_storage/"
    serverUrl = "http://flipnote.hatena.com"

    // ip to allow connections from
    // by default set to allow every connection,
    // regardless of origin
    listen = "0.0.0.0"
)

var (
    // Map to store flipnote sessions in
    sessions = make(map[string]struct{fsid string; username string; issued int64})

    // only the regions that are listed can be accessed thru /ds/xxxxx/foo/bar
    regions = []string{"v2-us", "v2-eu", "v2-jp", "v2"} // TODO: tv-jp (?)

    // eula files that should be returned by handleEula()
    // TODO: remove, add to function directly or to unified config file
    txtFiles = []string{"eula", "upload", "delete", "download"}

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
