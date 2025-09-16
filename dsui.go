package main

import (
	"fmt"
	"strconv"

	"net/http"

	"github.com/gorilla/mux"
)

func (e *env) eula(w http.ResponseWriter, r *http.Request) {

    name := mux.Vars(r)["txt"]
    
    t, ok := e.assets[fmt.Sprintf("text/%s", name)]
    if !ok {
        warnlog.Printf("eula: couldn't find %s in assets", name)
        t = []byte("placeholder")
    }

    w.Write(encUTF16LE(t))
}

// eulatsv handler returns the eula_list.tsv required by eu versions of flipnote
func eulatsv(w http.ResponseWriter, r *http.Request) {
    w.Write(append(encUTF16LE("English"), []byte{'\t', 'e', 'n'}...))
}

// movie ui
func (e *env) movieui(w http.ResponseWriter, r *http.Request) {
    sid := r.Header.Get("X-Dsi-Sid")

    id, err := strconv.Atoi(mux.Vars(r)["movieid"])
    if err != nil {
        w.WriteHeader(http.StatusBadRequest)
        return
    }

    // make it return a 404 if not found
    movie, err := getMovieByIdUpdView(e.pool, id)
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
    
    jc, err := getCodeByRes(e.pool, "movie", movie.ID)
    if err != nil {
        errorlog.Printf("while getting jump code for movie %d: %v", movie.ID, err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    d["jumpcode"] = jumpasciitonds(jc)

    if err = e.html.ExecuteTemplate(w, "movie.html", d); err != nil {
        errorlog.Printf("while executing template: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
}


func (e *env) jump(w http.ResponseWriter, r *http.Request) {
    s := e.sessions[r.Header.Get("X-Dsi-Sid")]
    jc := r.URL.Query().Get("command")
    if !jump_match.MatchString(jc) {
        infolog.Printf("%s queried invalid jumpcode", s.Username)
        w.WriteHeader(http.StatusBadRequest)
        return
    }
    
    t, id, err := getResByCode(e.pool, jc)
    if err != nil {
        errorlog.Printf("while getting resource at jumpcode %s: %v", jc, err)
        w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
        w.Write(encUTF16LE(err.Error()))
        return
    }
    
    var red string
    switch t {
    case "movie":
        red = fmt.Sprintf("movie/%d.htm", id)
    case "user":
        red = fmt.Sprintf("profile.htm?view=%d", id)
    case "channel":
        red = fmt.Sprintf("channel.uls?ch=%d&s=new&page=1", id)
    default:
        w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
        w.Write(encUTF16LE("no jump 4 u"))
        return
    }
    
    w.Header()["X-DSi-Dialog-Type"] = []string{"0"}
    w.Write([]byte(ub(e.cnf.Root, s.Region, red)))
}

func (e *env) replyui(w http.ResponseWriter, r *http.Request) {
    sid := r.Header.Get("X-Dsi-Sid")
    
    id, err := strconv.Atoi(mux.Vars(r)["movieid"])
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

// todo: movie movie htm here

// debug handler just runs a template
func (e *env) debug(w http.ResponseWriter, r *http.Request) {
    sid := r.Header.Get("X-Dsi-Sid")
    
    d, err := e.fillpage(sid)
    if err != nil {
        errorlog.Printf("while filling DSPage: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    
    sp := [64]string{}
    for i:=0;i<0x40;i++ {
        sp[i] = string(0xE000 + i) // intended
    }
    
    d["sp"] = sp
    d["jsp"] = jumpasciitonds("ABXYLRNSWE")
    
    d["sid"] = sid

    if err := e.html.ExecuteTemplate(w, "debug.html", d); err != nil {
        errorlog.Printf("while executing template: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
    }
}

func (e *env) profile(w http.ResponseWriter, r *http.Request) {
    sid := r.Header.Get("X-Dsi-Sid")
    s := e.sessions[sid]
    var view int

    d, err := e.fillpage(sid)
    if err != nil {
        errorlog.Printf("while filling page (profile view): %v", err)
        w.WriteHeader(http.StatusInternalServerError)
        return
    }
    
    
    viewr := r.URL.Query().Get("view")
    if viewr == "me" {
        view = s.UserID
        d["viewingself"] = true
    } else {
        idr, err := strconv.Atoi(viewr)
        if err != nil {
            w.WriteHeader(http.StatusBadRequest)
            return
        }
        
        view = idr
    }
    
    // get profile
    // todo implement user preferences stuff
    u, err := getUserById(e.pool, view)
    if err != nil {
        switch err {
        case ErrNoUser:
            w.WriteHeader(http.StatusNotFound)
            return

        default:
            errorlog.Printf("while getting user %d (profile view): %v", view, err)
            w.WriteHeader(http.StatusInternalServerError)
            return
        }
    }
	
	// get that user's stars
	us, err := getUserStars(e.pool, u.ID)
	if err != nil {
		errorlog.Printf("while getting user %d stars (profile view): %v", view, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
    
    d["vuser"] = u
	d["vuserstars"] = us
    
    e.html.ExecuteTemplate(w, "profile.html", d)
}

// misc handler is here for minor things that need to return something, but don't necessarily matter
func (e *env) misc(w http.ResponseWriter, r *http.Request) {
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

    case "/ds/v2-us/redirect.htm":
        w.Write([]byte(`<html><head></head><body>works</body</html>`))
    case "/ds/v2-us/mail.send":
        w.WriteHeader(http.StatusOK)
    case "/ds/v2-us/":
        w.Write(e.assets["images/ds/8x8test.npf"])
	case "/robots.txt":
		b, ok := e.assets["text/robots.txt"]
		if !ok {
			b = []byte("beep boop bitch")
		}
		w.Write(b)
    }
    
}