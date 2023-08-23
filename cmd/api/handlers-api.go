package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ruhancs/go-stripe/internal/cards"
	"github.com/ruhancs/go-stripe/internal/models"
	"github.com/stripe/stripe-go/v72"
)

type stripePayload struct {
	Currency string `json:"currency"`
	Amount string `json:"amount"`
	PaymentMethod string `json:"payment_method"`
	Email string `json:"email"`
	CardBrand string `json:"card_brand"`
	ExpiryMonth int `json:"exp_month"`
	ExpiryYear int `json:"exp_year"`
	LastFour string `json:"last_four"`
	Plan string `json:"plan"`
	ProductID string `json:"product_id"`
	FirstName string `json:"first_name"`
	LastName string `json:"last_name"`
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

func (app *application) CreateCustomerAndSubscribe(w http.ResponseWriter, r *http.Request) {
	//dados recebidos de do formulario de bronze-plan.page
	var data stripePayload
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	app.infolog.Println(data.Email,data.LastFour, data.PaymentMethod, data.Plan)

	card := cards.Card{
		Secret: app.config.stripe.secret,
		Key: app.config.stripe.key,
		Currency: data.Currency,
	}

	okay := true
	var subscription *stripe.Subscription
	transactionMsg := "Transaction successfull"

	stripeCustomer,msg,err := card.CreateCustomer(data.PaymentMethod,data.Email)
	if err != nil {
		app.errorLog.Println(err)
		okay = false
		transactionMsg = msg
	}
	
	if okay {
		subscription,err = card.SubscribeToPlan(stripeCustomer, data.Plan, data.Email,data.LastFour, "")
		if err != nil {
			app.errorLog.Println(err)
			okay = false
			transactionMsg = "Error subscribing customer"
		}
	
		app.infolog.Println("subscription ID is: ", subscription.ID)
	}
	
	if okay {
		//criar customer e transaction
		productID, _ := strconv.Atoi(data.ProductID)
		customerID, err := app.SaveCustomer(data.FirstName, data.LastName, data.Email)
		if err != nil {
			app.errorLog.Println(err)
			return
		}
		
		amount, _ := strconv.Atoi(data.Amount)
		
		transaction := models.Transaction{
			Amount: amount,
			Currency: "R$",
			LastFour: data.LastFour,
			ExpiryMonth: data.ExpiryMonth,
			ExpiryYear: data.ExpiryYear,
			TarnsactionStatusID: 2,
		}

		transactionID,err := app.SaveTransaction(transaction)
		if err != nil {
			app.errorLog.Println(err)
			return
		}

		order := models.Order{
			WidgetID: productID,
			TransactionID: transactionID,
			CustomerID: customerID,
			StatusID: 1,
			Quantity: 1,
			Amount: amount,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		_,err = app.SaveOrder(order)
		if err != nil {
			app.errorLog.Println(err)
			return
		}
	}

	resp := jsonresponse{
		Ok: okay,
		Message: transactionMsg,
	}

	out,err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

func (app *application) SaveCustomer(firstName string, lastName string, email string) (int, error) {
	customer := models.Customer{
		FirstName: firstName,
		LastName: lastName,
		Email: email,
	}

	id,err := app.DB.InsertCustomer(customer)
	if err != nil {
		app.errorLog.Println(err)
		return 0, err
	}
	return id, nil
}

func (app *application) SaveTransaction(t models.Transaction) (int, error) {
	id,err := app.DB.InsertTransaction(t)
	if err != nil {
		app.errorLog.Println(err)
		return 0, err
	}
	return id, nil
}

func (app *application) SaveOrder(order models.Order) (int, error) {
	id,err := app.DB.InsertOrder(order)
	if err != nil {
		app.errorLog.Println(err)
		return 0, err
	}
	return id, nil
}