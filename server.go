package main

// ugoserver: a flipnote hatena server with bundled image library
// Usage: accepts one argument in the form of a path to a json config file
//        ./ugoserver [/path/to/config]
// database: postgresql, must be compiled with openssl for pgcrypto
// ipc: thru unix socket, use bundled client for a simple command line
// api: thru REST, tbd
//
// support should only be enabled for the latest version of flipnote studio
// US/EU only had one, but JP had three so only the latest version will work
// theoretically rev2 works, modify to enable that if you want
//
// make sure that the client receives as few non-200 responses as possible
// (preferably zero), as this makes flipnote studio behave strangely sometimes
//
// get ready for boilerplate code galore, because this is my first big
// Go project and I have no idea what I'm doing sometimes!
//
// TODO:
// Command search
// Lots of tlc on the templates for movies, comments, secondary auth
// Text comments
// Users have expendable stars W
// Mail
// More sorting modes
// API
// Web interface
// Build channels menu automatically W
// Creator's room
// Documentation
// Inform the user when the session expires within flipnote studio

import (
	"database/sql"
	"fmt"

	"net/http"

	"github.com/gorilla/mux"

	"context"
	"os"
	"os/signal"
	"time"
)

var (
    db *sql.DB

    sessions = make(map[string]Session)
)

const (
    SOCKET_FILE = "/tmp/ugoserver.sock"
)

func main() {
    
    infolog.Println("starting ugoserver")
    defer infolog.Println("goodbye!")
    
    // barzo dzekuje
    if err := load_config(false); err != nil {
        errorlog.Fatalf("failed to load configuration: %v", err)
    }
    if err := load_menus(false); err != nil {
        errorlog.Printf("failed to load menus: %v", err)
    }
    if err := load_templates(false); err != nil {
        errorlog.Printf("failed to load templates: %v", err)
    }

    // listen for ^C
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, os.Interrupt)

    // connect to db
    cs := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", cnf.DB.Host, cnf.DB.Port, cnf.DB.User, cnf.DB.Pass, cnf.DB.Name)
    
    db, err := sql.Open("postgres", cs)
    if err != nil {
        errorlog.Fatalf("could not connect to database: %v", err)
    }
    if err := db.Ping(); err != nil { // Ping the database to ensure it is reachable
        errorlog.Fatalf("could not reach database: %v", err)
    }

    infolog.Printf("connected to database")

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
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/channels.ugo").Methods("GET").HandlerFunc(dsi_am(channelMainMenu, false, false)) //todo: query db for channels
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/debug.ugo").Methods("GET").HandlerFunc(dsi_am(menus["debug"].handle(), false, false))

    // comments
    // note: Due to how the servemux works, this has to be before
    // movieHandler in order to work
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/movie/{movieid}.{ext:(?:htm)}").Queries("mode", "comment").Methods("GET").HandlerFunc(dsi_am(replyHandler, false, false))
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/comment/{commentid}.{ext:(?:npf)}").Methods("GET").HandlerFunc(dsi_am(replyHandler, false, false))
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/comment/{movieid}.{ext:(?:reply)}").Methods("POST").HandlerFunc(dsi_am(replyPost, true, false))

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

    // testing
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/debug.htm").Methods("GET").HandlerFunc(dsi_am(debug, false, false))
    h.Path("/ds/redirect.htm").HandlerFunc(dsi_am(misc, true, true))
    h.Path("/ds/v2-us/").HandlerFunc(misc)

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
    
    // mail test
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/mail/addresses.ugo").Methods("GET").HandlerFunc(dsi_am(menus["addresstest"].handle(), false, false))
    h.Path("/ds/{reg:v2-(?:us|eu|jp)}/mail.send").Methods("POST").HandlerFunc(dsi_am(misc, false, false))
    
    // Planned, maybe
    h.PathPrefix("/api/v1").HandlerFunc(api)
    h.PathPrefix("/api/manage").HandlerFunc(mgmt)

    hatena := &http.Server{Addr: cnf.Listen + ":9000", Handler: h}

    // start web server
    go func() {
        infolog.Printf("serving http on %v", cnf.Listen)
        err := hatena.ListenAndServe()
        if err != http.ErrServerClosed {
            errorlog.Printf("server error: %v", err)
            sigs <- os.Interrupt
        }
    }()

    // start unix socket for ipc
    // curious how this works on windows
    os.RemoveAll(SOCKET_FILE)

    // cli commands
    ch := newCmdHandler()
    ch.register("whitelist", whitelist)
    ch.register("reload", reload)
    ch.register("ban", ban)
    ch.register("pardon", pardon)
    ch.register("stat", show)
    ch.register("channel", channel)
    ch.register("movie", movie)

    ipc := newIpcListener(SOCKET_FILE, *ch)
    infolog.Printf("serving unix socket on %v", SOCKET_FILE)

    // wait and do a graceful exit on ctrl-c / sigterm
    sig := <- sigs
    infolog.Printf("%v: shutting down...", sig)
    
    // close unix socket
    ipc.stop()

    ctx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
    defer cancel()

    if err := hatena.Shutdown(ctx); err != nil {
        errorlog.Fatalf("graceful shutdown failed: %v", err)
    }
    
    // close db connection
    db.Close()
}
