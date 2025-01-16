package main

import (
	"io"
	"os"

	"fmt"

	"net/http"

	"github.com/gorilla/mux"
	"github.com/nfnt/resize"

	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"strconv"
	"strings"

	"floc/ugoserver/img"
)

var (
    prettyPageTypes = map[string]string{"recent":"Recent"}
)

// Movie handler
// Responsible for updating view/download counts, deleting flipnotes,
// .info requests, returning ppm files, html overview
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
        cn, err := getMovieCommentsCount(id)
        if err != nil {
            errorlog.Printf("while getting comment count for flipnote %d: %v", id, err)
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
            
            if err = templates["comment"].Execute(w, CommentPage{
                Page: Page{
                    Root: cnf.Root,
                    Region: s.getregion(),
                    LoggedIn: s.is_logged_in,
                },
                Comments: comments,
                CommentCount: cn,
                MovieID: id,
            }); err != nil {
                errorlog.Printf("while executing template: %v", err)
                w.WriteHeader(http.StatusInternalServerError)
                return
            }
        }

        // otherwise the overview
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

        if err = templates["movie"].Execute(w, MoviePage{
            Page: Page{
                Root: cnf.Root,
                Region: s.getregion(),
                LoggedIn: s.is_logged_in,
            },
            Movie: movie,
            MovieAuthor: s.userid == movie.Au_userid,
            CommentCount: cn,
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


// Add stars to movie on request
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
    
    if err := updateMovieStars(s.userid, id, color, count); err != nil {
        errorlog.Printf("while updating star count for %d (user %d): %v", id, s.userid, err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
}


// Handler for building ugomenus for the feed
// recent, todo: hot, most liked, etc..
func movieFeed(w http.ResponseWriter, r *http.Request) {
    
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
        errorlog.Printf("while getting feed flipnotes: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    pm := countPages(total)

    // meta
    base.setTopScreenText("Feed", fmt.Sprintf("%d flipnotes", total), fmt.Sprintf("Page %d / %d", p, pm), "","")
    base.addDropdown(fmt.Sprintf("http://%s/ds/v2-xx/feed.uls?mode=%s&page=1", cnf.Root, pt), prettyPageTypes[pt], true)

    if p > 1 {
        base.addButton(fmt.Sprintf("http://%s/ds/v2-xx/feed.uls?mode=%s&page=%d", cnf.Root, pt, p-1), 100, "Previous")
    }

    for _, f := range flipnotes {
//      lock := btoi(f.lock)
        tempTmb, err := f.TMB()
        if err != nil {
            errorlog.Printf("nil tmb: %v", err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
        base.addButton(fmt.Sprintf("http://%s/ds/v2-xx/movie/%d.ppm", cnf.Root, f.ID), 3, "", f.Ys, 765, 573, 0)

        base.EmbedBytes = append(base.EmbedBytes, tempTmb)
        //fmt.Printf("debug: length of tmb %v is %v\n", n, len(tempTmb))
    }

    if pm > p {
        base.addButton(fmt.Sprintf("http://%s/ds/v2-xx/feed.uls?mode=%s&page=%d", cnf.Root, pt, p+1), 100, "Next")
    }

    data := base.pack(sessions[r.Header.Get("X-Dsi-Sid")].getregion())
    w.Write(data)
}


// Return text files in utf16
func eula(w http.ResponseWriter, r *http.Request) {

    vars := mux.Vars(r)
    txt := vars["txt"]

    text, err := os.ReadFile(cnf.Dir + "/static/txt/" + txt + ".txt")
    if err != nil {
        warnlog.Printf("failed to read %v: %v", txt, err)
        text = []byte("\n\nThis is a placeholder.\nYou shouldn't see this.")
    }

    w.Write(encUTF16LE(string(text)))
}

func eulatsv(w http.ResponseWriter, r *http.Request) {
    w.Write(append(encUTF16LE("English"), []byte("\ten")...))
}



// Movie post
// posts a flipnote
func moviePost(w http.ResponseWriter, r *http.Request) {

    // validation is done by middleware
    s := sessions[r.Header.Get("X-Dsi-Sid")]

    ppm, err := io.ReadAll(r.Body)
    if err != nil {
        errorlog.Printf("while reading ppm from request body: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }

    fsid := strings.ToUpper(hex.EncodeToString(reverse(ppm[0x5E : 0x66])))
    name := base64.StdEncoding.EncodeToString(decUTF16LE(ppm[0x40 : 0x56]))
    l := int(ppm[0x10])
    fn := strings.ToUpper(hex.EncodeToString(ppm[0x78 : 0x7B])) + "_" +
                string(ppm[0x7B : 0x88]) + "_" +
                editCountPad(binary.LittleEndian.Uint16(ppm[0x88 : 0x90]))

//  debuglog.Printf("received ppm body from %v %v %v", session.fsid, session.username, afn)

    id, err := addMovie(s.userid, fsid, name, fn, l)
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

    fp, err := os.OpenFile(cnf.StoreDir + "/movies/" + fmt.Sprint(id) + ".ppm", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
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

    infolog.Printf("%v (%v) uploaded flipnote %v", qd(s.username), s.fsid, fn)
    w.WriteHeader(http.StatusOK)
}

// simple function to return a status code
func returncode(code int) http.HandlerFunc {
    fn := func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(code)
    }

    return fn
}

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
            errorlog.Printf("multiple frames in comment memo from %v", s.userid)
            w.WriteHeader(http.StatusBadRequest)
            return
        }
        
        // Resize image
        scaled := resize.Resize(64, 48, im[0], resize.NearestNeighbor)
        
        // Convert to npf
        npf, err := img.ToNpf(scaled)
        if err != nil {
            errorlog.Printf("while converting reply to npf: %v", err)
        }
        
        id, err := addMovieReplyMemo(s.userid, movieid)
        if err != nil {
            errorlog.Printf("while adding movie reply to database: %v", err)
        }
        
        fp, err := os.OpenFile(cnf.StoreDir + "/comments/" + fmt.Sprint(id) + ".npf", os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
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

// todo: clean
func misc(w http.ResponseWriter, r *http.Request) {
    switch r.URL.Path {
    case "/ds/imagetest.htm":
        w.Write([]byte("<html><head><meta name=\"uppertitle\" content=\"big ol test\"></head><body><img src=\"http://flipnote.hatena.com/ds/v2-us/comment/1.npf\" width=\"64\" height=\"48\" align=\"left\"><p>test</p></body></html>"))
    case "/ds/postreplytest.htm":
        w.Write([]byte("<html><head><meta name=\"replybutton\" content=\"http://flipnote.hatena.com/ds/test.reply\"></head><body><p>reply test</p></body></html>"))
    case "/ds/test.reply":
        w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
        w.Write(encUTF16LE("baka"))
    case "/ds/car.htm":
        w.Write([]byte(`<html><head></head><body><img src="http://`+cnf.Root+`/images/ds/chr.ntft" width="50" height="50"></body</html>`))
    }
}

// Return files from the filesystem
// Dots and stuff *should* be filtered out by net/http, but
// if it becomes an issue I'll fix it
func static(w http.ResponseWriter, r *http.Request) {
    file, err := os.ReadFile(cnf.Dir + "/static" + r.URL.Path)
    if  err != nil {
        w.WriteHeader(http.StatusNotFound)
        return
    }

    w.Write(file)
}

// todo: refactor
func jump(w http.ResponseWriter, r *http.Request) {
    w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
    w.Write(encUTF16LE("bazinga"))
}

// todo: template
func debug(w http.ResponseWriter, r *http.Request) {
    sid := r.Header.Get("X-Dsi-Sid")
    s := sessions[sid]

    w.Write([]byte(fmt.Sprintf(`<html><head><link rel="stylesheet" type="text/css" href="http://`+cnf.Root+`/css/ds/basic.css"><meta name="uppertitle" content="debug haha"></head><body>This is debug menu<br>sid: %s<br>fsid: %s<br>ip: %s<br>username: %s<br>session issued: %s<br><br>userid: %d<br>is_unregistered: %t<br>is_logged_in: %t<br><br><a href="http://`+cnf.Root+`/ds/v2-`+s.getregion()+`/sa/login.htm">log in</a>|||<a href="http://`+cnf.Root+`/ds/v2-`+s.getregion()+`/sa/register.htm">register</a><br><br><a href="http://`+cnf.Root+`/ds/car.htm">click this</a></body></html>`, sid, s.fsid, s.ip, qd(s.username), s.issued.String(), s.userid, s.is_unregistered, s.is_logged_in)))
}