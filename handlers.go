package main

import (
	"io"
	"os"

	"fmt"

	"net/http"

	"github.com/gorilla/mux"

	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"strconv"
	"strings"
)

// Not my finest code up there so we're doing this a better way
func movieHandler(w http.ResponseWriter, r *http.Request) {

    vars := mux.Vars(r)

    id, err := strconv.Atoi(vars["movieid"])
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    ext := vars["ext"]

    path := fmt.Sprintf("/ds/%s/movie/%d", vars["reg"], id)

    switch ext {
    case "dl":
        err := updateViewDlCount(id, ext)
        if err != nil {
            errorlog.Printf("failed to update %v count: %v", ext, err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusOK)
        return

    case "delete":
        err := deleteMovie(id)
        if err != nil {
            errorlog.Printf("failed to delete %v: %v", id, err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusOK)
        return

    case "ppm":
        data, err := os.ReadFile(fmt.Sprintf("%s/movies/%d.ppm", cnf.StoreDir, id))
        if err != nil {
            w.WriteHeader(http.StatusNotFound)
            return
        }

        err = updateViewDlCount(id, ext)
        if err != nil {
            errorlog.Printf("failed to update %v count: %v", ext, err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        w.Write(data)
//          log.Printf("sent %d bytes to %v", len(data), r.Header.Get("X-Real-Ip"))
        return

    case "htm":
        // make it return a 404 if not found
        movie, err := getMovieSingle(id)
        if err != nil {
            errorlog.Printf("could not get flipnote %v: %v", id, err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }

        w.Write([]byte(fmt.Sprintf("<html><head><meta name=\"upperlink\" content=\"%s\"><meta name=\"playcontrolbutton\" content=\"1\"><meta name=\"savebutton\" content=\"%s\"><meta name=\"starbutton\" content=\"%s\"><meta name=\"starbutton1\" content=\"%s\"><meta name=\"starbutton2\" content=\"%s\"><meta name=\"starbutton3\" content=\"%s\"><meta name=\"starbutton4\" content=\"%s\"><meta name=\"deletebutton\" content=\"%s\"></head><body><p>wip<br>obviously this would be unfinished<br>yellow <span class=\"star0\">%d</span><br>green <span class=\"star1\">%d</span><br>red <span class=\"star2\">%d</span><br>blue <span class=\"star3\">%d</span><br>purple <span class=\"star4\">%d</span><br><br>debug:<br>id: %v</p></body></html>", cnf.URL+path+".ppm", cnf.URL+path+".ppm", cnf.URL+path+".star",cnf.URL+path+".star/green,99",cnf.URL+path+".star/red,99",cnf.URL+path+".star/blue,99",cnf.URL+path+".star/purple,99", cnf.URL+path+".delete", movie.ys, movie.gs, movie.rs, movie.bs, movie.ps, id)))
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
    s := sessions[r.Header.Get("X-Dsi-Sid")]
    id, err := strconv.Atoi(vars["movieid"])
    if err != nil {
        errorlog.Printf("bad movieid when adding star: %v", err)
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    count, err := strconv.Atoi(r.Header.Get("X-Hatena-Star-Count"))
    if err != nil {
        errorlog.Printf("bad star count from %d: %v", s.userid, err)
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    color, ok := vars["color"]
    if !ok {
        color = "yellow"
    }
    
    if err := updateMovieStars(s.userid, id, color, count); err != nil {
        errorlog.Printf("failed to update star count for %d (user %d): %v", id, s.userid, err)
        w.WriteHeader(http.StatusInternalServerError)
        return
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

    flipnotes, total, err := getFrontMovies(pt, p)
    if err != nil {
        errorlog.Printf("could not get flipnotes: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    pm := countPages(total)

    // meta
    base.setTopScreenText("Feed", fmt.Sprintf("Page %d / %d", p, pm), "","","")
    base.addDropdown(fmt.Sprintf("%s/ds/v2-xx/feed.uls?mode=%s&page=1", cnf.URL, pt), prettyPageTypes[pt], true)

    if p > 1 {
        base.addButton(fmt.Sprintf("%s/ds/v2-xx/feed.uls?mode=%s&page=%d", cnf.URL, pt, p-1), 100, "Previous")
    }

    for _, f := range flipnotes {
//      lock := btoi(f.lock)
        tempTmb, err := f.TMB()
        if err != nil {
            errorlog.Printf("nil tmb: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        base.addButton(fmt.Sprintf("%s/ds/v2-xx/movie/%d.ppm", cnf.URL, f.id), 3, "", f.ys, 765, 573, 0)

        base.EmbedBytes = append(base.EmbedBytes, tempTmb)
        //fmt.Printf("debug: length of tmb %v is %v\n", n, len(tempTmb))
    }

    if pm > p {
        base.addButton(fmt.Sprintf("%s/ds/v2-xx/feed.uls?mode=%s&page=%d", cnf.URL, pt, p+1), 100, "Next")
    }

    data := base.pack(sessions[r.Header.Get("X-Dsi-Sid")].getregion())
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
    
    text, err := os.ReadFile(cnf.Dir + "/static/txt/" + txt + ".txt")
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

    // validation is done by middleware
    s := sessions[r.Header.Get("X-Dsi-Sid")]

    ppmBody, err := io.ReadAll(r.Body)
    if err != nil {
        errorlog.Printf("failed to read ppm from POST request body! %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    fsid := strings.ToUpper(hex.EncodeToString(reverse(ppmBody[0x5E : 0x66])))
    name := base64.StdEncoding.EncodeToString(decUTF16LE(ppmBody[0x40 : 0x56]))
    l := int(ppmBody[0x10])
    fn := strings.ToUpper(hex.EncodeToString(ppmBody[0x78 : 0x7B])) + "_" +
                string(ppmBody[0x7B : 0x88]) + "_" +
                editCountPad(binary.LittleEndian.Uint16(ppmBody[0x88 : 0x90]))

//  debuglog.Printf("received ppm body from %v %v %v", session.fsid, session.username, afn)

    id, err := addMovie(s.userid, fsid, name, fn, l)
    if err == ErrMovieExists {
        w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
        w.Write(encUTF16LE("this flipnote has\nalready been uploaded"))
        return
    } else if err != nil {
        errorlog.Printf("could not add flipnote: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

//  fmt.Println(id)

    fp, err := os.OpenFile(cnf.StoreDir + "/movies/" + fmt.Sprint(id) + ".ppm", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
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

    infolog.Printf("%v (%v) uploaded flipnote %v", qd(s.username), s.fsid, fn)
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
}

func static(w http.ResponseWriter, r *http.Request) {
    file, err := os.ReadFile(cnf.Dir + "/static" + r.URL.Path)
    if  err != nil {
        w.WriteHeader(http.StatusNotFound)
        return
    }

    w.Write(file)
}

func jump(w http.ResponseWriter, r *http.Request) {
    w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
    w.Write(encUTF16LE("bazinga"))
}

func debug(w http.ResponseWriter, r *http.Request) {
    sid := r.Header.Get("X-Dsi-Sid")
    s := sessions[sid]

    w.Write([]byte(fmt.Sprintf("<html><head><meta name=\"uppertitle\" content=\"debug haha\"></head><body>This is debug menu<br>sid: %s<br>fsid: %s<br>ip: %s<br>username: %s<br>session issued: %s<br><br>userid: %d<br>is_unregistered: %t<br>is_logged_in: %t<br><br><a href=\""+cnf.URL+"/ds/v2-"+s.getregion()+"/sa/login.htm\">log in</a></body></html>", s.sid, s.fsid, s.ip, qd(s.username), s.issued.String(), s.userid, s.is_unregistered, s.is_logged_in)))
}