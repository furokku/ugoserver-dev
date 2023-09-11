package main;


import (
    "database/sql"
    "os"

    "floc/ugoserver/ugo"
    "fmt"
    "log"

    "github.com/gorilla/mux"
    "net/http"

    "encoding/base64"
    "slices"
    "strconv"
)


func ugoHandler(w http.ResponseWriter, r *http.Request) {

    log.Printf("received %v request to %v%v with header %v\n", r.Method, r.Host, r.URL.Path, w.Header())

    // there was method checking code here
    // but gorilla mux exists

    vars := mux.Vars(r)
    switch vars["ugo"] {

    case "index":
        w.Write(indexUGO.Pack())
    }

    log.Printf("respose 200 at %v%v with header %v", r.Host, r.URL.Path, w.Header())
}


// Replace this (eventually)
func returnFromFs(w http.ResponseWriter, r *http.Request) {

    log.Printf("received %v request to %v%v with header %v\n", r.Method, r.Host, r.URL, r.Header)

    fsPath := "/srv/hatena_storage"

    // TODO: The eula and etc files should probably be read and stored
    // ~~within the server~~ elsewhere
    // That would allow the base files to be in utf8 and get rid of
    // essentially the empty folders that are in
    // hatena/static/ds/ and get rid of some code i dont like
    //
    // done
    data, err := os.ReadFile(fsPath + r.URL.Path)
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
        base := frontBaseUGO

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
        base.Entries = append(base.Entries, ugo.MenuEntry{
            EntryType: 1,
            Data: []string{
                "0",
                base64.StdEncoding.EncodeToString(encUTF16LE("Front page")),
                base64.StdEncoding.EncodeToString(encUTF16LE(fmt.Sprintf("Page %d / %d", page, countPages(total)))),
            },
        })
        base.Entries = append(base.Entries, ugo.MenuEntry{
            EntryType: 2, // category
            Data: []string{
                "http://flipnote.hatena.com/front/recent.uls",
                base64.RawStdEncoding.EncodeToString(encUTF16LE("@" + pageName)),
                "1",
            },
        })

        for _, f := range flipnotes {
            tempTmb := getTmbData(f.author, f.filename)

            base.Entries = append(base.Entries, ugo.MenuEntry{
                EntryType: 4,
                Data: []string{
                    fmt.Sprintf("http://flipnote.hatena.com/flipnotes/creators/%s/%s.ppm", f.author, f.filename),
                    "3",
                    "0",
                    "42", // star counter (TODO)
                    fmt.Sprint(tempTmb.flipnoteIsLocked()),
                    "0", // ??
                },
            })

            base.Embed = append(base.Embed, tempTmb)
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


// I have no idea why this is needed
// nor what it does
func infoHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("0\n0\n"))
}


// Return delete, upload, download, eula
func eulaHandler(w http.ResponseWriter, r *http.Request) {
    log.Printf("received request to %v%v with header %v", r.Host, r.URL.Path, r.Header)

    vars := mux.Vars(r)
    file := vars["file"]

    if !slices.Contains(txtFiles, file) {
        http.Error(w, "not found", http.StatusNotFound)
        return
    }
    
    text, err := os.ReadFile(txtPath + file + ".txt")
    if err != nil {
        log.Printf("eulaHandler: WARNING: failed to read %v: %v", file, err)
        text = []byte("\n\nThis is a placeholder.\nYou shouldn't see this.")
    }

    w.Write(encUTF16LE(string(text)))
}


// Simply log the request and do nothing
func logRequest(w http.ResponseWriter, r *http.Request) {
    log.Printf("received request to %v%v with header %v", r.Host, r.URL.Path, r.Header)

    w.WriteHeader(http.StatusOK)
}
