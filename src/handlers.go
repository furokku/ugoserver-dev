package main


import (
    "os"
    "io"

    "floc/ugoserver/ugo"
    "fmt"

    "github.com/gorilla/mux"
    "net/http"

    "encoding/base64"
    "encoding/binary"
    "encoding/hex"
    "strconv"
    "strings"
)



// Not my finest code up there so we're doing this a better way
func serveFlipnotes(w http.ResponseWriter, r *http.Request) {

    vars := mux.Vars(r)

    id := vars["id"]
    ext := vars["ext"]

    path := "/flipnotes/" + id + ".ppm"

    switch ext {
        case "ppm":
            data, err := os.ReadFile(configuration.HatenaDir + "/hatena_storage" + path)
            if err != nil {
                w.WriteHeader(http.StatusNotFound)
                return
            }

            w.Write(data)
//          log.Printf("sent %d bytes to %v", len(data), r.Header.Get("X-Real-Ip"))
            return

        case "htm":
            if fi, err := os.Stat(configuration.HatenaDir + "/hatena_storage" + path); err == nil {
                w.Write([]byte(fmt.Sprintf("<html><head><meta name=\"upperlink\" content=\"%s\"><meta name=\"playcontrolbutton\" content=\"1\"><meta name=\"savebutton\" content=\"%s\"></head><body><p>wip<br>obviously this would be unfinished<br><br>debug:<br>file: %s<br>size: %d<br>modified: %s</p></body></html>", configuration.ServerUrl+path, configuration.ServerUrl+path, id, fi.Size(), fi.ModTime())))
                return
            } else {
                w.WriteHeader(http.StatusNotFound)
                return
            }

        case "info":
            w.Write([]byte{0x30, 0x0A, 0x30, 0x0A}) // write 0\n0\n because flipnote is weird
            return

        default:
            w.WriteHeader(http.StatusNotFound)
            return
    }
}


// Handler for building ugomenus for the front page
// recent, hot, most liked, etc..
func serveFrontPage(w http.ResponseWriter, r *http.Request) {
    
    vars := mux.Vars(r)
    base := gridBaseUGO

    pageType := vars["type"]
    pageQ := r.URL.Query().Get("page")

    page, err := strconv.Atoi(pageQ)
    if pageQ == "" {
        // do NOT print error message if the query is empty
        page = 1
    } else if err != nil {
        // When the page isn't specified this should be expected
        // TODO: get rid of this under above condition: done
        infolog.Printf("%v passed invalid page to %v%v: %v", r.Header.Get("X-Real-Ip"), r.Host, r.URL.Path, err)
        page = 1
    }

    flipnotes, total := getFrontFlipnotes(pageType, page)

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
            configuration.ServerUrl + "/front/recent.uls",
            base64.StdEncoding.EncodeToString(encUTF16LE(pageType)),
            "1",
        },
    })

    if page > 1 {
        base.Entries = append(base.Entries, ugo.MenuEntry{
            EntryType: 4,
            Data: []string{
                fmt.Sprintf(configuration.ServerUrl + "/front/%s.uls?page=%d", pageType, page-1),
                "100",
                base64.StdEncoding.EncodeToString(encUTF16LE("Previous page")),
            },
        })
    }

    for _, f := range flipnotes {
        tempTmb := f.TMB()
        if tempTmb == nil {
            warnlog.Printf("tmb is nil")
            w.WriteHeader(http.StatusInternalServerError)
            return
        }

        base.Entries = append(base.Entries, ugo.MenuEntry{
            EntryType: 4,
            Data: []string{
                fmt.Sprintf(configuration.ServerUrl + "/flipnotes/%d.ppm", f.id),
                "3",
                "0",
                "420", // star counter (TODO)
                fmt.Sprint(f.lock),
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


// I have no idea why this is needed
// nor what it does
// Changes some statistic in the flipnote viewer maybe?
// Replaced by a catchall function
/* func handleInfo(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("0\n0\n"))
} */


// Return delete, upload, download, eula
func handleEula(w http.ResponseWriter, r *http.Request) {

    vars := mux.Vars(r)
    txt := vars["txt"]

    // if !slices.Contains(txts, file) {
    //    http.Error(w, "not found", http.StatusNotFound)
    //    return
    //}
    
    text, err := os.ReadFile(configuration.HatenaDir + "/static/txt/" + txt + ".txt")
    if err != nil {
        warnlog.Printf("failed to read %v: %v", txt, err)
        text = []byte("\n\nThis is a placeholder.\nYou shouldn't see this.")
    }

    w.Write(encUTF16LE(string(text)))
}


// accept flipnotes uploaded thru internal ugomemo:// url
// or flipnote.post url
func postFlipnote(w http.ResponseWriter, r *http.Request) {

    // make sure request has a valid SID
    // we don't want a flood of random flipnotes
    // after all...
    session, ok := sessions[r.Header.Get("X-Dsi-Sid")]
    if !ok {
        warnlog.Printf("unauthorized attempt to post flipnote")
        w.WriteHeader(http.StatusUnauthorized)
        return
    }

    ppmBody, err := io.ReadAll(r.Body)
    if err != nil {
        errorlog.Printf("failed to read ppm from POST request body! %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    var id int
    aid := strings.ToUpper(hex.EncodeToString(reverse(ppmBody[0x5E : 0x66])))
    an := base64.StdEncoding.EncodeToString(decUTF16LE(ppmBody[0x40 : 0x56 ]))
    paid := strings.ToUpper(hex.EncodeToString(reverse(ppmBody[0x56 : 0x5E])))
    pan := base64.StdEncoding.EncodeToString(decUTF16LE(ppmBody[0x2A : 0x40]))
    l := int(ppmBody[0x10])
    afn := strings.ToUpper(hex.EncodeToString(ppmBody[0x78 : 0x7B])) + "_" +
                string(ppmBody[0x7B : 0x88]) + "_" +
                editCountPad(binary.LittleEndian.Uint16(ppmBody[0x88 : 0x90]))

    debuglog.Printf("received ppm body from %v %v %v", session.fsid, session.username, afn)

    if err := db.QueryRow("INSERT INTO flipnotes (author_id, author_name, parent_author_id, parent_author_name, author_filename, lock) VALUES ($1, $2, $3, $4, $5, $6) RETURNING (id)", aid, an, paid, pan, afn, l).Scan(&id); err != nil {
        errorlog.Printf("failed to update database: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    fmt.Println(id)

    if _, err := db.Exec("INSERT INTO stars (id) VALUES ($1)", id); err != nil {
        errorlog.Printf("%v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    fp, err := os.OpenFile(configuration.HatenaDir + "/hatena_storage/flipnotes/" + fmt.Sprint(id) + ".ppm", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
    if err != nil {
        // store by id to not allow filename clashes
        errorlog.Printf("failed to open path to ppm: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    defer func() {
        if err := fp.Close(); err != nil {
            panic(err)
        }
    }()

    if _, err := fp.Write(ppmBody); err != nil {
        errorlog.Printf("failed to write ppm to file: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }


    w.WriteHeader(http.StatusOK)
}

func retErrorHandler(code int) http.HandlerFunc {
    fn := func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(code)
    }

    return http.HandlerFunc(fn)
}
