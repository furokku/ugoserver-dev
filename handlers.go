package main

import (
	"fmt"
	"strings"

	"net/http"
	"os"

	"github.com/gorilla/mux"
)

func eula(w http.ResponseWriter, r *http.Request) {

    name := mux.Vars(r)["txt"]
    
    t, ok := cache_assets[fmt.Sprintf("text/%s", name)]
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
    case "/ds/v2-us/mail.send":
        w.WriteHeader(http.StatusOK)
    case "/ds/v2-us/":
        w.Write(cache_assets["/images/ds/8x8test.npf"])
	case "/robots.txt":
		b, ok := cache_assets["text/robots.txt"]
		if !ok {
			b = []byte("beep boop bitch")
		}
		w.Write(b)
    }
    
}

// return whatever random file from assets/
func asset(w http.ResponseWriter, r *http.Request) {
    rs, _ := strings.CutPrefix(r.URL.Path, "/")
    // try cache
    c, ok := cache_assets[rs]
    if !ok {
        c, err = os.ReadFile(fmt.Sprintf("%s/assets/%s", cnf.Dir, rs))
        if err != nil {
            w.WriteHeader(http.StatusNotFound)
            return
        }
    } else {
        debuglog.Printf("fetched %s from cache", rs)
    }
    
    w.Header().Add("Content-Type", "application/octet-stream")
    w.Write(c)
}

// same as above but content-type header set to css
func css(w http.ResponseWriter, r *http.Request) {
    rs, _ := strings.CutPrefix(r.URL.Path, "/")
    // try cache
    c, ok := cache_assets[rs]
    if !ok {
        c, err = os.ReadFile(fmt.Sprintf("%s/assets/%s", cnf.Dir, rs))                
        if err != nil {
            w.WriteHeader(http.StatusNotFound)
            return
        }
    } else {
        debuglog.Printf("fetched %s from cache", rs)
    }
    
    w.Header().Add("Content-Type", "text/css")
    w.Write(c)
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

    if err := cache_html.ExecuteTemplate(w, "debug.html", DSPage{
        Session: s,
        Root: cnf.Root,
        Region: s.getregion(),
        SID: sid,
    }); err != nil {
        errorlog.Printf("while executing template: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
    }
}
