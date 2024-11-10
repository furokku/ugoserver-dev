package main


import (
    "os"
    "io"

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
func movieHandler(w http.ResponseWriter, r *http.Request) {

    vars := mux.Vars(r)

    idn, err := strconv.Atoi(vars["id"])
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    ext := vars["ext"]

    path := fmt.Sprintf("/ds/%s/movie/%d", vars["reg"], idn)

    switch ext {
    case "dl":
        err := updateViewDlCount(idn, ext)
        if err != nil {
            errorlog.Printf("failed to update %v count: %v", ext, err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusOK)
        return

    case "delete":
        err := deleteFlipnote(idn)
        if err != nil {
            errorlog.Printf("failed to delete %v: %v", idn, err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusOK)
        return

    case "ppm":
        data, err := os.ReadFile(fmt.Sprintf("%s/hatena_storage/flipnotes/%d.ppm", configuration.HatenaDir, idn))
        if err != nil {
            w.WriteHeader(http.StatusNotFound)
            return
        }

        err = updateViewDlCount(idn, ext)
        if err != nil {
            errorlog.Printf("failed to update %v count: %v", ext, err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        w.Write(data)
//          log.Printf("sent %d bytes to %v", len(data), r.Header.Get("X-Real-Ip"))
        return

    case "htm":
        fi, err := os.Stat(fmt.Sprintf("%s/hatena_storage/flipnotes/%d.ppm", configuration.HatenaDir, idn))

        if err != nil {
            w.WriteHeader(http.StatusNotFound)
            return
        }
        flip, err := getFlipnoteById(idn)
        if err != nil {
            errorlog.Printf("could not get flipnote %v: %v", idn, err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }

        w.Write([]byte(fmt.Sprintf("<html><head><meta name=\"upperlink\" content=\"%s\"><meta name=\"playcontrolbutton\" content=\"1\"><meta name=\"savebutton\" content=\"%s\"><meta name=\"starbutton\" content=\"%s\"><meta name=\"starbutton1\" content=\"%s\"><meta name=\"starbutton2\" content=\"%s\"><meta name=\"starbutton3\" content=\"%s\"><meta name=\"starbutton4\" content=\"%s\"><meta name=\"deletebutton\" content=\"%s\"></head><body><p>wip<br>obviously this would be unfinished<br>yellow<span class=\"star0\">%d</span><br>green<span class=\"star1\">%d</span><br>red<span class=\"star2\">%d</span><br>blue<span class=\"star3\">%d</span><br>purple<span class=\"star4\">%d</span><br><br>debug:<br>file: %v<br>size: %d<br>modified: %s</p></body></html>", configuration.ServerUrl+path+".ppm", configuration.ServerUrl+path+".ppm", configuration.ServerUrl+path+".star",configuration.ServerUrl+path+".star/green,99",configuration.ServerUrl+path+".star/red,99",configuration.ServerUrl+path+".star/blue,99",configuration.ServerUrl+path+".star/purple,99", configuration.ServerUrl+path+".delete", flip.stars["yellow"], flip.stars["green"], flip.stars["red"], flip.stars["blue"], flip.stars["purple"], idn, fi.Size(), fi.ModTime())))
        return

    case "info":
        w.Write([]byte{0x30, 0x0A, 0x30, 0x0A}) // write 0\n0\n because flipnote is weird
        return

    default:
        w.WriteHeader(http.StatusNotFound)
        return
    }
}

// add stars to flipnote
func starMovieHandler(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := strconv.Atoi(vars["id"])
    if err != nil {
        errorlog.Printf("bad id when adding star: %v", err)
    }
    count, err := strconv.Atoi(r.Header.Get("X-Hatena-Star-Count"))
    if err != nil {
        errorlog.Printf("bad star count: %v", err)
    }
    color, ok := vars["color"]
    if !ok {
        color = "yellow"
    }
    
    sess, ok := sessions[r.Header.Get("X-Dsi-Sid")]
    if !ok {
        w.WriteHeader(http.StatusForbidden)
        return
    }

    err = updateStarCount(id, color, count)
    if err != nil {
        errorlog.Printf("failed to update star count for %v: %v", id, err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    //TODO: add to user's starred flipnotes
    err = updateUserStarredMovies(id, sess.fsid)
    if err != nil {
        print("what the fuck")
    }
}

// Handler for building ugomenus for the front page
// recent, hot, most liked, etc..
func serveFrontPage(w http.ResponseWriter, r *http.Request) {
    
    base := ugoNew()
    base.setLayout(2)

    pt := r.URL.Query().Get("mode")
    pq := r.URL.Query().Get("page")

    p, err := strconv.Atoi(pq)
    if pq == "" {
        // do NOT print error message if the query is empty
        p = 1
    } else if err != nil {
        infolog.Printf("%v passed invalid page to %v%v: %v", r.Header.Get("X-Real-Ip"), r.Host, r.URL.Path, err)
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    flipnotes, total, err := getFrontFlipnotes(pt, p)
    if err != nil {
        errorlog.Printf("could not get flipnotes: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    pm := countPages(total)

    // meta
    base.setTopScreenText("Feed", fmt.Sprintf("Page %d / %d", p, pm), "","","")
    base.addDropdown(fmt.Sprintf("%s/ds/v2-xx/feed.uls?mode=%s&page=1", configuration.ServerUrl, pt), prettyPageTypes[pt], true)

    if p > 1 {
        base.addButton(fmt.Sprintf("%s/ds/v2-xx/feed.uls?mode=%s&page=&d", configuration.ServerUrl, pt, p-1), 100, "Previous")
    }

    for _, f := range flipnotes {
//      lock := btoi(f.lock)
        tempTmb, err := f.TMB()
        if err != nil {
            errorlog.Printf("nil tmb: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        base.addButton(fmt.Sprintf("%s/ds/v2-xx/movie/%d.ppm", configuration.ServerUrl, f.id), 3, "", f.stars["yellow"], 765, 573, 0)

        base.EmbedBytes = append(base.EmbedBytes, tempTmb)
        //fmt.Printf("debug: length of tmb %v is %v\n", n, len(tempTmb))
    }

    if pm > p {
        base.addButton(fmt.Sprintf("%s/ds/v2-xx/feed.uls?mode=%s&page=%d", configuration.ServerUrl, pt, p+1), 100, "Next")
    }

    data := base.pack(mux.Vars(r)["reg"])
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

func handleEulaTsv(w http.ResponseWriter, r *http.Request) {
    w.Write(append(encUTF16LE("English"), []byte("\ten")...))
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
    an := base64.StdEncoding.EncodeToString(decUTF16LE(ppmBody[0x40 : 0x56]))
    paid := strings.ToUpper(hex.EncodeToString(reverse(ppmBody[0x56 : 0x5E])))
    pan := base64.StdEncoding.EncodeToString(decUTF16LE(ppmBody[0x2A : 0x40]))
    l := int(ppmBody[0x10])
    afn := strings.ToUpper(hex.EncodeToString(ppmBody[0x78 : 0x7B])) + "_" +
                string(ppmBody[0x7B : 0x88]) + "_" +
                editCountPad(binary.LittleEndian.Uint16(ppmBody[0x88 : 0x90]))

//  debuglog.Printf("received ppm body from %v %v %v", session.fsid, session.username, afn)

    if ok, err := checkMovieExistsAfn(afn); ok && err == nil {
        w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
        w.Write(encUTF16LE("duplicate flipnote"))
        return
    } else if err != nil {
        errorlog.Printf("could not check if flipnote %v exists: %v", afn, err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    if err := db.QueryRow("INSERT INTO flipnotes (author_id, author_name, parent_author_id, parent_author_name, author_filename, lock) VALUES ($1, $2, $3, $4, $5, $6) RETURNING (id)", aid, an, paid, pan, afn, l).Scan(&id); err != nil {
        errorlog.Printf("failed to update database: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

//  fmt.Println(id)

    fp, err := os.OpenFile(configuration.HatenaDir + "/hatena_storage/flipnotes/" + fmt.Sprint(id) + ".ppm", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
    if err != nil {
        // >> store by id to not allow filename clashes
        // this is kinda stupid because filenames allow to identify
        // whether a flipnote has already been uploaded. However I think
        // an id-based system is better for querying flipnotes vs
        // very long, hard to remember filenames. This isn't
        // 2008 after all
        errorlog.Printf("failed to open path to ppm: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    defer fp.Close()

    if _, err := fp.Write(ppmBody); err != nil {
        errorlog.Printf("failed to write ppm to file: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    infolog.Printf("%v (%v) uploaded flipnote %v", session.username, session.fsid, afn)
    w.WriteHeader(http.StatusOK)
}

func retErrorHandler(code int) http.HandlerFunc {
    fn := func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(code)
    }

    return fn
}

func misc(w http.ResponseWriter, r *http.Request) {
    switch r.URL.Path {
    case "/ds/imagetest.htm":
        w.Write([]byte("<html><head><meta name=\"uppertitle\" content=\"big ol test\"></head><body><img src=\"http://flipnote.hatena.com/images/ds/demo2.npf\" width=\"50\" height=\"50\" align=\"left\"><p>test</p></body></html>"))
    case "/ds/postreplytest.htm":
        w.Write([]byte("<html><head><meta name=\"replybutton\" content=\"http://flipnote.hatena.com/ds/test.reply\"></head><body><p>reply test</p></body></html>"))
    case "/ds/test.reply":
        w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
        w.Write(encUTF16LE("baka"))
    }

    return
}

func static(w http.ResponseWriter, r *http.Request) {
    file, err := os.ReadFile(configuration.HatenaDir + "/static" + r.URL.Path)
    if  err != nil {
        w.WriteHeader(http.StatusNotFound)
        return
    }

    w.Write(file)
    return
}


func cmd(r string) string {
    args := strings.Split(r, " ")

    switch args[0] {
    case "ban":
        if len(args) < 5 {
            return fmt.Sprintf("expected 5 arguments; got %d: %v", len(args), args)
        }
        return "big day"
    default:
        return "invalid command"
    }
}
