package main

import (
	//fmt"
	//html/template"

	"net/http"
)

func (e *env) ui_front(w http.ResponseWriter, r *http.Request) {
	if err := e.html.ExecuteTemplate(w, "webui_front.html", nil); err != nil {
		errorlog.Printf("while executing template: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (e *env) ui_account(w http.ResponseWriter, r *http.Request) {
	resp := map[string]interface{}{
		"return": r.URL.Query().Get("ret"),
	}

	cookie, err := r.Cookie("token")
	if err != nil {
		switch err {
		case http.ErrNoCookie:
			resp["user"] = User{ID:0}
		default:
			errorlog.Printf("while getting cookie from request: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	} else {
		// TODO encrypt cookie
		
		user, err := getUserApiToken(e.pool, (*cookie).Value)
		if err != nil {
			errorlog.Printf("while fetching user from token in cookie: %v", err)
		}
		resp["user"] = user
	}
	

	if err := e.html.ExecuteTemplate(w, "webui_account.html", resp); err != nil {
		errorlog.Printf("while executing template: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}