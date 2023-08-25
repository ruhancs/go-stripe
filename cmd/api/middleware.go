package main

import "net/http"

func (app *application) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_,err := app.AuthenticateToken(r)
		if err != nil {
			app.invalidCredencials(w)
			return
		}
		next.ServeHTTP(w,r)
	})
}