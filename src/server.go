package main

import (
    _ "github.com/lib/pq"
    "database/sql"

    "log"

    "github.com/gorilla/mux"
    "net/http"

    "context"
    "os"
    "os/signal"
    "time"
)

func main() {

    // prep graceful exit
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, os.Kill, os.Interrupt)

    // connect to database
    db, err := sql.Open("postgres", "postgresql://furokku:passwd@localhost/ugo?sslmode=disable")
    if err != nil {
        log.Fatalf("failed to open database (sql.Open): %v", err)
    } else if err := db.Ping(); err != nil {
        log.Fatalf("failed to reach database (sql.Ping): %v", err)
    }

    defer db.Close()

    // start a thread to remove old, expired sessions
    // the time for a session to expire is 2 hours
    // may increase later if needed
    go pruneSids()

    // start the hatena auth/general http server
    //
    // ~~in future this may run on the main goroutine as
    // nas is not explicitly required thanks to wiimmfi
    // and such~~
    // will implement signal handling later so this should
    // still be a separate thread
    log.Println("starting server...")

    // gorilla/mux allows accepting requests for
    // a range of urls, then filtering them as needed
    h := mux.NewRouter() // hatena

    // http's servemux works fine for this
    n := http.NewServeMux() // nas

    // didn't produce desired results, using
    // basic "read file, return it if it exists" handler
//  fs := http.FileServer(http.Dir("./static"))
//  h.Handle("/", denyIndex(fs))

    // v1
    // likely not going to support this because it's just
    // going to be a giant hassle
//  m.Path("/ds/{sub}/{ugo}.ugo").HandlerFunc(ugoHandler)
//  m.Path("/ds/{ugo}.ugo").HandlerFunc(ugoHandler)
//  m.Path("/ds/auth").HandlerFunc(hatenaAuthHandler)

    // TODO: tv-jp
    // v2-us, v2-eu, v2-jp, v2 auth
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/auth").Methods("GET", "POST").HandlerFunc(hatenaAuth)

    // eula
    h.Path("/ds/{reg}/{lang}/{file}.txt").Methods("GET").HandlerFunc(eulaHandler)
    h.Path("/ds/{reg}/{lang}/confirm/{file}.txt").Methods("GET").HandlerFunc(eulaHandler)

    h.Path("/ds/{reg}/{file}.txt").Methods("GET").HandlerFunc(eulaHandler) // v2
    h.Path("/ds/{reg}/confirm/{file}.txt").Methods("GET").HandlerFunc(eulaHandler) // v2

    h.Path("/ds/{reg}/{ugo}.ugo").Methods("GET").HandlerFunc(ugoHandler)
    h.Path("/ds/{reg}/{file}.htm").Methods("GET").HandlerFunc(returnFromFs)

    // return a built ugo file with flipnotes
    h.Path("/front/{type:(?:recent|hot|liked)}.ugo").Methods("GET").HandlerFunc(serveFrontPage(db))

    // stuff
    h.Path("/flipnotes/{filename}.ppm").Methods("GET").HandlerFunc(returnFromFs)
    h.Path("/flipnotes/{filename}.htm").Methods("GET").HandlerFunc(logRequest)
    h.Path("/flipnotes/{filename}.info").Methods("GET").HandlerFunc(infoHandler)

    n.HandleFunc("/", nasAuth)

    // define servers
    nas := &http.Server{Addr: "9001", Handler: n}
    hatena := &http.Server{Addr: ":9000", Handler: h}

    // start on separate thread
    go func() {
        err := hatena.ListenAndServe()
        if err != http.ErrServerClosed {
            log.Fatalf("server error: %v", err)
        }
    }()

    // need to choose whether to use own nas auth or to
    // use wiimmfi/kaeru nas
    // wiimmfi seems to be kinda weird and unstable because
    // it returns a 404 on /ac or /pr randomly
    if enableNas {

        log.Println("(nas) starting server...")

        // start on separate thread
        go func() {
            err := nas.ListenAndServe()
            if err != http.ErrServerClosed {
                log.Fatalf("(nas) server error: %v", err)
            }
        }()
    } else {
        log.Println("(nas) enableNas set to false, not hosting")
    }


    // wait and do a graceful exit on ctrl-c / sigterm
    sig := <- sigs
    log.Printf("%v: exiting...\n", sig)

    ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
    defer cancel()

    if err := hatena.Shutdown(ctx); err != nil {
        log.Fatalf("graceful shutdown failed! %v", err)
    }

    if enableNas { if err := nas.Shutdown(ctx); err != nil {
        log.Fatalf("(nas) graceful shutdown failed! %v", err)
    } }

    log.Println("server shutdown")
}
