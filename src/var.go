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

    prettyPageTypes = map[string]string{"recent":"Recent"}

    loadedUgos = make(map[string]Ugomenu)
)
