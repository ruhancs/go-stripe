package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (app *application) routes() http.Handler{
	mux := chi.NewRouter()
	mux.Use(SessionLoad)//habilitar carregamento de sessao

	mux.Get("/", app.Home)
	mux.Get("/virtual-terminal", app.VirtualTerminal)
	mux.Post("/payment-succeeded", app.PaymentSucceeded)
	mux.Get("/receipt", app.Receipt)
	mux.Get("/widget/{id}", app.ChargeOnce)

	//informar diretorio dos arquivos estaticos
	fileServer := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/*", http.StripPrefix("/static", fileServer))

	return mux
}