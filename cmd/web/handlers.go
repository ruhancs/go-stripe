package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ruhancs/go-stripe/internal/cards"
	"github.com/ruhancs/go-stripe/internal/encryption"
	"github.com/ruhancs/go-stripe/internal/models"
	"github.com/ruhancs/go-stripe/internal/urlsigner"
)

func (app *application) Home(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w, r, "home", &templateData{}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) VirtualTerminal(w http.ResponseWriter, r *http.Request) {
	//renderizar template e inserir o stripe-js para utilizar na template
	if err := app.renderTemplate(w, r, "terminal", &templateData{}); err != nil {
		app.errorLog.Println(err)
	}
}

type TransactionData struct {
	FirstName string
	LastName string
	Email string
	PaymentIntentID string
	PaymentMethodID string
	PaymentAmount int
	PaymentCurrency string
	LastFour string
	ExpiryMonth int
	ExpiryYear int
	BankReturnCode string
}

//pegar informacoes do post para comprar e do stripe
func(app *application) GetTransactionData(r *http.Request) (TransactionData,error) {
	var transactionData TransactionData
	err:= r.ParseForm()//pegar erros do formulario
	if err != nil {
		app.errorLog.Println(err)
		return transactionData,err
	}

	//read post data
	//dados do formulario
	firstName := r.Form.Get("first_name")
	lastName := r.Form.Get("last_name")
	//cardHolderName := r.Form.Get("cardholder_name")
	email := r.Form.Get("email")
	paymentIntent := r.Form.Get("payment_intent")
	paymentMethod := r.Form.Get("payment_method")
	paymentAmount := r.Form.Get("payment_amount")
	paymentCurrency := r.Form.Get("payment_currency")

	amount,_ := strconv.Atoi(paymentAmount)

	card := cards.Card{
		Secret: app.config.stripe.secret,
		Key: app.config.stripe.key,
	}

	pi, err := card.GetPaymentIntent(paymentIntent)
	if err != nil {
		app.errorLog.Println(err)
		return transactionData,nil
	}
	
	pm, err := card.GetPaymentMethod(paymentMethod)
	if err != nil {
		app.errorLog.Println(err)
		return transactionData,nil
	}

	lastFour := pm.Card.Last4
	expiryMonth := pm.Card.ExpMonth
	expiryYear := pm.Card.ExpYear

	transactionData = TransactionData{
		FirstName: firstName,
		LastName: lastName,
		Email: email,
		PaymentIntentID: paymentIntent,
		PaymentMethodID: paymentMethod,
		PaymentAmount: amount,
		PaymentCurrency: paymentCurrency,
		LastFour: lastFour,
		ExpiryMonth: int(expiryMonth),
		ExpiryYear: int(expiryYear),
		BankReturnCode: pi.Charges.Data[0].ID,
	}
	return transactionData,nil
}

type Invoice struct {
	ID int `json:"id"`
	Quantity int `json:"quantity"`
	Amount int `json:"amount"`
	Product string `json:"product"`
	FirstName string `json:"first_name"`
	LastName string `json:"last_name"`
	Email string `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	//Items []Products
}

func (app *application) PaymentSucceeded(w http.ResponseWriter, r *http.Request) {
	err:= r.ParseForm()//pegar erros do formulario
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	widgetID,_ := strconv.Atoi(r.Form.Get("product_id")) 

	transactionData,err := app.GetTransactionData(r)
	if err != nil {
		app.errorLog.Panicln(err)
		return
	}

	//create customer
	customerID, err := app.SaveCustomer(transactionData.FirstName,transactionData.LastName,transactionData.Email)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	app.infolog.Println(customerID)

	//create transaction
	transaction := models.Transaction{
		Amount: transactionData.PaymentAmount,
		Currency: transactionData.PaymentCurrency,
		LastFour: transactionData.LastFour,
		ExpiryMonth: transactionData.ExpiryMonth,
		ExpiryYear: transactionData.ExpiryYear,
		PaymentIntent: transactionData.PaymentIntentID,
		PaymentMethod: transactionData.PaymentMethodID,
		BankReturnCode: transactionData.BankReturnCode,
		TarnsactionStatusID: 2,//transaction status cleared ocorreu tudo certo
	}
	transactionID,err := app.SaveTransaction(transaction)
	if err != nil {
		app.errorLog.Println(err)
		return
	}
	
	//create order
	order:= models.Order{
		WidgetID: widgetID,
		TransactionID: transactionID,
		CustomerID: customerID,
		StatusID: 1,//status cleared
		Quantity: 1,
		Amount: transactionData.PaymentAmount,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	orderID,err := app.SaveOrder(order)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	// chamar microservico para gerar nota fiscal por email
	invoice := Invoice {
		ID: orderID,
		Amount: order.Amount,
		Product: "Widget",
		Quantity: order.Quantity,
		FirstName: transactionData.FirstName,
		LastName: transactionData.LastName,
		Email: transactionData.Email,
		CreatedAt: time.Now(),
	}

	err = app.CallInvoiceMicro(invoice)
	if err != nil {
		app.errorLog.Println(err)
	}

	//should write this data to session, and redirect user
	//inserir o contexto da requisicao na sessao
	app.Session.Put(r.Context(), "receipt", transactionData)

	//redirecionar para receipt
	http.Redirect(w,r, "/receipt", http.StatusSeeOther)
}

func(app *application) CallInvoiceMicro(invoice Invoice) error {
	url := "http://localhost:5000/invoice/create-and-send"
	out, err := json.MarshalIndent(invoice, "", "\t")
	if err != nil {
		return err
	}

	req,err := http.NewRequest("POST", url, bytes.NewBuffer(out))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	app.infolog.Println(resp.Body)
	return nil
}

func (app *application) VirtualTerminalPaymentSucceeded(w http.ResponseWriter, r *http.Request) {
	transactionData,err := app.GetTransactionData(r)
	if err != nil {
		app.errorLog.Panicln(err)
		return
	}

	//create transaction
	transaction := models.Transaction{
		Amount: transactionData.PaymentAmount,
		Currency: transactionData.PaymentCurrency,
		LastFour: transactionData.LastFour,
		ExpiryMonth: transactionData.ExpiryMonth,
		ExpiryYear: transactionData.ExpiryYear,
		PaymentIntent: transactionData.PaymentIntentID,
		PaymentMethod: transactionData.PaymentMethodID,
		BankReturnCode: transactionData.BankReturnCode,
		TarnsactionStatusID: 2,//transaction status cleared ocorreu tudo certo
	}
	_,err = app.SaveTransaction(transaction)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	//should write this data to session, and redirect user
	//inserir o contexto da requisicao na sessao
	app.Session.Put(r.Context(), "receipt", transactionData)

	//redirecionar para receipt
	http.Redirect(w,r, "/virtual-terminal-receipt", http.StatusSeeOther)
}

func (app *application) Receipt(w http.ResponseWriter, r *http.Request) {
	txn := app.Session.Get(r.Context(), "receipt").(TransactionData)//pegar os dados da requisicao em receipt apos o pagamento
	data := make(map[string]interface{})
	data["txn"] = txn
	app.Session.Remove(r.Context(), "receipt")// remover os dados da sessao apos utilizados
	if err := app.renderTemplate(w,r,"receipt", &templateData{
		Data: data,
	}); err != nil {
		app.errorLog.Panicln(err)
	}
}

func (app *application) VirtualTerminalReceipt(w http.ResponseWriter, r *http.Request) {
	txn := app.Session.Get(r.Context(), "receipt").(TransactionData)//pegar os dados da requisicao em receipt apos o pagamento
	data := make(map[string]interface{})
	data["txn"] = txn
	app.Session.Remove(r.Context(), "receipt")// remover os dados da sessao apos utilizados
	if err := app.renderTemplate(w,r,"virtual-terminal-receipt", &templateData{
		Data: data,
	}); err != nil {
		app.errorLog.Panicln(err)
	}
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

//dispaly page to buy one item
func (app *application) ChargeOnce(w http.ResponseWriter, r *http.Request) {
	//pegar o id da url
	id := chi.URLParam(r, "id")
	widgetId, _ := strconv.Atoi(id)//converter id para numero

	widget, err := app.DB.GetWidget(widgetId)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	data := make(map[string]interface{})
	data["widget"] = widget
	if err := app.renderTemplate(w, r, "buy-once", &templateData{
		Data: data,
	}, "stripe-js"); err != nil {
		app.errorLog.Panicln(err)
	}
}

func (app *application) BronzePlan(w http.ResponseWriter, r * http.Request) {

	widget,err := app.DB.GetWidget(2)
	if err != nil {
		app.errorLog.Println(err)
	}
	//variavel para enviar dados para atemplate
	data := make(map[string]interface{})
	data["widget"] = widget

	if err := app.renderTemplate(w,r, "bronze-plan", &templateData{
		Data: data,
	}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) BronzePlanreceipt(w http.ResponseWriter, r * http.Request) {

	if err := app.renderTemplate(w,r, "receipt-plan", &templateData{}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) LoginPage(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w,r, "login", &templateData{}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) PostLoginPage(w http.ResponseWriter, r *http.Request) {
	app.Session.RenewToken(r.Context())//renovar o token da sessao

	err := r.ParseForm()
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	email := r.Form.Get("email")
	password := r.Form.Get("password")

	id, err := app.DB.Authenticate(email,password)
	if err != nil {
		fmt.Println("error no db")
		http.Redirect(w,r, "/login", http.StatusSeeOther)
		return
	}

	//inserir o userID no contexto
	app.Session.Put(r.Context(), "userID", id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *application) Logout(w http.ResponseWriter, r *http.Request) {
	app.Session.Destroy(r.Context())
	app.Session.RenewToken(r.Context())
	http.Redirect(w,r, "/login", http.StatusSeeOther)
}

func (app *application) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w,r, "forgot-password", &templateData{}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) ShowResetPassword(w http.ResponseWriter,r *http.Request) {
	email := r.URL.Query().Get("email")//pegar o email do queryparams
	resetUrl := r.RequestURI
	testUrl := fmt.Sprintf("%s%s", app.config.frontend, resetUrl)

	signer := urlsigner.Signer{
		Secret: []byte(app.config.secretKey),
	}

	valid := signer.VerifyToken(testUrl)//checar se a url é valida

	if !valid {
		app.errorLog.Println("Invalid url - reset password ")
		return
	}

	//verificar se o token nao expirou testUrl é o token
	expired := signer.Expire(testUrl, 60) //60 minutos para expiracao
	if expired {
		app.errorLog.Println("Link expired")
		return
	}

	//encryptografar email
	encryptor := encryption.Encryption{
		Key: []byte(app.config.secretKey),
	}
	encryptedEmail, err := encryptor.Encrypt(email)
	if err != nil {
		app.errorLog.Println("encryption failed")
		return
	}
	
	data := make(map[string]interface{})
	data["email"] = encryptedEmail

	if err := app.renderTemplate(w,r, "reset-password", &templateData{
		Data: data,
	}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) AllSales(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w,r, "all-sales", &templateData{}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) AllSubscriptions(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w,r, "all-subscriptions", &templateData{}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) ShowSale(w http.ResponseWriter, r *http.Request) {
	stringMap := make(map[string]string)
	stringMap["title"] = "Sale"
	stringMap["cancel"] = "/admin/all-sales"
	stringMap["refund-url"] = "/api/admin/refund"
	stringMap["refund-btn"] = "Refunds Order"
	stringMap["refunded-msg"] = "Charge refunded"
	stringMap["refunded-badge"] = "Refunded"
	if err := app.renderTemplate(w,r, "sale", &templateData{
		StringMap: stringMap,
	}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) ShowSubscription(w http.ResponseWriter, r *http.Request) {
	stringMap := make(map[string]string)
	stringMap["title"] = "Subscription"
	stringMap["cancel"] = "/admin/all-subscriptions"
	stringMap["refund-url"] = "/api/admin/cancel-subscription"
	stringMap["refund-btn"] = "Cancel Subscription"
	stringMap["refunded-msg"] = "Subscription cancelled"
	stringMap["refunded-badge"] = "Cancelled"
	if err := app.renderTemplate(w,r, "sale", &templateData{
		StringMap: stringMap,
	}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) AllUsers(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w,r, "all-users", &templateData{}); err != nil {
		app.errorLog.Println(err)
	}
}

func (app *application) OneUser(w http.ResponseWriter, r *http.Request) {
	if err := app.renderTemplate(w,r, "one-user", &templateData{}); err != nil {
		app.errorLog.Println(err)
	}
}
