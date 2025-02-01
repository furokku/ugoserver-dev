package main

// ugoserver: a flipnote hatena server with bundled image library
// Usage: accepts one argument in the form of a path to a json config file
//        ./ugoserver [/path/to/config]
// database: postgresql, must be compiled with openssl for pgcrypto
// ipc: thru a unix socket connection and ugoctl control program, tbd
// api: thru REST, tbd
//
// support should only be enabled for the latest version of flipnote studio
// US/EU only had one, but JP had three so only the latest version will work
// theoretically rev2 works, modify to enable that if you want
//
// make sure that the client receives as few non-200 responses as possible
// (preferably zero), as this makes flipnote studio behave strangely sometimes
//
// TODO:
// Command search
// Lots of tlc on the templates for movies, comments, secondary auth
// Text comments
// Users have expendable stars
// Mail
// More sorting modes
// API
// Web interface
// Build channels menu automatically
// Creator's room
// Documentation
// Rate limit
// Inform the user when the session expires within flipnote studio
// CLI tool for things like whitelist, channels, bans, etc.

import (
	"database/sql"
	"fmt"

	"encoding/json"
	"html/template"
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

    sessions = make(map[string]Session)

    menus = make(map[string]Ugomenu)
    templates = make(map[string]*template.Template)
)

const (
    SOCKET_FILE = "/tmp/ugoserver.sock"
)

func main() {
    
    infolog.Printf("starting ugoserver")

    // Load config file
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
    infolog.Printf("loaded config %s", cf)
    
    // load html templates
    // parsing lots of templates into one *template.Template produced weird results, so
    // they are stored in a map
    rd, err := os.ReadDir(cnf.Dir + "/static/template")
    if err != nil {
        errorlog.Fatalln(err)
    }
    for _, tpl := range rd {
        if tpl.IsDir() {
            continue
        }
        name := strings.Split(tpl.Name(), ".")[0]
        p, err := template.ParseFiles(fmt.Sprintf("%s/static/template/%s", cnf.Dir, tpl.Name()))
        if err != nil {
            errorlog.Printf("%v\n", err)
            continue
        }
        templates[name] = p
    }
    infolog.Printf("loaded %d html templates", len(templates))

    // load ugomenus
    rd, err = os.ReadDir(cnf.Dir + "/static/menu")
    if err != nil {
        errorlog.Fatalln(err)
    }
    for _, menu := range rd {
        if menu.IsDir() { // ignore subdirs
            continue
        }
        name := strings.Split(menu.Name(), ".")[0]
        bytes, err := os.ReadFile(fmt.Sprintf("%s/static/menu/%s", cnf.Dir, menu.Name()))
        if err != nil {
            errorlog.Printf("%v\n", err)
            continue
        }
        tu := Ugomenu{}
        err = json.Unmarshal(bytes, &tu)
        if err != nil {
            errorlog.Printf("error loading %s: %v", name, err)
            continue
        }

        menus[name] = tu
    }
    infolog.Printf("loaded %d ugomenus", len(menus))

    // prep graceful exit
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, os.Interrupt)


    // connect to db
    cs := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", cnf.DB.Host, cnf.DB.Port, cnf.DB.User, cnf.DB.Pass, cnf.DB.Name)
    
    db, err = sql.Open("postgres", cs)
    if err != nil {
        errorlog.Fatalf("could not connect to database: %v", err)
    }
    if err := db.Ping(); err != nil { // Ping the database to ensure it is reachable
        errorlog.Fatalf("could not reach database: %v", err)
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
    // may change later if needed
    go pruneSids()

    // hatena auth/general http server
    //
    // gorilla/mux allows accepting requests for
    // a range of urls, then filtering them as needed
    h := mux.NewRouter() // hatena

    // log requests as they come in, eliminates a bunch of redundant code
    h.Use(logger)
    
    h.NotFoundHandler = logger(returncode(http.StatusNotFound))
    h.MethodNotAllowedHandler = logger(returncode(http.StatusMethodNotAllowed))

    // Unsupport
    // Rev1 is fundamentally incompatible with this server, display a message in the eula
    // rev2 is basically the same as rev3, with some minor differences, display a message during auth
    //h.Path("/ds/{reg:v2-(?:us|eu|jp)}/{txt:(?:eula)}.txt").Methods("GET").HandlerFunc(eula) // v2
    //h.Path("/ds/{reg:v2-(?:us|eu|jp)}/confirm/{txt:(?:delete|download|upload)}.txt").Methods("GET").HandlerFunc(eula) // v2
    h.Path("/ds/v2/auth").HandlerFunc(nosupport)
    h.Path("/ds/auth").HandlerFunc(nosupport)
    h.Path("/ds/eula.txt").Methods("GET").HandlerFunc(misc)
    h.Path("/ds/confirm/{u:(?:download|delete|upload)}.txt").Methods("GET").HandlerFunc(misc)
    h.Path("/ds/notices.lst").Methods("GET").HandlerFunc(misc)

    // NAS
    h.Path("/ac").Methods("POST").HandlerFunc(nasAuth)
    h.Path("/pr").Methods("POST").HandlerFunc(nasAuth)

    // rev3 auth
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/auth").Methods("GET", "POST").HandlerFunc(hatenaAuth)

    // eula
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/{lang:(?:en)}/{txt:(?:eula)}.txt").Methods("GET").HandlerFunc(eula)
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/{lang:(?:en)}/confirm/{txt:(?:delete|download|upload)}.txt").Methods("GET").HandlerFunc(eula)
    h.Path("/ds/v2-eu/eula_list.tsv").Methods("GET").HandlerFunc(eulatsv) // eu

    // static ugomenus
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/index.ugo").Methods("GET").HandlerFunc(dsi_am(menus["index"].handle(), false, false))
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/channels.ugo").Methods("GET").HandlerFunc(dsi_am(menus["channels"].handle(), false, false)) //todo: query db for channels

    // movies
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/feed.ugo").Methods("GET").HandlerFunc(dsi_am(movieFeed, false, false))
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/channel.ugo").Methods("GET").HandlerFunc(dsi_am(movieChannelFeed, false, false))

    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/flipnote.post").Methods("POST").HandlerFunc(dsi_am(moviePost, true, false))

    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/movie/{movieid}.{ext:(?:ppm|htm|info)}").Methods("GET").HandlerFunc(dsi_am(movieHandler, false, false))
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/movie/{movieid}.{ext:(?:dl)}").Methods("POST").HandlerFunc(dsi_am(movieHandler, false, false))
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/movie/{movieid}.{ext:(?:delete)}").Methods("POST").HandlerFunc(dsi_am(movieHandler, true, false))

    // stars
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/movie/{movieid}.star").Methods("POST").HandlerFunc(dsi_am(starMovie, true, false))
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/movie/{movieid}.star/{color:(?:green|red|blue|purple)}").Methods("POST").HandlerFunc(dsi_am(starMovie, true, false))

    // comments
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/comment/{commentid}.{ext:(?:npf)}").Methods("GET").HandlerFunc(dsi_am(movieReply, true, false))
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/comment/{movieid}.{ext:(?:reply)}").Methods("POST").HandlerFunc(dsi_am(movieReply, true, false))

    // testing
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/debug.htm").Methods("GET").HandlerFunc(dsi_am(debug, false, false))
    h.Path("/ds/redirect.htm").HandlerFunc(dsi_am(misc, true, true))

    // secondary authentication
    h.Path("/ds/{reg:v2-(?:us|eu|jp)/sa/auth.htm}").Methods("GET").HandlerFunc(dsi_am(sa, false, false))
    h.Path("/ds/{reg:v2-(?:us|eu|jp)/sa/register.kbd}").Methods("POST").HandlerFunc(dsi_am(sa_reg_kbd, false, false))
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/sa/login.kbd").Methods("POST").HandlerFunc(dsi_am(sa_login_kbd, false, false))

    // command search
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/jump").HandlerFunc(dsi_am(jump, false, false))

    // static content
    h.PathPrefix("/images").HandlerFunc(static)
    h.PathPrefix("/css").HandlerFunc(static)
    h.Path("/robots.txt").HandlerFunc(static)
    
    // Planned, maybe
    h.PathPrefix("/api/v1").HandlerFunc(api)
    h.PathPrefix("/api/manage").HandlerFunc(mgmt)

    hatena := &http.Server{Addr: cnf.Listen + ":9000", Handler: h}

    // start on separate thread
    go func() {
        infolog.Printf("started http server")
        err := hatena.ListenAndServe()
        if err != http.ErrServerClosed {
            errorlog.Printf("server error: %v", err)
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
