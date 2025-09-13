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


// TODO
func jump(w http.ResponseWriter, r *http.Request) {
    w.Header()["X-DSi-Dialog-Type"] = []string{"1"}
    w.Write(encUTF16LE("bazinga"))
}

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