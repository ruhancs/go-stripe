package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/ruhancs/go-stripe/internal/cards"
)

type stripePayload struct {
	Currency string `json:"currency"`
	Amount string `json:"amount"`
}

type jsonresponse struct {
	Ok bool `json:"ok"`
	Message string `json:"message,omitempty"`
	Content string `json:"content,omitempty"`
	Id int `json:"id,omitempty"`
}

func (app *application) GetPaymentIntent(w http.ResponseWriter, r *http.Request) {
	var payload stripePayload

	//decodificar o body da request para o formato do stripepayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		app.errorLog.Println(err)
		return
	}
	
	//converter o payload.amount que sera recebido como string em numero
	amount, err := strconv.Atoi(payload.Amount)
	if err != nil {
		app.errorLog.Println(err)
		return
	}
	
	card := cards.Card{
		Secret: app.config.stripe.secret,
		Key: app.config.stripe.key,
		Currency: payload.Currency,
	}
	
	okay := true
	
	paymentIntent, msg, err := card.Charge(payload.Currency,amount)
	if err != nil {
		okay = false
	}

	//se a paymentIntent ocorrer tudo certo convert o paymentIntent para json com identacao
	if okay {
		out, err := json.MarshalIndent(paymentIntent, "", "  ")
		if err != nil {
			app.errorLog.Println(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
	} else {
		jsonRes := jsonresponse{
			Ok: false,
			Message: msg,
			Content: "",
		}
	
		out, err := json.MarshalIndent(jsonRes, "", "")
		if err != nil {
			app.errorLog.Println(err)
		}
	
		w.Header().Set("Content-Type", "application/json")
		w.Write(out)
	}
}

func (app *application) GetWidgetById(w http.ResponseWriter, r *http.Request) {
	//pegar o id da url
	id := chi.URLParam(r, "id")
	widgetId, _ := strconv.Atoi(id)//converter id para numero

	widget, err := app.DB.GetWidget(widgetId)
	if err != nil {
		app.errorLog.Println(err)
		return
	}
	
	out,err := json.MarshalIndent(widget, "", "  ")
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}