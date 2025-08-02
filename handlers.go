package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

// eula handler returns text files in static/txt as utf16le
func eula(w http.ResponseWriter, r *http.Request) {

    name := mux.Vars(r)["txt"]
    
    content, ok := texts[name]
    if !ok {
        content = "you're weird"
    }

    w.Write(encUTF16LE(content))
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
        bytes, _ := os.ReadFile("static/images/ds/8x8test.npf")
        w.Write(bytes)
	case "/robots.txt":
		b, ok := texts["robots"]
		if !ok {
			b = "beep boop bitch"
		}
		w.Write([]byte(b))
    }
    
}

// static handler returns the file from cnf.Dir/static/path
func static(w http.ResponseWriter, r *http.Request) {
    file, err := os.ReadFile(fmt.Sprintf("%s/static/%s", cnf.Dir, r.URL.Path))
    if err != nil {
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

    if err := templates.ExecuteTemplate(w, "debug.html", Page{
        Session: s,
        Root: cnf.Root,
        Region: s.getregion(),
        SID: sid,
    }); err != nil {
        errorlog.Printf("while executing template: %v", err)
        w.WriteHeader(http.StatusInternalServerError)
    }
}
