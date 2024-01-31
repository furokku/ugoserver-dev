package main

import (
    _ "github.com/lib/pq"
    "database/sql"

    "fmt"

    "github.com/gorilla/mux"
    "net/http"

    "context"
    "os"
    "os/signal"
    "time"
)

var db *sql.DB

func main() {

    // prep graceful exit
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, os.Kill, os.Interrupt)

    dbCfg := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
                         "localhost", 5432, os.Getenv("DBUSER"), os.Getenv("DBPASS"), "ugo")

    // make it shut up
    var err error

    // connect to database
    db, err = sql.Open("postgres", dbCfg)
    if err != nil {
        errorlog.Fatalf("failed to open database: %v", err)
    } else if err := db.Ping(); err != nil {
        errorlog.Fatalf("failed to reach database: %v", err)
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
    infolog.Println("starting server...")

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
    // add regex here instead of an if statement in the function
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/{lang}/{txt:(?:eula)}.txt").Methods("GET").HandlerFunc(handleEula)
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/{lang}/confirm/{txt:(?:delete|download|upload)}.txt").Methods("GET").HandlerFunc(handleEula)

    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/{txt:(?:eula)}.txt").Methods("GET").HandlerFunc(handleEula) // v2
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/confirm/{txt:(?:delete|download|upload)}.txt").Methods("GET").HandlerFunc(handleEula) // v2

    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/{ugo:(?:index)}.ugo").Methods("GET").HandlerFunc(indexUGO.UgoHandle())
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/{file}.htm").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request){w.WriteHeader(http.StatusNotImplemented);return})

    // return a built ugo file with flipnotes
    // only implemented recent so far
    h.Path("/front/{type:(?:recent|liked|random)}.ugo").Methods("GET").HandlerFunc(serveFrontPage)

    // uploading
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/flipnote.post").Methods("POST").HandlerFunc(postFlipnote)
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/movie/{id:[0-9A-Z]{1}[0-9A-F]{5}_[0-9A-F]{13}_[0-9]{3}}.ppm").Methods("POST").HandlerFunc(postFlipnote)

    // related to fetching flipnotes
    // may or may not survive next update
    h.Path("/flipnotes/{id:[0-9A-Z]{1}[0-9A-F]{5}_[0-9A-F]{13}_[0-9]{3}}.{ext:(?:ppm|htm|info)}").Methods("GET").HandlerFunc(serveFlipnotes)

    n.HandleFunc("/", nasAuth)

    // define servers
    nas := &http.Server{Addr: listen + ":9001", Handler: n}
    hatena := &http.Server{Addr: listen + ":9000", Handler: h}

    // start on separate thread
    go func() {
        err := hatena.ListenAndServe()
        if err != http.ErrServerClosed {
            errorlog.Fatalf("server error: %v", err)
        }
    }()

    // need to choose whether to use own nas auth or to
    // use wiimmfi/kaeru nas
    // wiimmfi seems to be kinda weird and unstable because
    // it returns a 404 on /ac or /pr randomly
    if enableNas {

        infolog.Println("(nas) starting server...")

        // start on separate thread
        go func() {
            err := nas.ListenAndServe()
            if err != http.ErrServerClosed {
                errorlog.Fatalf("(nas) server error: %v", err)
            }
        }()
    } else {
        infolog.Println("(nas) enableNas set to false, not hosting")
    }


    // wait and do a graceful exit on ctrl-c / sigterm
    sig := <- sigs
    infolog.Printf("%v: exiting...\n", sig)

    ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
    defer cancel()

    if err := hatena.Shutdown(ctx); err != nil {
        errorlog.Fatalf("graceful shutdown failed! %v", err)
    }

    if enableNas { if err := nas.Shutdown(ctx); err != nil {
        errorlog.Fatalf("(nas) graceful shutdown failed! %v", err)
    } }

    infolog.Println("server shutdown")
}
