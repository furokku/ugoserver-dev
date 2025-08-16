package main

import (
	//fmt"
	//html/template"

	"net/http"
)

func catchall(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/", "/index.html":
		if err := cache_html.ExecuteTemplate(w, "webui_front.html", nil); err != nil {
			errorlog.Printf("while executing template: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
}

func ui_account(w http.ResponseWriter, r *http.Request) {
	resp := WebPage{
		Return: r.URL.Query().Get("ret"),
	}

	cookie, err := r.Cookie("token")
	if err != nil {
		switch err {
		case http.ErrNoCookie:
			resp.User = User{ID:0}
		default:
			errorlog.Printf("while getting cookie from request: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		// TODO encrypt cookie
		
		user, err := getUserApiToken((*cookie).Value)
		if err != nil {
			errorlog.Printf("while fetching user from token in cookie: %v", err)
		}
		resp.User = user
	}
	

	if err := cache_html.ExecuteTemplate(w, "webui_account.html", resp); err != nil {
		errorlog.Printf("while executing template: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}