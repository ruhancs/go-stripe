package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

func (app *application) routes() http.Handler {
	mux := chi.NewRouter()

	//habilitar cors
	mux.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"https://*", "http://*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: false,
		MaxAge: 300,// 5 minutos
	}))

	mux.Post("/api/payment-intent", app.GetPaymentIntent)

	mux.Get("/api/widget/{id}", app.GetWidgetById)

	mux.Post("/api/create-customer-and-subscribe-to-plan", app.CreateCustomerAndSubscribe)

	mux.Post("/api/authenticate", app.CreateAuthToken)

	mux.Post("/api/is-authenticated", app.CheckAthentication)

	mux.Post("/api/forgot-password",app.SendPasswordResetEmail)
	mux.Post("/api/reset-password",app.ResetPassword)

	//adicionar middleware de protecao de rotas
	mux.Route("/api/admin", func(mux chi.Router) {
		mux.Use(app.Auth)//middleware para verificar auth
		
		mux.Post("/virtual-terminal-succeded",app.VirtualTerminalPaymentSucceded)
		mux.Post("/all-sales", app.AllSales)
		mux.Post("/all-subscriptions", app.AllSubscriptions)
		mux.Post("/get-sale/{id}", app.GetSale)
		//mux.Post("/get-subscription/{id}", app.GetSale)

	})


	return mux
}
