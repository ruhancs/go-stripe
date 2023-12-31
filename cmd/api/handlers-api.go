package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ruhancs/go-stripe/internal/cards"
	"github.com/ruhancs/go-stripe/internal/encryption"
	"github.com/ruhancs/go-stripe/internal/models"
	"github.com/ruhancs/go-stripe/internal/urlsigner"
	"github.com/ruhancs/go-stripe/internal/validator"
	"github.com/stripe/stripe-go/v72"
	"golang.org/x/crypto/bcrypt"
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

func (app *application) CreateCustomerAndSubscribe(w http.ResponseWriter, r *http.Request) {
	//dados recebidos de do formulario de bronze-plan.page
	var data stripePayload
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	//validate data
	v := validator.New()
	//validacao do first_name tamanho de no minimo 2 caracteres
	//first_name deve ser exatamente o campo de id do componente na template
	v.Check(len(data.FirstName) > 1, "first_name", "must be at least 2 character")

	if !v.Valid() {
		app.failedValidation(w, r, v.Errors)
		return
	}

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
			PaymentIntent: subscription.ID,
			PaymentMethod: data.PaymentMethod,
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

		orderID,err := app.SaveOrder(order)
		if err != nil {
			app.errorLog.Println(err)
			return
		}

		//criar invoice
		invoice := Invoice {
			ID: orderID,
			Amount: order.Amount,
			Product: order.Widget.Description,
			Quantity: order.Quantity,
			FirstName: data.FirstName,
			LastName: data.LastName,
			Email: data.Email,
			CreatedAt: time.Now(),
		}

		//chamar micro de invoice
		app.CallInvoiceMicro(invoice)

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

func (app *application) CreateAuthToken(w http.ResponseWriter, r *http.Request) {
	var userInput struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &userInput)
	if err != nil {
		app.badRequest(w,r,err)
		return
	}

	//checar email se existe cadastro
	user,err := app.DB.GetUserByEmail(userInput.Email)
	if err != nil {
		app.invalidCredencials(w)
		return
	}

	//checar se a senha esta certa
	validPassword,err := app.passwordMatch(user.Password, userInput.Password)
	if err != nil {
		app.invalidCredencials(w)
		return
	}

	if !validPassword {
		app.invalidCredencials(w)
		return
	}

	//gerar token
	token,err := models.GenerateToken(user.ID, 24 * time.Hour, models.ScopeAuthentication)
	if err != nil {
		app.badRequest(w,r,err)
		return
	}
	
	//salvar o token no db
	err = app.DB.InsertToken(token,user)
	if err != nil {
		app.badRequest(w,r,err)
		return
	}

	var payload struct {
		Error bool `json:"error"`
		Message string `json:"message"`
		Token *models.Token `json:"authentication_token"`
	}
	payload.Error = false
	payload.Message = fmt.Sprintf("token for %s created", userInput.Email)
	payload.Token = token

	_ = app.writeJSON(w, http.StatusOK, payload)
}

func (app *application) AuthenticateToken(r *http.Request) (*models.User, error) {
	//pegar o token de Authorization
	authorizationHeader := r.Header.Get("Authorization")
	if authorizationHeader == "" {
		return nil, errors.New("no authorization header received")
	}
	
	headerParts := strings.Split(authorizationHeader, " ")
	if len(headerParts) != 2 || headerParts[0] != "Bearer" {
		return nil, errors.New("no authorization header received")
	}
	
	token := headerParts[1]
	if len(token) != 26 {
		return nil, errors.New("invalid token")
	}

	//pegar o usuario da tabela de tokens
	user,err := app.DB.GetUserForToken(token)
	if err != nil {
		return nil, errors.New("no user with this token")
	}

	return  user, nil
}

func (app *application) CheckAthentication(w http.ResponseWriter, r *http.Request) {
	//validar token e pegar o usuario do token
	user,err := app.AuthenticateToken(r)
	if err != nil {
		app.invalidCredencials(w)
		return
	}

	//valid user
	var payload struct {
		Error bool `json:"error"`
		Message string `json:"message"`
	}
	payload.Error = false
	payload.Message = fmt.Sprintf("authenticate user %s", user.Email)
	app.writeJSON(w,http.StatusOK, payload)
}

func (app *application) VirtualTerminalPaymentSucceded(w http.ResponseWriter, r *http.Request) {
	var txnData struct {
		PaymentAmount int `json:"amount"`
		PaymentCurrency string `json:"currency"`
		FirstName string `json:"first_name"`
		LastName string `json:"last_name"`
		Email string `json:"email"`
		PaymentIntent string `json:"payment_intent"`
		PaymentMethod string `json:"payment_method"`
		BankReturnCode string `json:"bank_return_code"`
		ExpiryMonth int `json:"expiry_month"`
		ExpiryYear int `json:"expiry_year"`
		LastFour string `json:"last_four"`
	}

	//inserir em txnData os dados do post
	err := app.readJSON(w,r, &txnData)
	if err != nil {
		app.badRequest(w,r, err)
		return
	}
	
	card := cards.Card{
		Secret: app.config.stripe.secret,
		Key: app.config.stripe.key,
	}
	
	pi, err := card.GetPaymentIntent(txnData.PaymentIntent)
	if err != nil {
		app.badRequest(w,r, err)
		return
	}

	pm,err := card.GetPaymentMethod(txnData.PaymentMethod)
	if err != nil {
		app.badRequest(w,r, err)
		return
	}

	txnData.LastFour = pm.Card.Last4
	txnData.ExpiryMonth = int(pm.Card.ExpMonth)
	txnData.ExpiryYear = int(pm.Card.ExpYear)

	txn := models.Transaction {
		Amount: txnData.PaymentAmount,
		Currency: txnData.PaymentCurrency,
		LastFour: txnData.LastFour,
		ExpiryMonth: txnData.ExpiryMonth,
		ExpiryYear: txnData.ExpiryYear,
		PaymentIntent: txnData.PaymentIntent,
		PaymentMethod: txnData.PaymentMethod,
		BankReturnCode: pi.Charges.Data[0].ID,
		TarnsactionStatusID: 2,
	}

	_, err = app.SaveTransaction(txn)
	if err != nil {
		app.badRequest(w,r, err)
		return
	}

	app.writeJSON(w, http.StatusOK, txn)
}

func (app *application) SendPasswordResetEmail(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email string `json:"email"`
	}

	err := app.readJSON(w, r, &payload)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	//verificar se o email esta cadastrado
	_, err = app.DB.GetUserByEmail(payload.Email)
	if err != nil {
		var resp struct {
			Error bool `json:"error"`
			Message string `json:"message"`
		}
		resp.Error = true
		resp.Message = "Email does not registered"
		app.writeJSON(w, http.StatusAccepted, resp)
		return
	}

	link := fmt.Sprintf("%s/reset-password?email=%s", app.config.frontend, payload.Email)

	//utilzado para gerar o token de reset password
	sign := urlsigner.Signer{
		Secret: []byte(app.config.secretKey),
	}

	//url com token para resetar senha
	signedLin := sign.GenerateTokenFromString(link)

	var data struct {
		Link string
	}

	data.Link = signedLin

	err = app.SendEmail("info@widgets.com", payload.Email, "Password Reset Request", "password-reset", data)
	if err != nil {
		app.errorLog.Println("Error senemail")
		app.badRequest(w, r, err)
		return
	}

	var resp struct {
		Error bool `json:"error"`
		Message string `json:"message"`
	}

	resp.Error = false

	app.writeJSON(w, http.StatusCreated, resp)
}

func (app *application) ResetPassword(w http.ResponseWriter, r * http.Request) {
	var payload struct {
		Email string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &payload)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	encripter := encryption.Encryption{
		Key: []byte(app.config.secretKey),
	}
	decriptedEmail,err := encripter.Decrypt(payload.Email)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}
	
	user,err := app.DB.GetUserByEmail(decriptedEmail)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	//criar hash para nova senha
	newHash, err := bcrypt.GenerateFromPassword([]byte(payload.Password), 12)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	err = app.DB.UpdatePasswordForUser(user, string(newHash))
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	var resp struct {
		Error bool `json:"error"`
		Message string `json:"message"`
	}

	resp.Error = false
	resp.Message = "password successefuly changed"

	app.writeJSON(w, http.StatusCreated, resp)
}

func(app *application) AllSales(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PageSize int `json:"page_size"`
		CurrentPage int `json:"page"`
	}
	err := app.readJSON(w,r, &payload)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	//pageSize =2 e page = 1
	allSales,lastPage, totalRecords, err := app.DB.GetAllOrdersPaginated(payload.PageSize,payload.CurrentPage)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	var resp struct {
		CurrentPage int `json:"current_page"`
		PageSize int `json:"page_size"`
		LastPage int `json:"last_page"`
		TotalRecords int `json:"total_records"`
		Orders []*models.Order `json:"orders"`
	}
	resp.CurrentPage = payload.CurrentPage
	resp.PageSize = payload.PageSize
	resp.LastPage = lastPage
	resp.TotalRecords = totalRecords
	resp.Orders = allSales

	app.writeJSON(w, http.StatusOK, resp)
}

func (app *application) AllSubscriptions(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		PageSize int `json:"page_size"`
		CurrentPage int `json:"page"`
	}
	err := app.readJSON(w,r, &payload)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}
	allSubscriptions,lastPage, totalRecords, err := app.DB.GetAllSubscriptionsPaginated(payload.CurrentPage,payload.PageSize)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}
	
	var resp struct {
		CurrentPage int `json:"current_page"`
		PageSize int `json:"page_size"`
		LastPage int `json:"last_page"`
		TotalRecords int `json:"total_records"`
		Orders []*models.Order `json:"orders"`
	}
	resp.CurrentPage = payload.CurrentPage
	resp.PageSize = payload.PageSize
	resp.LastPage = lastPage
	resp.TotalRecords = totalRecords
	resp.Orders = allSubscriptions

	app.writeJSON(w, http.StatusOK, resp)
}

func (app *application) GetSale(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	
	orderID,_ := strconv.Atoi(id)//converter para numero

	order,err := app.DB.GetOrderByID(orderID)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	app.writeJSON(w, http.StatusOK, order)
}

func(app *application) RefundCharge(w http.ResponseWriter, r *http.Request) {
	var chargeToRefund struct {
		ID int `json:"id"`
		PaymentIntent string `json:"pi"`
		Amount int `json:"amount"`
		Currency string `json:"currency"`
	}

	err := app.readJSON(w,r,&chargeToRefund)
	if err != nil {
		app.badRequest(w,r,err)
		return
	}

	//validar amount da order se confere com o de refund
	card := cards.Card{
		Secret: app.config.stripe.secret,
		Key: app.config.stripe.key,
		Currency: chargeToRefund.Currency,
	}
	
	err = card.Refunds(chargeToRefund.PaymentIntent, chargeToRefund.Amount)
	if err != nil {
		app.badRequest(w,r,err)
		return
	}

	//atualizar order para status de refund 2
	err = app.DB.UpdateOrderStatus(chargeToRefund.ID, 2)
	if err != nil{
		app.badRequest(w,r,errors.New("charge refund but database not be updated"))
		return
	}

	var res struct {
		Error bool `json:"error"`
		Message string `json:"message"`
	}
	res.Error = false
	res.Message = "Charge refunded"

	app.writeJSON(w, http.StatusOK,res)
}

func (app *application) CancelSubscription(w http.ResponseWriter, r *http.Request) {
	var subToCancel struct {
		ID int `json:"id"`
		PaymentIntent string `json:"pi"`
		Currency string `json:"currency"`
	}

	err := app.readJSON(w,r, &subToCancel)
	if err != nil{
		app.badRequest(w,r,err)
		return
	}
	card := cards.Card{
		Secret: app.config.stripe.secret,
		Key: app.config.stripe.key,
		Currency: subToCancel.Currency,
	}

	err = card.CancelSubscription(subToCancel.PaymentIntent)
	if err != nil{
		app.badRequest(w,r,err)
		return
	}

	//atualizar order para status de refund 2
	err = app.DB.UpdateOrderStatus(subToCancel.ID, 3)//status 3 é inscricao cancelada
	if err != nil{
		app.badRequest(w,r,errors.New("subscription was cancelled but database not be updated"))
		return
	}

	var res struct {
		Error bool `json:"error"`
		Message string `json:"message"`
	}
	res.Error = false
	res.Message = "Subscription Cancelled"

	app.writeJSON(w, http.StatusOK,res)
}

func (app *application) AllUsers(w http.ResponseWriter, r *http.Request) {
	allUser,err := app.DB.GetAllUsers()
	if err != nil {
		app.badRequest(w,r,err)
		return
	}

	app.writeJSON(w,http.StatusOK,allUser)
}

func (app *application) OneUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userId,_ := strconv.Atoi(id)
	user,err := app.DB.GetUser(userId)
	if err != nil {
		app.badRequest(w,r,err)
		return
	}

	app.writeJSON(w,http.StatusOK,user)
}

func(app *application) EditUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userID,_ := strconv.Atoi(id)

	var user models.User

	err := app.readJSON(w,r,&user)
	if err != nil {
		app.badRequest(w,r,err)
		return
	}
	
	if userID > 0 {
		err := app.DB.EditUser(user)
		if err != nil {
			app.badRequest(w,r,err)
			return
		}
		if user.Password != "" {
			newHash,err := bcrypt.GenerateFromPassword([]byte(user.Password),12)
			if err != nil {
				app.badRequest(w,r,err)
				return
			}

			err = app.DB.UpdatePasswordForUser(user,string(newHash))
			if err != nil {
				app.badRequest(w,r,err)
				return
			}
		}
	} else {
		newHash,err := bcrypt.GenerateFromPassword([]byte(user.Password),12)
		if err != nil {
			app.badRequest(w,r,err)
			return
		}
		err = app.DB.AddtUser(user, string(newHash))
		if err != nil {
			app.badRequest(w,r,err)
			return
		}
	}

	var resp struct {
		Err bool `json:"error"`
		Message string `json:"message"`
	}
	resp.Err = false
	app.writeJSON(w,http.StatusOK, resp)
}

func(app *application) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	userId,_ := strconv.Atoi(id)
	err := app.DB.DeleteUser(userId)
	if err != nil {
		app.badRequest(w,r,err)
		return
	}

	var resp struct {
		Err bool `json:"error"`
		Message string `json:"message"`
	}
	resp.Err = false
	app.writeJSON(w,http.StatusOK, resp)
}

