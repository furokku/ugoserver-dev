package main;


import (
    "time"
    "log"
    "net/http"
    "fmt"
    "encoding/base64"
    "strings"
    "os"
    "github.com/gorilla/mux"
//  "slices"
//  "regexp"
    "database/sql"
    "strconv"
    "io"
)


func runHatenaServer(db *sql.DB) {
    
    // gorilla/mux allows accepting requests for
    // a range of urls, then filtering them as needed
    m := mux.NewRouter()

    // didn't produce desired results, using
    // basic "read file, return it if it exists" handler
//  fs := http.FileServer(http.Dir("./static"))
//  hatenaMux.Handle("/", denyIndex(fs))

    // v1
    // likely not going to support this because it's just
    // going to be a giant hassle and is probably missing
    // features that are in v2 proper
    // 
    // has reduced headers, sends two GET requests to auth
//  m.Path("/ds/{sub}/{ugo}.ugo").HandlerFunc(ugoHandler)
//  m.Path("/ds/{ugo}.ugo").HandlerFunc(ugoHandler)
//  m.Path("/ds/auth").HandlerFunc(hatenaAuthHandler)

//  m.Path("/ds/{file}.txt").HandlerFunc(returnFromFs)
//  m.Path("/ds/confirm/{file}.txt").HandlerFunc(returnFromFs)

    // TODO: tv-jp
    // v2-us, v2-eu, v2-jp, v2 auth
    m.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/auth").HandlerFunc(hatenaAuthHandler)

    // eula
    m.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/{lang}/{file}.txt").HandlerFunc(returnFromFs)
    m.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/{lang}/confirm/{file}.txt").HandlerFunc(returnFromFs)

    m.Path("/ds/{reg}:v2(?:-(?:us|eu|jp))?/{file}.txt").HandlerFunc(returnFromFs) // v2
    m.Path("/ds/{reg}:v2(?:-(?:us|eu|jp))?/confirm/{file}.txt").HandlerFunc(returnFromFs) // v2

    // region no longer matters from here on down
    m.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/{ugo}.ugo").HandlerFunc(ugoHandler)
    m.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/{sub}/{ugo}.ugo").HandlerFunc(ugoHandler)
    m.Path("/ds/{reg:v2(?:-(?:us|eu|jp))?}/{file}.htm").HandlerFunc(returnFromFs)

    // return a built ugo file with flipnotes
    m.Path("/front/{type:(?:recent|hot|liked)}.ugo").HandlerFunc(serveFrontPage(db))

    // stuff
    m.Path("/flipnotes/creators/{authorfsid:[A-F0-9]{16}}/{filename}.ppm").HandlerFunc(returnFromFs)
    m.Path("/flipnotes/creators/{authorfsid:[A-F0-9]{16}}/{filename}.htm").HandlerFunc(flipnoteHtmHandler)
    m.Path("/flipnotes/creators/{authorfsid:[A-F0-9]{16}}/{filename}.info").HandlerFunc(infoHandler)

    
    err := http.ListenAndServe(":9000", m)
    if err != nil {
        log.Fatalf("server closed, error: %v", err)
    }
}

func hatenaAuthHandler(w http.ResponseWriter, r *http.Request) {

    log.Printf("received %v request to %v%v with header %v\n", r.Method, r.Host, r.URL.Path, r.Header)

    // feels kinda redundant but i wrote this
    // earlier and don't feel like removing it (entirely)
//  match, _ := regexp.MatchString("/ds/v2(-[a-z]{2})?/auth", r.URL.Path)
//  vars := mux.Vars(r)

    // verify region in auth url is correct
    // mux inline regex works so this can be commented out
//  if !slices.Contains(regions, vars["reg"]) {
//      http.Error(w, "invalid region", http.StatusNotFound)
//      log.Printf("response 404 (invalid region) at %v%v", r.Host, r.URL.Path)
//      return
//  }

    switch r.Method {

    // > only GET and POST requests will
    // > ever be sent
    // correction: initial rev of flipnote studio sends
    // two GET requests, will need to consider later
    // how to handle that
    //
    // Likely wont
    case "GET":

        // seems like it's used to handle some sort of
        // server-wide notifications, as opposed to
        // user-specific ones which could be set later
        // TODO: Maybe this but seems unnecessary
        // Could be read from database if implemented
        const serverUnread int = 0

        if (serverUnread != 0) && (serverUnread != 1) {
            w.WriteHeader(http.StatusInternalServerError)
            return
        } else {
            w.Header()["X-DSi-Unread-Notices"] = []string{fmt.Sprint(serverUnread)}
            w.Header()["X-DSi-New-Notices"] = []string{fmt.Sprint(serverUnread)}
        }

        // TODO: validate auth challenge
        // I know it has something to do with XOR keys
        // but is it really needed? probably not
        w.Header()["X-DSi-Auth-Challenge"] = []string{randAsciiString(8)}
        w.Header()["X-DSi-SID"] = []string{genUniqueSession()}

    case "POST":

        req := authPostRequest{
            mac:      r.Header.Get("X-Dsi-Mac"),
            id:       r.Header.Get("X-Dsi-Id"), // FSID
            auth:     r.Header.Get("X-Dsi-Auth-Response"),
            sid:      r.Header.Get("X-Dsi-Sid"),
            ver:      r.Header.Get("X-Ugomemo-Version"), // it could be made so that only V2 is
                                                         // accepted for this header, but the region
                                                         // thing pretty much does that already
            username: r.Header.Get("X-Dsi-User-Name"),
            region:   r.Header.Get("X-Dsi-Region"),
            lang:     r.Header.Get("X-Dsi-Lang"),
            country:  r.Header.Get("X-Dsi-Country"),
            birthday: r.Header.Get("X-Birthday"), // weird how this one doesn't have DSi in it
            datetime: r.Header.Get("X-Dsi-Datetime"),
            color:    r.Header.Get("X-Dsi-Color"),
        }

        // TODO: function to validate auth request
//      if !req.validate() {
        if false {
            w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
            w.Write(encUTF16LE("error authenticating!"))
            return
        } else {
            sessions[req.sid] = struct{fsid string; issued int64}{fsid: req.id, issued: time.Now().Unix()}
            w.Header()["X-DSi-SID"] = []string{req.sid}

            // TODO: handle on per user basis
            // both of these do the same thing probably but
            // for convenience sake likely only one
            // will be set
            w.Header()["X-DSi-New-Notices"] = []string{"0"}
            w.Header()["X-DSi-Unread-Notices"] = []string{"0"}

//          log.Println(sessions)
        }

    default:
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        log.Printf("response 405 at %v%v", r.Host, r.URL.Path)
        return
    }

    w.WriteHeader(http.StatusOK)
    log.Printf("response 200 at %v%v with header %v\n", r.Host, r.URL.Path, w.Header())
}

func ugoHandler(w http.ResponseWriter, r *http.Request) {

    log.Printf("received %v request to %v%v with header %v\n", r.Method, r.Host, r.URL.Path, w.Header())

    if r.Method != "GET" {
        // this should only really receive GET requests
        // unless I find out otherwise
        http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        log.Printf("response 405 at %v%v", r.Host, r.URL.Path)
        return
    }

    vars := mux.Vars(r)
    switch vars["ugo"] {

    case "index":
        w.Write(indexUGO.Pack())
    }

    log.Printf("respose 200 at %v%v with header %v", r.Host, r.URL.Path, w.Header())
}


func returnFromFs(w http.ResponseWriter, r *http.Request) {

    log.Printf("received %v request to %v%v with header %v\n", r.Method, r.Host, r.URL, r.Header)

    staticUrl := "/srv/ugoserver/hatena/static"
    if strings.Contains(r.URL.Path, ".ppm") {
        staticUrl = "/srv/ugoserver/hatena"
    }

    // http.FileServer did not produce the results
    // that were preferable for me so i am
    // manually checking if files exist and returning them
    // on a per-url basis
    //
    // TODO: The eula and etc files should probably be read and stored
    // in a buffer within the server
    // That would allow the base files to be in utf8 and get rid of
    // essentially the empty folders that are in
    // hatena/static/ds/ and get rid of this
    data, err := os.ReadFile(strings.Join([]string{staticUrl, r.URL.Path}, ""))
    if err != nil {
        http.Error(w, "not found", http.StatusNotFound)
        log.Printf("response 404 at %v%v (file handler): %v", r.Host, r.URL.Path, err)
        return
    }

    w.Write(data)
    log.Printf("response 200 at %v%v with headers %v", r.Host, r.URL.Path, w.Header())
}


// Handler for building ugomenus for the front page
// recent, hot, most liked, etc..
func serveFrontPage(db *sql.DB) http.HandlerFunc {

    // very amazing wrapper i am programer
    fn := func(w http.ResponseWriter, r *http.Request) {

        log.Printf("received request to %v%v", r.Host, r.URL.Path)
        
        var total int
        var pageName string
        var flipnotes []flipnote

        vars := mux.Vars(r)
        base := fpBase

        page, err := strconv.Atoi(r.URL.Query().Get("page"))
        if err != nil {
            // When the page isn't specified this should be expected
            // TODO: get rid of this under above condition
            log.Printf("invalid page passed to %v%v: %v", r.Host, r.URL.Path, err)
            page = 1
        }

        // TODO: Hot / most liked flipnotes
        switch vars["type"] {
        case "recent":
            pageName = "Recent"
            flipnotes, total = getLatestFlipnotes(db, page)
        }

        // Add top screen titles
        base.entries = append(base.entries, menuEntry{
            entryType: 1,
            data: []string{
                "0",
                base64.StdEncoding.EncodeToString(encUTF16LE("Front page")),
                base64.StdEncoding.EncodeToString(encUTF16LE(fmt.Sprintf("Page %d / %d", page, countPages(total)))),
            },
        })
        base.entries = append(base.entries, menuEntry{
            entryType: 2, // category
            data: []string{
                "http://flipnote.hatena.com/front/recent.uls",
                base64.RawStdEncoding.EncodeToString(encUTF16LE("@" + pageName)),
                "1",
            },
        })

        for _, f := range flipnotes {
            tempTmb := getTmbData(f.author, f.filename)

            base.entries = append(base.entries, menuEntry{
                entryType: 4,
                data: []string{
                    fmt.Sprintf("http://flipnote.hatena.com/flipnotes/creators/%s/%s.ppm", f.author, f.filename),
                    "3",
                    "0",
                    "42", // star counter (TODO)
                    fmt.Sprint(tempTmb.flipnoteIsLocked()),
                    "0", // ??
                },
            })

            base.embed = append(base.embed, tempTmb)
            //fmt.Printf("debug: length of tmb %v is %v\n", n, len(tempTmb))
        }

        // TODO: add previous/next page buttons
        data := base.Pack()

        //fmt.Println(string(data))
        w.Write(data)
    }


    // inline return is ugly
    return fn
}


func logRequest(w http.ResponseWriter, r *http.Request) {
    log.Printf("received request to %v%v with header %v", r.Host, r.URL.Path, r.Header)

    w.WriteHeader(http.StatusOK)
}


func infoHandler(w http.ResponseWriter, r *http.Request) {
    io.WriteString(w, "0\n0\n")
    w.WriteHeader(http.StatusOK)
}


func flipnoteHtmHandler(w http.ResponseWriter, r *http.Request) {
    log.Printf("received request to %v%v with header %v", r.Host, r.URL.Path, r.Header)

    w.WriteHeader(http.StatusOK)
}
