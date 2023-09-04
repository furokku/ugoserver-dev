package main

import (
    "encoding/base64"
    "database/sql"
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

// an anonymous struct is better for this
//type fsidSession struct {
//    fsid   string
//    issued int64
//}

var (
    // database stuff
    sessions = make(map[string]struct{fsid string; issued int64})
    db *sql.DB

    chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")
    regions = []string{"v2-us", "v2-eu", "v2-jp", "v2"} // TODO: tv-jp

    // for readability in ugo.go
    newline byte = 0x0A
    tab     byte = 0x09
    nul     byte = 0x00

    // template/static ugomenus
    indexUGO = ugomenu{
        entries: []menuEntry{
            {
                entryType: 0,
                data: []string{
                    "0",
                },
            }, {
                entryType: 4,
                data: []string{
                    "http://flipnote.hatena.com/front/recent.uls",
                    "103",
                    base64.StdEncoding.EncodeToString(encUTF16LE("Browse flipnotes")),
                },
            },
            {
                entryType: 4,
                data: []string{
                    "http://flipnote.hatena.com/ds/v2-us/test.uls",
                    "104",
                    base64.StdEncoding.EncodeToString(encUTF16LE("test(ing)")),
                },
            },
        },
    }

    fpBase = ugomenu{
        entries: []menuEntry{
            {
                entryType: 0,
                data: []string{
                    "2",
                },
            },
        },
    }
)
