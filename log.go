package main

import (
	"log"
	"net/http"
	"os"
	"slices"
)

var (
    colorReset = "\033[0m"

    debuglog = log.New(os.Stdout, "[debug] ", log.Lshortfile|log.Ldate|log.Ltime)
    infolog = log.New(os.Stdout, "[info] ", log.Ldate|log.Ltime)
    reqlog = log.New(os.Stdout, "[req] ", log.Ldate|log.Ltime)
    resplog = log.New(os.Stdout, "[resp] ", log.Ldate|log.Ltime)
    warnlog = log.New(os.Stdout, "\033[33m[warn] " + colorReset, log.Ldate|log.Ltime)
    errorlog = log.New(os.Stdout, "\033[31m[error] " + colorReset, log.Lshortfile|log.Ldate|log.Ltime)
)

func wrapResponseWriter(w http.ResponseWriter) *rwWrapper {
    return &rwWrapper{ResponseWriter: w}
}

func (rw *rwWrapper) Status() int {
    return rw.status
}

func (rw *rwWrapper) WriteHeader(code int) {
    if rw.done {
        return
    }

    rw.status = code
    rw.ResponseWriter.WriteHeader(code)
    rw.done = true
}

// logger is middleware to log all HTTP requests and responses
func logger(next http.Handler) http.Handler {
    fn := func(w http.ResponseWriter, r *http.Request) {
        ua := r.Header.Get("User-Agent")

        // ignore bots and etc.
        if cnf.UseHosts && r.URL.Path != "/robots.txt" {
            if !slices.Contains(cnf.Hosts, r.Host) ||
               (ua != "" && ua != "Nitro WiFi SDK/5.1") {
                w.Write([]byte("you may not access this resource in this way"))
                return
            }
        }
        //log request
        reqlog.Printf("%v %v %v%v // %v", r.Header.Get("X-Real-Ip"), r.Method, r.Host, r.RequestURI, r.Header)

        rww := wrapResponseWriter(w)
        next.ServeHTTP(rww, r)
        //log response
        resplog.Printf("%v %v %v%v %v // %v", rww.status, r.Method, r.Host, r.RequestURI, r.Header.Get("X-Real-Ip"), w.Header())
    }

    return http.HandlerFunc(fn)
}
