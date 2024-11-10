package main

import (
    _ "github.com/lib/pq"
    "database/sql"

    "fmt"
    "encoding/json"
    "strings"

    "github.com/gorilla/mux"
    "net/http"

    "context"
    "os"
    "os/signal"
    "time"
)

var (
    db *sql.DB
    configuration = Configuration{}
    sessions = make(map[string]session)
    prettyPageTypes = map[string]string{"recent":"Recent"}
    loadedUgos = make(map[string]Ugomenu)
)

func main() {

    // Flags are kinda useless because this will always
    // be used with a configuration file
    configFile := "default.json"
    if len(os.Args) > 1 {
        configFile = os.Args[1]
    }

    cbytes, err := os.ReadFile(configFile)
    if err != nil {
        errorlog.Fatalf("failed to open config file: %v", err)
    }

    json.Unmarshal(cbytes, &configuration)
    if err != nil {
        errorlog.Fatalf("failed to load config file: %v", err)
    }
    debuglog.Printf("loaded config %v", configuration)

    // temporary workaround until i come up with a better format
    // for static/template ugos that don't need to change

    // done
    ugos, err := os.ReadDir(configuration.HatenaDir + "/ugo")
    if err != nil {
        errorlog.Printf("%v", err)
    }
    for _, ugo := range ugos {
        if ugo.IsDir() { // ignore dirs in /ugo/
            continue
        }
        name := strings.Split(ugo.Name(), ".")[0]
        bytes, err := os.ReadFile(configuration.HatenaDir + "/ugo/" + ugo.Name())
        if err != nil {
            errorlog.Printf("%v", err)
        }
        tu := Ugomenu{}
        json.Unmarshal(bytes, &tu)

        loadedUgos[name] = tu
    }

    // prep graceful exit
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, os.Kill, os.Interrupt)

    dbCfg := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
                         configuration.DbHost, configuration.DbPort, configuration.DbUser, configuration.DbPass, configuration.DbName)


    // connect to database
    db, err = sql.Open("postgres", dbCfg)
    if err != nil {
        errorlog.Fatalf("failed to open database: %v", err)
    } else if err := db.Ping(); err != nil {
        errorlog.Fatalf("failed to reach database: %v", err)
    }
    debuglog.Printf("connected to database %v @ %v:%v as %v", configuration.DbName, configuration.DbHost, configuration.DbPort, configuration.DbUser)

    defer db.Close()

    // start unix socket for ipc
    sf := "/tmp/ugoserver.sock"
    os.RemoveAll(sf)
    ipcS := newIpcListener(sf)
    debuglog.Printf("started unix socket listener")

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
    h.Path("/ds/v2-eu/eula_list.tsv").Methods("GET").HandlerFunc(handleEulaTsv)

    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/index.ugo").Methods("GET").HandlerFunc(loadedUgos["index"].ugoHandle())
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/{file}.htm").Methods("GET").HandlerFunc(func(w http.ResponseWriter, r *http.Request){w.WriteHeader(http.StatusNotImplemented);return})

    // return a built ugo file with flipnotes
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/feed.ugo").Methods("GET").HandlerFunc(serveFrontPage)

    // uploading
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/flipnote.post").Methods("POST").HandlerFunc(postFlipnote)

    // related to fetching flipnotes
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/movie/{id}.{ext:(?:ppm|htm|info|dl|delete)}").Methods("GET", "POST").HandlerFunc(movieHandler)
    // stars
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/movie/{id}.star").Methods("POST").HandlerFunc(starMovieHandler)
    h.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/movie/{id}.star/{color:(?:green|red|blue|purple)}").Methods("POST").HandlerFunc(starMovieHandler)

    h.Path("/ac").Methods("POST").HandlerFunc(nasAuth).Host("nas.nintendowifi.net")
    h.Path("/pr").Methods("POST").HandlerFunc(nasAuth).Host("nas.nintendowifi.net")

    h.Path("/ds/imagetest.htm").HandlerFunc(misc)
    h.Path("/ds/postreplytest.htm").HandlerFunc(misc)
    h.Path("/ds/test.reply").Methods("POST").HandlerFunc(misc)

    h.NotFoundHandler = loggerMiddleware(retErrorHandler(http.StatusNotFound))
    h.MethodNotAllowedHandler = loggerMiddleware(retErrorHandler(http.StatusMethodNotAllowed))

    h.Use(loggerMiddleware)

    h.PathPrefix("/images").HandlerFunc(static)

    // define servers
    hatena := &http.Server{Addr: configuration.Listen + ":9000", Handler: h}

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
