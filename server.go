package main

// ugoserver: a flipnote hatena server with bundled image library
// Usage: accepts one argument in the form of a path to a json config file
//        ./ugoserver [/path/to/config]
// database: postgresql, must be compiled with openssl for pgcrypto!
// ipc: thru a unix socket connection. TBD
//
// some guidelines to stick to:
// never trust the user;
// make sure that the client receives as few non-200 responses as possible
// (preferably zero), as this makes flipnote studio behave strangely;
//
// TODO:
// 1. figure out a way to make html templates work
// |_ this one is kinda hard because all pages will need something else
// |_ depending on various factors. For example don't add the star button
// |_ when viewing a flipnote if the user isn't logged in
// |_ Will probably make my own html template function specifically for
// |_ flipnote-related html pages
// 2. clean up manually written urls and replace with ub()
// 3. user-friendly register/login pages etc.

import (
	"database/sql"

	"encoding/json"
	"strings"

	"net/http"

	"github.com/gorilla/mux"

	"context"
	"os"
	"os/signal"
	"time"
)

var (
    db *sql.DB
    cnf = Configuration{}
    sessions = make(map[string]session)
    prettyPageTypes = map[string]string{"recent":"Recent"}
    loadedUgos = make(map[string]Ugomenu)
)

const (
    SOCKET_FILE = "/tmp/ugoserver.sock"
)

func main() {

    // Flags are kinda useless because this will always
    // be used with a configuration file
    cf := "config.json" // default file to look for
    if len(os.Args) > 1 {
        cf = os.Args[1]
    }

    cbytes, err := os.ReadFile(cf)
    if err != nil {
        errorlog.Fatalf("failed to open config file: %v", err)
    }

    json.Unmarshal(cbytes, &cnf)
    if err != nil {
        errorlog.Fatalf("failed to load config file: %v", err)
    }
    infolog.Printf("read config %s", cf)

    // read ugo directory for static/template ugomenus
    var nl int
    ugos, err := os.ReadDir(cnf.Dir + "/ugo")
    if err != nil {
        errorlog.Printf("%v", err)
    }
    for _, ugo := range ugos {
        if ugo.IsDir() { // ignore subdirs
            continue
        }
        name := strings.Split(ugo.Name(), ".")[0]
        bytes, err := os.ReadFile(cnf.Dir + "/ugo/" + ugo.Name())
        if err != nil {
            errorlog.Printf("%v", err)
            continue
        }
        tu := Ugomenu{}
        err = json.Unmarshal(bytes, &tu)
        if err != nil {
            errorlog.Printf("error parsing %s: %v", name, err)
            continue
        }

        loadedUgos[name] = tu
        nl += 1
    }
    infolog.Printf("ugos loaded: %d", nl)

    // prep graceful exit
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, os.Interrupt)


    // connect to db
    db, err = connect()
    if err != nil {
        errorlog.Fatalf("could not connect/reach db: %v", err)
    }
    infolog.Printf("connected to database")

    defer db.Close()
    

    // start unix socket for ipc
    // curious how this works on windows
    os.RemoveAll(SOCKET_FILE)
    ipcS := newIpcListener(SOCKET_FILE)
    infolog.Printf("started unix socket listener")

    defer ipcS.stop()


    // start a thread to remove old, expired sessions
    // the time for a session to expire is 2 hours
    // may increase later if needed
    go pruneSids()

    // hatena auth/general http server
    //
    // gorilla/mux allows accepting requests for
    // a range of urls, then filtering them as needed
    h := mux.NewRouter() // hatena

    // log requests as they come in, eliminates a bunch of redundant code
    h.Use(loggerMiddleware)
    
    // TODO: tv-jp
    // v2-us, v2-eu, v2-jp, v2 auth
    // maybe v1
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/auth").Methods("GET", "POST").HandlerFunc(hatenaAuth)

    // eula
    // add regex here instead of an if statement in the function
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/{lang}/{txt:(?:eula)}.txt").Methods("GET").HandlerFunc(handleEula)
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/{lang}/confirm/{txt:(?:delete|download|upload)}.txt").Methods("GET").HandlerFunc(handleEula)

    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/{txt:(?:eula)}.txt").Methods("GET").HandlerFunc(handleEula) // v2
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/confirm/{txt:(?:delete|download|upload)}.txt").Methods("GET").HandlerFunc(handleEula) // v2
    h.Path("/ds/v2-eu/eula_list.tsv").Methods("GET").HandlerFunc(handleEulaTsv) // europe stuff

    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/index.ugo").Methods("GET").HandlerFunc(dsi_am(false, loadedUgos["index"].ugoHandle()))

    // return a built ugo file with flipnotes
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/feed.ugo").Methods("GET").HandlerFunc(dsi_am(false, serveFrontPage))

    // uploading
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/flipnote.post").Methods("POST").HandlerFunc(dsi_am(true, postFlipnote))

    // related to fetching flipnotes
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/movie/{movieid}.{ext:(?:ppm|htm|info|dl)}").Methods("GET", "POST").HandlerFunc(dsi_am(false, movieHandler))
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/movie/{movieid}.{ext:(?:delete)}").Methods("POST").HandlerFunc(dsi_am(true, movieHandler))
    // stars
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/movie/{movieid}.star").Methods("POST").HandlerFunc(dsi_am(true, starMovieHandler))
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/movie/{movieid}.star/{color:(?:green|red|blue|purple)}").Methods("POST").HandlerFunc(dsi_am(true, starMovieHandler))

    h.Path("/ac").Methods("POST").HandlerFunc(nasAuth)
    h.Path("/pr").Methods("POST").HandlerFunc(nasAuth)
    
    // debug menu for testing features / quick access
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/debug.htm").Methods("GET").HandlerFunc(dsi_am(false, debug))

    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?/sa/register.htm}").Methods("GET").HandlerFunc(dsi_am(false, sa_reg))
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?/sa/register.kbd}").Methods("GET").HandlerFunc(dsi_am(false, sa_reg_kbd))
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/sa/login.htm").Methods("GET").HandlerFunc(dsi_am(false, sa_login))
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/sa/login.kbd").Methods("POST").HandlerFunc(dsi_am(false, sa_login_kbd))
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/sa/success.htm").Methods("GET").HandlerFunc(dsi_am(true, sa_success))

    h.Path("/ds/imagetest.htm").HandlerFunc(dsi_am(false, misc))
    h.Path("/ds/car.htm").HandlerFunc(dsi_am(false, misc))
    h.Path("/ds/postreplytest.htm").HandlerFunc(dsi_am(false, misc))
    h.Path("/ds/test.reply").Methods("POST").HandlerFunc(dsi_am(false, misc))
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/jump").HandlerFunc(dsi_am(false, jump))

    h.NotFoundHandler = loggerMiddleware(retErrorHandler(http.StatusNotFound))
    h.MethodNotAllowedHandler = loggerMiddleware(retErrorHandler(http.StatusMethodNotAllowed))

    h.PathPrefix("/images").HandlerFunc(static)
    h.PathPrefix("/css").HandlerFunc(static)
    
    h.PathPrefix("/api/v1").HandlerFunc(api)
    h.PathPrefix("/api/manage").HandlerFunc(mgmt)

    // define servers
    hatena := &http.Server{Addr: cnf.Listen + ":9000", Handler: h}

    // start on separate thread
    go func() {
        infolog.Printf("started http server")
        err := hatena.ListenAndServe()
        if err != http.ErrServerClosed {
            errorlog.Fatalf("server error: %v", err)
        }
    }()

    // wait and do a graceful exit on ctrl-c / sigterm
    sig := <- sigs
    infolog.Printf("%v: exiting...\n", sig)

    ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
    defer cancel()

    if err := hatena.Shutdown(ctx); err != nil {
        errorlog.Fatalf("graceful shutdown failed! %v", err)
    }

    infolog.Println("server shutdown")
}
