package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (app *application) routes() http.Handler{
	mux := chi.NewRouter()
	mux.Use(SessionLoad)//habilitar carregamento de sessao

	mux.Get("/", app.Home)
	//quando o usuario esta logado conecta ele no ws no base.page.gohtml
	mux.Get("/ws",app.WsEndpoint)
	
	mux.Route("/admin", func(mux chi.Router) {
		mux.Use(app.Auth)
		mux.Get("/virtual-terminal", app.VirtualTerminal)
		mux.Get("/all-sales", app.AllSales)
		mux.Get("/all-subscriptions", app.AllSubscriptions)
		mux.Get("/sales/{id}", app.ShowSale)
		mux.Get("/subscriptions/{id}", app.ShowSubscription)
		mux.Get("/all-users", app.AllUsers)
		mux.Get("/all-users/{id}", app.OneUser)
	})
	
	mux.Get("/widget/{id}", app.ChargeOnce)
	mux.Post("/payment-succeeded", app.PaymentSucceeded)
	mux.Get("/receipt", app.Receipt)

	mux.Get("/plans/bronze", app.BronzePlan)
	mux.Get("/receipt/bronze", app.BronzePlanreceipt)

	mux.Get("/login", app.LoginPage)
	mux.Post("/login", app.PostLoginPage)
	mux.Get("/logout", app.Logout)

	mux.Get("/forgot-password", app.ForgotPassword)
	mux.Get("/reset-password", app.ShowResetPassword)

	//informar diretorio dos arquivos estaticos
	fileServer := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/*", http.StripPrefix("/static", fileServer))

	return mux
}