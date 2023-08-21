package models

import (
	"context"
	"database/sql"
	"time"
)

//DbModel é o tipo para conexao do database com os valores
type DbModel struct {
	DB *sql.DB
}

//Models é o envolucro de todos models
type Models struct {
	DB DbModel
}

func NewwModels(db *sql.DB) Models {
	return Models{
		DB: DbModel{DB: db},
	}
}

//tipo de todos widgets para criar os produtos para vender
type Widget struct {
	ID int `json:"id"`
	Name string `json:"name"`
	Description string `json:"description"`
	InventoryLevel int `json:"inventory_level"`
	Price int `json:"price"`
	Image string `json:"image"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

// order table
type Order struct {
	ID int `json:"id"`
	WidgetID int `json:"widget_id"`
	TransactionID int `json:"transction_id"`
	CustomerID int `json:"customer_id"`
	StatusID int `json:"status_id"`
	Quantity int `json:"quantity"`
	Amount int `json:"amount"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

//tabela status
type Status struct {
	ID int `json:"id"`
	Name string `json:"name"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

type Transaction struct {
	ID int `json:"id"`
	Amount int `json:"amount"`
	Currency string `json:"currency"`
	LastFour string `json:"last_four"`
	ExpiryMonth int `json:"expire_month"`
	ExpiryYear int `json:"expire_year"`
	PaymentIntent string `json:"payment_intent"`
	PaymentMethod string `json:"payment_method"`
	BankReturnCode string `json:"bank_return_code"`
	TarnsactionStatusID int `json:"transaction_status_id"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

type User struct {
	ID int `json:"id"`
	FirstName string `json:"first_name"`
	LastName string `json:"last_name"`
	Email string `json:"email"`
	Password string `json:"password"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

type Customer struct {
	ID int `json:"id"`
	FirstName string `json:"first_name"`
	LastName string `json:"last_name"`
	Email string `json:"email"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

type TransactionStatus struct {
	ID int `json:"id"`
	Name string `json:"name"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

func (m *DbModel) GetWidget(id int) (Widget, error) {
	//se demorar mais de 3 segundos algo esta errado no contexto para o db
	ctx,cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()

	var widget Widget

	row := m.DB.QueryRowContext(ctx, 
		`select id, name, description, inventory_level, price, coalesce(image, ''), created_at, updated_at from widgets where id= ?`, id)
	
	err := row.Scan(
		&widget.ID, 
		&widget.Name,
		&widget.Description,
		&widget.InventoryLevel,
		&widget.Price,
		&widget.Image,
		&widget.CreatedAt,
		&widget.UpdatedAt,
	)
	if err != nil {
		return widget, err
	}

	return widget, nil
}

func (m *DbModel) InsertTransaction(transaction Transaction) (int, error) {
	//se demorar mais de 3 segundos algo esta errado no contexto para o db
	ctx,cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()

	stmt := `
		insert into transactions (amount, currency, last_four, bank_return_code, expiry_month, expiry_year, payment_intent,
			payment_method, transaction_status_id, created_at, updated_at)
		values(?,?,?,?,?,?,?,?,?,?,?)
	`

	result,err := m.DB.ExecContext(ctx, stmt,
		transaction.Amount,
		transaction.Currency,
		transaction.LastFour,
		transaction.BankReturnCode,
		transaction.ExpiryMonth,
		transaction.ExpiryYear,
		transaction.PaymentIntent,
		transaction.PaymentMethod,
		transaction.TarnsactionStatusID,
		time.Now(),
		time.Now(),
	)
	if err != nil {
		return 0, err
	}
	id,err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), err
}

func (m *DbModel) InsertOrder(order Order) (int, error) {
	//se demorar mais de 3 segundos algo esta errado no contexto para o db
	ctx,cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()

	stmt := `
		insert into orders (widget_id, transaction_id, status_id, quantity, customer_id, amount, created_at, updated_at)
		values(?,?,?,?,?,?,?,?)
	`

	result,err := m.DB.ExecContext(ctx, stmt,
		order.WidgetID,
		order.TransactionID,
		order.StatusID,
		order.Quantity,
		order.CustomerID,
		order.Amount,
		time.Now(),
		time.Now(),
	)
	if err != nil {
		return 0, err
	}
	id,err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), err
}

func (m *DbModel) InsertCustomer(customer Customer) (int, error) {
	//se demorar mais de 3 segundos algo esta errado no contexto para o db
	ctx,cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()

	stmt := `
		insert into customers (first_name, last_name, email, created_at, updated_at)
		values(?,?,?,?,?)
	`

	result,err := m.DB.ExecContext(ctx, stmt,
		customer.FirstName,
		customer.LastName,
		customer.Email,
		time.Now(),
		time.Now(),
	)
	if err != nil {
		return 0, err
	}
	id,err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return int(id), err
}