package main;


import (
    "os"
    "io"
    "slices"

    "floc/ugoserver/ugo"
    "fmt"
    "log"

    "github.com/gorilla/mux"
    "net/http"
    "database/sql"

    "encoding/base64"
    "encoding/binary"
    "encoding/hex"
    "strconv"
    "strings"
)


func handleUgo(w http.ResponseWriter, r *http.Request) {

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

    // Why did I even do this this way?
    // This is stupid and should be replaced with
    // a different handler entirely
    // 
    // This approach is stupid and insecure and blehhh!!!
    // but for now it works so I'll put it off

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
        
        vars := mux.Vars(r)
        base := frontBaseUGO

        pageType := vars["type"]
        pageQ := r.URL.Query().Get("page")

        page, err := strconv.Atoi(pageQ)
        if pageQ == "" {
            // do NOT print error message if the query is empty
            page = 1
        } else if err != nil {
            // When the page isn't specified this should be expected
            // TODO: get rid of this under above condition: done
            log.Printf("invalid page passed to %v%v: %v", r.Host, r.URL.Path, err)
            page = 1
        }

        flipnotes, total := getFrontFlipnotes(db, pageType, page)

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
                base64.RawStdEncoding.EncodeToString(encUTF16LE("@" + pageType)),
                "1",
            },
        })

        if page > 1 {
            base.Entries = append(base.Entries, ugo.MenuEntry{
                EntryType: 4,
                Data: []string{
                    fmt.Sprintf("http://flipnote.hatena.com/front/%v.uls?page=%v", pageType, page-1),
                    "100",
                    base64.RawStdEncoding.EncodeToString(encUTF16LE("Previous page")),
                },
            })
        }

        for _, f := range flipnotes {
            tempTmb := getTmbData(f.filename)

            base.Entries = append(base.Entries, ugo.MenuEntry{
                EntryType: 4,
                Data: []string{
                    fmt.Sprintf("http://flipnote.hatena.com/flipnotes/%s.ppm", f.filename),
                    "3",
                    "0",
                    "0", // star counter (TODO)
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
// Changes some statistic in the flipnote viewer maybe?
func handleInfo(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("0\n0\n"))
}


// Return delete, upload, download, eula
func handleEula(w http.ResponseWriter, r *http.Request) {
    log.Printf("received request to %v%v with header %v", r.Host, r.URL.Path, r.Header)

    vars := mux.Vars(r)
    file := vars["file"]

    if !slices.Contains(txtFiles, file) {
        http.Error(w, "not found", http.StatusNotFound)
        return
    }
    
    text, err := os.ReadFile(txtPath + file + ".txt")
    if err != nil {
        log.Printf("handleEula(): WARNING: failed to read %v: %v", file, err)
        text = []byte("\n\nThis is a placeholder.\nYou shouldn't see this.")
    }

    w.Write(encUTF16LE(string(text)))
}


// Simply log the request and do nothing
func sendWip(w http.ResponseWriter, r *http.Request) {
    log.Printf("received request to %v%v with header %v", r.Host, r.URL.Path, r.Header)

    vars := mux.Vars(r)
    ppmPath := "http://flipnote.hatena.com/flipnotes/" + vars["filename"] + ".ppm"

    w.Write([]byte("<html><head><meta name=\"upperlink\" content=\"" + ppmPath + "\"><meta name=\"playcontrolbutton\" content=\"1\"><meta name=\"savebutton\" content=\"" + ppmPath + "\"></head><body><p>wip<br>obviously this would be unfinished</p></body></html>"))
}

// accept flipnotes uploaded thru internal ugomemo:// url
// or flipnote.post url
func postFlipnote(db *sql.DB) http.HandlerFunc {

    // deja vu
    fn := func(w http.ResponseWriter, r *http.Request) {

        log.Printf("received request to %v%v with header %v", r.Host, r.URL.Path, r.Header)

        // make sure request has a valid SID
        // we don't want a flood of random flipnotes
        // after all...
        session, ok := sessions[r.Header.Get("X-Dsi-Sid")]
        if !ok {
            log.Printf("postFlipnote(): unauthorized attempt to post flipnote")
            w.WriteHeader(http.StatusUnauthorized)
            return
        }

        ppmBody, err := io.ReadAll(r.Body)
        if err != nil {
            log.Printf("postFlipnote(): error: failed to read ppm from POST request body! %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }

        filename := strings.ToUpper(hex.EncodeToString(ppmBody[0x78 : 0x7B])) + "_" +
                    string(ppmBody[0x7B : 0x88]) + "_" +
                    editCountPad(binary.LittleEndian.Uint16(ppmBody[0x88 : 0x90]))

        log.Printf("received ppm body from %v %v %v", session.fsid, session.username, filename)

        fp, err := os.OpenFile(dataPath + "flipnotes/" + filename + ".ppm", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
        if err != nil {
            // Realistically, two flipnote filenames shouldn't clash.
            // if it becomes an issue, I will either save them in reference
            // to their id in the database or start adding randomized
            // characters in the end
            log.Printf("postFlipnote(): failed to write open path to ppm: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }

        defer func() {
            if err := fp.Close(); err != nil {
                panic(err)
            }
        }()

        if _, err := fp.Write(ppmBody); err != nil {
            log.Printf("postFlipnote(): failed to write ppm to file: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }

        if _, err := db.Exec("INSERT INTO flipnotes (author_id, filename) VALUES ($1, $2)", session.fsid, filename); err != nil {
            log.Printf("postFlipnote(): failed to update database! %v", err)
        }

        w.WriteHeader(http.StatusOK)
    }
    return fn
}
