package main

import (
	"net/http"
)

func api(w http.ResponseWriter, r *http.Request) {

}

func mgmt(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/api/manage":
		w.Write([]byte("ugoserver management panel test"))
		return
	case "/api/manage/brief":
		w.Write([]byte("burp"))
		return
	}
}