package main

import "net/http"

func SessionLoad(next http.Handler) http.Handler {
	return session.LoadAndSave(next)
}

func (app *application) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//buscar o userID na sessao
		if !app.Session.Exists(r.Context(), "userID"){
			http.Redirect(w,r, "/login", http.StatusTemporaryRedirect)
			return
		} 
		next.ServeHTTP(w,r)
	})
}