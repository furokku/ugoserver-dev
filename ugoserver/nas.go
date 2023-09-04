package main

import (
    "fmt"
    "io"
    "net/http"
    "net/url"
    "log"
    "errors"
    "time"
)

func runNasServer() {

    // these are set up because http.DefaultServeMux()
    // would have two handlers for "/" assigned at once
    // and cause an error
    nasMux := http.NewServeMux()
    nasMux.HandleFunc("/", nasAuthHandler)

    // server is multithreaded
    go func() {
        err := http.ListenAndServe(":9001", nasMux)
        if errors.Is(err, http.ErrServerClosed) {
            log.Println("nas server closed")
        } else if err != nil {
            fmt.Printf("error: %v\n", err)
        }
    }()
    log.Println("nas server up")
}

func nasAuthHandler(w http.ResponseWriter, r *http.Request) {

    // deny requests other than POST
    if r.Method != "POST" {
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    body, _ := io.ReadAll(r.Body)
    nasRequest, err := url.ParseQuery(string(body))
    if err != nil {
        log.Fatal("error parsing urlencoded form")
    }

    // decode base64 values to plaintext for logging reasons
    // and to check action key
    for key := range nasRequest {
        // only one value is set per key so this is fine
        nasRequest[key][0] = decode(nasRequest[key][0])
    }

    log.Printf("received request to %v%v with data %v\n%v\n", r.Host, r.URL.Path, string(body), r.Header)
    log.Printf("%v\n\n", nasRequest)

    action := nasRequest.Get("action")
    resp := make(url.Values)

    switch r.URL.Path {
    case "/ac":
        switch action {

        // known action values are login, acctcreate and svcloc
        // those can be handled later
        case "login":
            resp.Set("challenge", encode(randAsciiString(8)))
            resp.Set("locator", encode("gamespy.com"))
            resp.Set("retry", encode("0"))
            resp.Set("returncd", encode("001"))
            resp.Set("token", encode(append([]byte("NDS"), randBytes(96)...)))

        default:
            http.Error(w, "invalid request", http.StatusBadRequest)
            return
        }

    // nintendo profanity filter thing
    case "/pr":
        resp.Set("prwords", encode("0"))
        resp.Set("returncd", encode("000"))

    default:
        http.Error(w, "invalid request", http.StatusNotFound)
        return
    }

    // datetime will be sent regardless
    resp.Set("datetime", encode(time.Now().Format("20060102150405")))
    w.Write([]byte(resp.Encode()))
}
