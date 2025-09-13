package main

import (
	"context"
	"io"
	"os"
	"time"

	"fmt"

	"net/http"

	"image"
	"image/color/palette"

	"github.com/KononK/resize"
	"github.com/esimov/colorquant"

	"github.com/gorilla/mux"

	"encoding/binary"
	"encoding/hex"
	"strconv"
	"strings"

	"floc/ugoserver/nx"
)

const (
    MSG_MOVIE_RATELIMIT string = "rate limit message: %s"
)


//
// MOVIES
//

// movieHandler handler is responsible for returning .ppm files, building web pages for viewing
// a movie's details and comments, and returning a few bytes on .info requests
func (e *env) movieHandler(w http.ResponseWriter, r *http.Request) {

    vars := mux.Vars(r)

    id, err := strconv.Atoi(vars["movieid"])
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    ext := vars["ext"]
    
    sid := r.Header.Get("X-Dsi-Sid")

    switch ext {
    case "dl":
        err := updateDlCount(e.pool, id)
        if err != nil {
            errorlog.Printf("while updating dl count: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusOK)
        return

    case "delete":
        err := deleteMovie(e.pool, id)
        if err != nil {
            errorlog.Printf("while deleting %v: %v", id, err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusOK)
        return

    case "ppm":
        data, err := os.ReadFile(fmt.Sprintf("%s/movies/%d.ppm", e.cnf.StoreDir, id))
        if err != nil {
            w.WriteHeader(http.StatusNotFound)
            return
        }

        w.Write(data)
        //log.Printf("sent %d bytes to %v", len(data), r.Header.Get("X-Real-Ip"))
        return

    case "htm":
        // make it return a 404 if not found
        movie, err := getMovieById(e.pool, id)
        if err != nil {
            switch err {
            case ErrNoMovie:
                w.WriteHeader(http.StatusNotFound)
                return
            default:
                errorlog.Printf("while getting flipnote %v: %v", id, err)
                w.WriteHeader(http.StatusInternalServerError)
                return
            }
        }
        
        d, err := e.fillpage(sid)
        if err != nil {
            errorlog.Printf("while filling DSPage: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        
        d["movie"] = movie
        au, err := getUserById(e.pool, movie.AuUserID)
        if err != nil {
            errorlog.Printf("while getting user %d (movie view): %v", movie.AuUserID, err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        d["author"] = au

        if err = e.html.ExecuteTemplate(w, "movie.html", d); err != nil {
            errorlog.Printf("while executing template: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }

    case "info":
        w.Write([]byte{0x30, 0x0A, 0x30, 0x0A}) // write 0\n0\n because flipnote is weird

    default:
        w.WriteHeader(http.StatusNotFound)
        return
    }
}

// moviePost handler posts a movie to a channel
func (e *env) moviePost(w http.ResponseWriter, r *http.Request) {
    
    chq := r.URL.Query().Get("ch")
    ch, err := strconv.Atoi(chq)
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    // validation is done by middleware
    s := e.sessions[r.Header.Get("X-Dsi-Sid")]
    
    if d, err := getUserMovieRatelimit(e.pool, s.UserID); err != nil {
        errorlog.Printf("while checking user ratelimit: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    } else if d != nil {
        w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
        w.Write(encUTF16LE(fmt.Sprintf(MSG_MOVIE_RATELIMIT, time.Until(*d).String() )))
        return
    }

    ppm, err := io.ReadAll(r.Body)
    if err != nil {
        errorlog.Printf("while reading ppm from request body: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    
    bt, _ := time.Parse(time.DateTime, "2000-01-01 00:00:00")
    
    nm := Movie{
        ChannelID: ch,

        AuUserID: s.UserID,
        AuFSID: strings.ToUpper(hex.EncodeToString(reverse(ppm[0x5E : 0x66]))),
        AuName: string(stripnull(decUTF16LE(ppm[0x40 : 0x56]))),
        AuFN: strings.ToUpper(hex.EncodeToString(ppm[0x78 : 0x7B])) + "_" + string(ppm[0x7B : 0x88]) + "_" + editCountPad(binary.LittleEndian.Uint16(ppm[0x88 : 0x90])), // long ahh
        
        OGAuFSID: strings.ToUpper(hex.EncodeToString(reverse(ppm[0x8A : 0x92]))),
        OGAuName: string(stripnull(decUTF16LE(ppm[0x14 : 0x2A]))),
        OGAuFNFrag: strings.ToUpper(hex.EncodeToString(ppm[0x92 : 0x95])) + "_" + strings.ToUpper(hex.EncodeToString(ppm[0x95 : 0x9A])), // not as long ahh
        
        LastMod: bt.Add(time.Duration(binary.LittleEndian.Uint32(ppm[0x9A : 0x9E])) * time.Second), // mess
        
        Lock: itob(int(ppm[0x10])),
    }

    if nm.AuFSID != s.FSID {
        warnlog.Printf("%s (%d) tried to upload flipnote with differing FSID", s.Username, s.UserID)
        w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
        w.Write(encUTF16LE("an error occurred"))
        return
    }
    if !fsid_match.MatchString(nm.AuFSID) || !fn_match.MatchString(nm.AuFN) {
        warnlog.Printf("%s (%d) tried to upload malformed movie", s.Username, s.UserID)
        w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
        w.Write(encUTF16LE("an error occurred"))
        return
    }

//  debuglog.Printf("received ppm body from %v %v %v", session.fsid, session.username, afn)
    tx, _ := e.pool.Begin(context.Background())
    defer tx.Commit(context.Background())

    id, err := addMovie(tx, nm)
    if err != nil {
        switch err {
        case ErrMovieExists:
            w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
            w.Write(encUTF16LE("this flipnote has\nalready been uploaded"))
            return
        default:
            errorlog.Printf("while adding flipnote: %v", err)
            debuglog.Println(nm)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
    }

//  fmt.Println(id)

    fp, err := os.OpenFile(fmt.Sprintf("%s/movies/%d.ppm", e.cnf.StoreDir, id), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
    if err != nil {
        // >> store by id to not allow filename clashes
        // this isn't really an issue and i was being dumb because
        // all filenames are unique. But i like this more
        tx.Rollback(context.Background())
        infolog.Printf("transaction rollback")

        errorlog.Printf("while opening path to ppm: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    defer fp.Close()

    if _, err := fp.Write(ppm); err != nil {
        tx.Rollback(context.Background())
        infolog.Printf("transaction rollback")

        errorlog.Printf("while writing ppm to file: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    
    infolog.Printf("%v (%v) uploaded flipnote %v", s.Username, s.FSID, nm.AuFN)
    w.WriteHeader(http.StatusOK)
}

// starMovie handler updates star counts for movies
func (e *env) starMovie(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    s := e.sessions[r.Header.Get("X-Dsi-Sid")]
    id, err := strconv.Atoi(vars["movieid"])
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    count, err := strconv.Atoi(r.Header.Get("X-Hatena-Star-Count"))
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    color, ok := vars["color"]
    if !ok {
        color = "yellow"
    }
    
    if err := updateMovieStars(e.pool, s.UserID, id, color, count); err != nil {
        errorlog.Printf("while updating star count for %d (user %d): %v", id, s.UserID, err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
}

// movieFeed handler returns a menu with the movies in the main feed
func (e *env) movieFeed(w http.ResponseWriter, r *http.Request) {

    // new ugomenu
    base := newMenu()
    base.setLayout(2)

    // url query
    sort := r.URL.Query().Get("s")
    pq := r.URL.Query().Get("p")

    // strings from query to int
    // check page
    p, err := strconv.Atoi(pq)
    if pq == "" {
        p = 1
    } else if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    // get movies
    flipnotes, total, err := getFrontMovies(e.pool, sort, p)
    if err != nil {
        errorlog.Printf("while getting feed flipnotes: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    pm := countPages(total, 50)

    // start adding stuff to the menu
    base.setTopScreenText("Feed", fmt.Sprintf("%d flipnotes", total), fmt.Sprintf("Page %d/%d", p, pm), "","")
    //base.addDropdown(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/feed.uls?s=%s&p=1", mode), mode, true)
    for _, sn := range []string{"hot", "top", "new"} {
        if sn == sort {
            base.addDropdown(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/feed.uls?s=%s&p=1", sn), sn, true)
        } else {
            base.addDropdown(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/feed.uls?s=%s&p=1", sn), sn, false)
        }
    }

    // back button
    if p > 1 {
        base.addButton(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/feed.uls?s=%s&p=%d", sort, p-1), 100, "Previous")
    }

    for _, f := range flipnotes {
//      lock := btoi(f.lock)
        t, err := tmb(e.cnf.Root, f.ID)
        if err != nil {
            errorlog.Printf("nil tmb: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        base.addButton(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/movie/%d.ppm", f.ID), 3, "", f.Stars[0], 765, 573, 0)

        base.addEmbed(t)
        //fmt.Printf("debug: length of tmb %v is %v\n", n, len(tempTmb))
    }

    // forward button
    if pm > p {
        base.addButton(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/feed.uls?s=%s&p=%d", sort, p+1), 100, "Next")
    }

    data := e.pack(*base, e.sessions[r.Header.Get("X-Dsi-Sid")].Region)
    w.Write(data)
}



// 
// COMMENTS/REPLIES
//

func (e *env) replyHandler(w http.ResponseWriter, r *http.Request) {
    sid := r.Header.Get("X-Dsi-Sid")
    //s := e.sessions[sid]
    vars := mux.Vars(r)

    switch vars["ext"] {
    case "npf":
        id, err := strconv.Atoi(vars["commentid"])
        if err != nil {
            w.WriteHeader(http.StatusNotFound)
            return
        }
        
        // get the file
        npf, err := os.ReadFile(fmt.Sprintf("%s/comments/%d.npf", e.cnf.StoreDir, id))
        if err != nil {
            errorlog.Printf("while reading reply %d file: %v", id, err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        
        w.Write(npf)
        
    case "htm":
        id, err := strconv.Atoi(vars["movieid"])
        if err != nil {
            w.WriteHeader(http.StatusBadRequest)
            return
        }
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
        
        movie, err := getMovieById(e.pool, id)
        if err != nil {
            w.WriteHeader(http.StatusNotFound)
        }
        
        comments, err := getMovieComments(e.pool, id, p)
        if err != nil {
            errorlog.Printf("while getting comments for flipnote %v: %v", id, err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        
        d, err := e.fillpage(sid)
        if err != nil {
            errorlog.Printf("while filling out DSPage template: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        
        d["movie"] = movie
        d["comments"] = comments
        
        if err = e.html.ExecuteTemplate(w, "comment.html", d); err != nil {
            errorlog.Printf("while executing template: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
    }
}

// replyPost handler handles requests for .npf (reply image data) and .reply (commenting on a movie)
func (e *env) replyPost(w http.ResponseWriter, r *http.Request) {
    s := e.sessions[r.Header.Get("X-Dsi-Sid")]
    v := mux.Vars(r)

    movieid, err := strconv.Atoi(v["movieid"])
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    // for now only memo replies from the ds
    reply, err := io.ReadAll(r.Body)
    if err != nil {
        errorlog.Printf("while reading reply body: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    im, err := nx.FromPpm(reply)
    if err != nil {
        errorlog.Printf("while decoding reply: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    // only one page flipnotes should be posted
    // this is in case of a bad actor manually POSTing a custom ppm
    if len(im) != 1 {
        errorlog.Printf("multiple frames in comment memo from %v", s.UserID)
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    
    // Convert to npf
    // Quantizer is needed because while downsizing the image it introduces lots of other colors
    src := resize.Resize(64, 48, im[0], resize.NearestNeighbor)
    dst := image.NewPaletted(src.Bounds(), palette.WebSafe)

    colorquant.NoDither.Quantize(src, dst, 15, false, true)

    npf, err := nx.ToNpf(dst)
    if err != nil {
        errorlog.Printf("while converting reply to npf: %v", err)
        return
    }
    
    // undo if can't write file
    tx, _ := e.pool.Begin(context.Background())
    defer tx.Commit(context.Background())

    id, err := addMovieCommentMemo(tx, s.UserID, movieid)
    if err != nil {
        errorlog.Printf("while adding movie reply to database: %v", err)
        return
    }
    
    fp, err := os.OpenFile(fmt.Sprintf("%s/comments/%d.npf", e.cnf.StoreDir, id), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
    if err != nil {
        tx.Rollback(context.Background())
        infolog.Printf("transaction rollback")

        errorlog.Printf("while opening path to reply npf: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    defer fp.Close()

    if _, err := fp.Write(npf); err != nil {
        tx.Rollback(context.Background())
        infolog.Printf("transaction rollback")

        errorlog.Printf("while writing reply npf to file: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusOK)
}



//
// CHANNELS
//

// movieChannelFeed handler is mostly the same as movieFeed, but it queries
// only a specific channel instead of all of them --
// integrated into movieFeed
func (e *env) movieChannelFeed(w http.ResponseWriter, r *http.Request) {
    
    s := e.sessions[r.Header.Get("X-Dsi-Sid")]

    // new ugomenu
    base := newMenu()
    base.setLayout(2)

    // url query
    sort := r.URL.Query().Get("s")
    pq := r.URL.Query().Get("p")
    chq := r.URL.Query().Get("ch")

    // strings from query to int
    // check page
    p, err := strconv.Atoi(pq)
    if pq == "" {
        p = 1
    } else if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    // get channel
    chid, err := strconv.Atoi(chq)
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    chs, chl, err := getChannelInfo(e.pool, chid)
    if err != nil {
        errorlog.Printf("while getting channel %d info: %v", chid, err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    // get movies
    flipnotes, total, err := getChannelMovies(e.pool, chid, sort, p)
    if err != nil {
        errorlog.Printf("while getting channel flipnotes: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    pm := countPages(total, 50)

    // start adding stuff to the menu
    base.setTopScreenText(chs, fmt.Sprintf("%d flipnotes", total), fmt.Sprintf("Page %d/%d", p, pm), "", chl)
    if s.IsLoggedIn {
        base.addCorner(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/flipnote.post?ch=%d", chid), "Post flipnote")
    }
    //base.addDropdown(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/feed.uls?s=%s&p=1", mode), mode, true)
    for _, sn := range []string{"hot", "top", "new"} {
        if sn == sort {
            base.addDropdown(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/channel.uls?ch=%d&s=%s&p=1", chid, sn), sn, true)
        } else {
            base.addDropdown(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/channel.uls?ch=%d&s=%s&p=1", chid, sn), sn, false)
        }        
    }

    // back button
    if p > 1 {
        base.addButton(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/channel.uls?ch=%d&s=%s&p=%d", chid, sort, p-1), 100, "Previous")
    }

    for _, f := range flipnotes {
//      lock := btoi(f.lock)
        t, err := tmb(e.cnf.Root, f.ID)
        if err != nil {
            errorlog.Printf("nil tmb: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        base.addButton(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/movie/%d.ppm", f.ID), 3, "", f.Stars[0], 765, 573, 0)

        base.addEmbed(t)
        //fmt.Printf("debug: length of tmb %v is %v\n", n, len(tempTmb))
    }

    // forward button
    if pm > p {
        base.addButton(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/channel.uls?ch=%d&s=%s&p=%d", chid, sort, p+1), 100, "Next")
    }

    data := e.pack(*base, s.Region)
    w.Write(data)
}

func (e *env) channelMainMenu(w http.ResponseWriter, r *http.Request) {

    menu := newMenu()
    menu.setLayout(3, 4)
    
    // get first 8 channels
    chs, err := getChannelList(e.pool, 0)
    if err != nil {
        errorlog.Printf("while getting main channels: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    
    for _, ch := range chs {
        menu.addButton(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/channel.uls?ch=%d&s=hot&p=1", ch.ID), 100, ch.Name)
    }
    
    data := e.pack(*menu, e.sessions[r.Header.Get("X-Dsi-Sid")].Region)
    w.Write(data)
}