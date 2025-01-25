package main

import (
	"io"
	"os"

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

	"floc/ugoserver/img"
)

var (
    modes = map[string]string{"new": "Recent flipnotes"}
)

// movieHandler handler is responsible for returning .ppm files, building web pages for viewing
// a movie's details and comments, and returning a few bytes on .info requests
func movieHandler(w http.ResponseWriter, r *http.Request) {

    vars := mux.Vars(r)

    id, err := strconv.Atoi(vars["movieid"])
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    ext := vars["ext"]
    
    s := sessions[r.Header.Get("X-Dsi-Sid")]

    switch ext {
    case "dl":
        err := updateViewDlCount(id, ext)
        if err != nil {
            errorlog.Printf("while updating %v count: %v", ext, err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        w.WriteHeader(http.StatusOK)
        return

    case "delete":
        err := deleteMovie(id)
        if err != nil {
            errorlog.Printf("while deleting %v: %v", id, err)
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
            errorlog.Printf("while updating %v count: %v", ext, err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        w.Write(data)
        //log.Printf("sent %d bytes to %v", len(data), r.Header.Get("X-Real-Ip"))
        return

    case "htm":
        mode := r.URL.Query().Get("mode")

        // make it return a 404 if not found
        movie, err := getMovieSingle(id)
        if err == ErrNoMovie {
            w.WriteHeader(http.StatusNotFound)
            return
        } else if err != nil {
            errorlog.Printf("while getting flipnote %v: %v", id, err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }

        // comments
        if mode == "comment" {
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
            
            comments, err := getMovieComments(id, p)
            if err != nil {
                errorlog.Printf("while getting comments for flipnote %v: %v", id, err)
                w.WriteHeader(http.StatusInternalServerError)
                return
            }
            
            if err = templates["comment"].Execute(w, Page{
                Session: s,
                Root: cnf.Root,
                Region: s.getregion(),
                Movie: movie,
                Comments: comments,
            }); err != nil {
                errorlog.Printf("while executing template: %v", err)
                w.WriteHeader(http.StatusInternalServerError)
                return
            }
        }

        if err = templates["movie"].Execute(w, Page{
            Session: s,
            Root: cnf.Root,
            Region: s.getregion(),
            Movie: movie,
        }); err != nil {
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

// starMovie handler updates star counts for movies
func starMovie(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    s := sessions[r.Header.Get("X-Dsi-Sid")]
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
    
    if err := updateMovieStars(s.UserID, id, color, count); err != nil {
        errorlog.Printf("while updating star count for %d (user %d): %v", id, s.UserID, err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
}

// movieFeed handler returns a menu with the movies in the main feed
func movieFeed(w http.ResponseWriter, r *http.Request) {

    base := newMenu()
    base.setLayout(2)

    mode := r.URL.Query().Get("mode")
    pq := r.URL.Query().Get("page")

    p, err := strconv.Atoi(pq)
    if pq == "" {
        p = 1
    } else if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    flipnotes, total, err := getFrontMovies(mode, p)
    if err != nil {
        errorlog.Printf("while getting feed flipnotes: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    pm := countPages(total, 50)

    // TODO: other page modes ie most popular
    base.setTopScreenText("Feed", fmt.Sprintf("%d flipnotes", total), fmt.Sprintf("Page %d/%d", p, pm), "","")
    base.addDropdown(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/feed.uls?mode=%s&page=1", mode), modes[mode], true)

    if p > 1 {
        base.addButton(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/feed.uls?mode=%s&page=%d", mode, p-1), 100, "Previous")
    }

    for _, f := range flipnotes {
//      lock := btoi(f.lock)
        t, err := f.tmb()
        if err != nil {
            errorlog.Printf("nil tmb: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        base.addButton(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/movie/%d.ppm", f.ID), 3, "", f.Ys, 765, 573, 0)

        base.EmbedBytes = append(base.EmbedBytes, t)
        //fmt.Printf("debug: length of tmb %v is %v\n", n, len(tempTmb))
    }

    if pm > p {
        base.addButton(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/feed.uls?mode=%s&page=%d", mode, p+1), 100, "Next")
    }

    data := base.pack(sessions[r.Header.Get("X-Dsi-Sid")].getregion())
    w.Write(data)
}

// movieChannelFeed handler is mostly the same as movieFeed, but it queries
// only a specific channel instead of all of them
func movieChannelFeed(w http.ResponseWriter, r *http.Request) {
    
    base := newMenu()
    base.setLayout(2)

    mode := r.URL.Query().Get("mode")
    idq := r.URL.Query().Get("id")
    pq := r.URL.Query().Get("page")

    p, err := strconv.Atoi(pq)
    if pq == "" {
        // do NOT print error message if the query is empty
        p = 1
    } else if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    id, err := strconv.Atoi(idq)
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    
    ds, dl, err := getChannelInfo(id)
    if err != nil {
        errorlog.Printf("white getting channel description: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    flipnotes, total, err := getChannelMovies(id, mode, p)
    if err != nil {
        errorlog.Printf("while getting feed flipnotes: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    pm := countPages(total, 50)

    // meta
    base.setTopScreenText(ds, fmt.Sprintf("%d flipnotes", total), fmt.Sprintf("Page %d/%d", p, pm), "", dl)
    base.addDropdown(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/channel.uls?id=%d&mode=%s&page=1", id, mode), modes[mode], true)
    base.addCorner(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/flipnote.post?channel=%d", id), "Post flipnote")

    if p > 1 {
        base.addButton(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/channel.uls?id=%d&mode=%s&page=%d", id, mode, p-1), 100, "Previous")
    }

    for _, f := range flipnotes {
//      lock := btoi(f.lock)
        t, err := f.tmb()
        if err != nil {
            errorlog.Printf("nil tmb: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        base.addButton(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/movie/%d.ppm", f.ID), 3, "", f.Ys, 765, 573, 0)

        base.EmbedBytes = append(base.EmbedBytes, t)
        //fmt.Printf("debug: length of tmb %v is %v\n", n, len(tempTmb))
    }

    if pm > p {
        base.addButton(fmt.Sprintf("http://flipnote.hatena.com/ds/v2-xx/channel.uls?id=%d&mode=%s&page=%d", id, mode, p+1), 100, "Next")
    }

    data := base.pack(sessions[r.Header.Get("X-Dsi-Sid")].getregion())
    w.Write(data)
}

// eula handler returns text files in static/txt as utf16le
func eula(w http.ResponseWriter, r *http.Request) {

    vars := mux.Vars(r)
    txt := vars["txt"]

    text, err := os.ReadFile(fmt.Sprintf("%s/static/txt/%s.txt", cnf.Dir, txt))
    if err != nil {
        warnlog.Printf("failed to read %v: %v", txt, err)
        text = []byte("\n\nThis is a placeholder.\nYou shouldn't see this.")
    }

    w.Write(encUTF16LE(string(text)))
}

// eulatsv handler returns the eula_list.tsv required by eu versions of flipnote
func eulatsv(w http.ResponseWriter, r *http.Request) {
    w.Write(append(encUTF16LE("English"), []byte("\ten")...))
}

// moviePost handler posts a movie to a channel
func moviePost(w http.ResponseWriter, r *http.Request) {
    
    chq := r.URL.Query().Get("channel")
    ch, err := strconv.Atoi(chq)
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    // validation is done by middleware
    s := sessions[r.Header.Get("X-Dsi-Sid")]

    ppm, err := io.ReadAll(r.Body)
    if err != nil {
        errorlog.Printf("while reading ppm from request body: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    fsid := strings.ToUpper(hex.EncodeToString(reverse(ppm[0x5E : 0x66])))
    name := string(stripnull(decUTF16LE(ppm[0x40 : 0x56])))
    l := int(ppm[0x10])
    fn := strings.ToUpper(hex.EncodeToString(ppm[0x78 : 0x7B])) + "_" +
                string(ppm[0x7B : 0x88]) + "_" +
                editCountPad(binary.LittleEndian.Uint16(ppm[0x88 : 0x90]))

//  debuglog.Printf("received ppm body from %v %v %v", session.fsid, session.username, afn)

    id, err := addMovie(s.UserID, fsid, name, fn, l, ch)
    if err == ErrMovieExists {
        w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
        w.Write(encUTF16LE("this flipnote has\nalready been uploaded"))
        return
    } else if err != nil {
        errorlog.Printf("while adding flipnote: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

//  fmt.Println(id)

    fp, err := os.OpenFile(fmt.Sprintf("%s/movies/%d.ppm", cnf.StoreDir, id), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
    if err != nil {
        // >> store by id to not allow filename clashes
        // this isn't really an issue and i was being dumb because
        // all filenames are unique. But i like this more
        errorlog.Printf("while opening path to ppm: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    defer fp.Close()

    if _, err := fp.Write(ppm); err != nil {
        errorlog.Printf("while writing ppm to file: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    infolog.Printf("%v (%v) uploaded flipnote %v", qd(s.Username), s.FSID, fn)
    w.WriteHeader(http.StatusOK)
}

// movieReply handler handles requests for .npf (reply image data) and .reply (replying to a movie)
func movieReply(w http.ResponseWriter, r *http.Request) {
    s := sessions[r.Header.Get("X-Dsi-Sid")]
    v := mux.Vars(r)

    switch v["ext"] {
    case "npf":
        commentid, err := strconv.Atoi(v["commentid"])
        if err != nil {
            w.WriteHeader(http.StatusNotFound)
            return
        }
        
        // get the file
        npf, err := os.ReadFile(fmt.Sprintf("%s/comments/%d.npf", cnf.StoreDir, commentid))
        if err != nil {
            errorlog.Printf("while reading reply %d file: %v", commentid, err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        
        w.Write(npf)

    case "reply":
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

        im, err := img.FromPpm(reply)
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

        npf, err := img.ToNpf(dst)
        if err != nil {
            errorlog.Printf("while converting reply to npf: %v", err)
            return
        }
        
        id, err := addMovieReplyMemo(s.UserID, movieid)
        if err != nil {
            errorlog.Printf("while adding movie reply to database: %v", err)
            return
        }
        
        fp, err := os.OpenFile(fmt.Sprintf("%s/comments/%d.npf", cnf.StoreDir, id), os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
        if err != nil {
            errorlog.Printf("while opening path to reply npf: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }

        defer fp.Close()

        if _, err := fp.Write(npf); err != nil {
            errorlog.Printf("while writing reply npf to file: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        
        w.WriteHeader(http.StatusOK)
    }
}

// misc handler is here for minor things that need to return something, but don't necessarily matter
func misc(w http.ResponseWriter, r *http.Request) {
    switch r.URL.Path {
    case "/ds/notices.lst":
        w.Write([]byte{0x00})
    case "/ds/eula.txt":
        w.Write(encUTF16LE(MSG_NO_SUPPORT))
    case "/ds/confirm/upload.txt":
        w.Write([]byte{0x00, 0x00})
    case "/ds/confirm/delete.txt":
        w.Write([]byte{0x00, 0x00})
    case "/ds/confirm/download.txt":
        w.Write([]byte{0x00, 0x00})

    case "/ds/redirect.htm":
        w.Write([]byte(`<html><head></head><body>works</body</html>`))
    }
}

// static handler returns the file from cnf.Dir/static/path
func static(w http.ResponseWriter, r *http.Request) {
    file, err := os.ReadFile(fmt.Sprintf("%s/static/%s", cnf.Dir, r.URL.Path))
    if  err != nil {
        w.WriteHeader(http.StatusNotFound)
        return
    }

    w.Write(file)
}

// TODO
func jump(w http.ResponseWriter, r *http.Request) {
    w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
    w.Write(encUTF16LE("bazinga"))
}

// debug handler just runs a template
func debug(w http.ResponseWriter, r *http.Request) {
    sid := r.Header.Get("X-Dsi-Sid")
    s := sessions[sid]

    if err := templates["debug"].Execute(w, Page{
        Session: s,
        Root: cnf.Root,
        Region: s.getregion(),
        SID: sid,
    }); err != nil {
        errorlog.Printf("while executing template: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
    }
}