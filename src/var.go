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
    enableNas = true
    txtPath = "/srv/hatena/txt/"
    dataPath = "/srv/hatena/hatena_storage/"
)

var (
    // database stuff
    sessions = make(map[string]struct{fsid string; username string; issued int64})

    // things
    regions = []string{"v2-us", "v2-eu", "v2-jp", "v2"} // TODO: tv-jp
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
                    "http://flipnote.hatena.com/front/recent.uls",
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
