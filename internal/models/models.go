package models

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
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
	IsRecurring bool `json:"is_recurring"`
	PlanID string `json:"plan_id"`
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
	Widget Widget `json:"widget"`
	Transaction Transaction `json:"transaction"`
	Customer Customer `json:"customer"`
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
		`select id, name, description, inventory_level, price, coalesce(image, ''), is_recurring, plan_id, created_at, updated_at from widgets where id= ?`, id)
	
	err := row.Scan(
		&widget.ID, 
		&widget.Name,
		&widget.Description,
		&widget.InventoryLevel,
		&widget.Price,
		&widget.Image,
		&widget.IsRecurring,
		&widget.PlanID,
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

func (m *DbModel) GetUserByEmail(email string) (User, error) {
	ctx,cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()

	email = strings.ToLower(email)
	var u User

	row := m.DB.QueryRowContext(ctx, `
		select id,first_name,last_name,email,password,created_at,updated_at
		from users where email=?`, email)
	
	err := row.Scan(
		&u.ID,
		&u.FirstName,
		&u.LastName,
		&u.Email,
		&u.Password,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		return u, err
	}
	return u, nil
}

func (m *DbModel) Authenticate(email, password string) (int, error) {
	ctx,cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()

	var id int
	var hashedPassword string

	row := m.DB.QueryRowContext(ctx, "select id, password from users where email=?", email)
	err := row.Scan(&id, &hashedPassword)
	if err != nil {
		return id, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err == bcrypt.ErrMismatchedHashAndPassword{
		return 0, errors.New("invalid credential")
	} else if err != nil {
		return 0, err
	}

	return id, nil
}

func(m *DbModel) UpdatePasswordForUser(u User, hashPassword string) error {
	ctx,cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()

	stmt := `update users set password = ? where id = ?`
	_,err := m.DB.ExecContext(ctx, stmt, hashPassword, u.ID)
	if err != nil {
		return err
	}

	return nil
}

func (m *DbModel) GetAllOrders() ([]*Order, error) {
	ctx,cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()

	var orders []*Order

	query := `
	select
		o.id, o.widget_id, o.transaction_id, o.customer_id, 
		o.status_id, o.quantity, o.amount, o.created_at,
		o.updated_at, w.id, w.name, t.id, t.amount, t.currency,
		t.last_four, t.expiry_month, t.expiry_year, t.payment_intent,
		t.bank_return_code, c.id, c.first_name, c.last_name, c.email	
	
	from
		orders o
		left join widgets w on (o.widget_id = w.id)
		left join transactions t on (o.transaction_id = t.id)
		left join customers c on (o.customer_id = c.id)
	where
		w.is_recurring = 0
	order by
		o.created_at desc
	`

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var o Order
		err = rows.Scan(
			&o.ID,
			&o.WidgetID,
			&o.TransactionID,
			&o.CustomerID,
			&o.StatusID,
			&o.Quantity,
			&o.Amount,
			&o.CreatedAt,
			&o.UpdatedAt,
			&o.Widget.ID,
			&o.Widget.Name,
			&o.Transaction.ID,
			&o.Transaction.Amount,
			&o.Transaction.Currency,
			&o.Transaction.LastFour,
			&o.Transaction.ExpiryMonth,
			&o.Transaction.ExpiryYear,
			&o.Transaction.PaymentIntent,
			&o.Transaction.BankReturnCode,
			&o.Customer.ID,
			&o.Customer.FirstName,
			&o.Customer.LastName,
			&o.Customer.Email,
		)
		if err != nil {
			return nil,err
		}

		orders = append(orders, &o)
	}
	return orders, nil
}

func (m *DbModel) GetAllSubscriptions() ([]*Order, error) {
	ctx,cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()

	var orders []*Order

	query := `
	select
		o.id, o.widget_id, o.transaction_id, o.customer_id, 
		o.status_id, o.quantity, o.amount, o.created_at,
		o.updated_at, w.id, w.name, t.id, t.amount, t.currency,
		t.last_four, t.expiry_month, t.expiry_year, t.payment_intent,
		t.bank_return_code, c.id, c.first_name, c.last_name, c.email	
	
	from
		orders o
		left join widgets w on (o.widget_id = w.id)
		left join transactions t on (o.transaction_id = t.id)
		left join customers c on (o.customer_id = c.id)
	where
		w.is_recurring = 1
	order by
		o.created_at desc
	`

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var o Order
		err = rows.Scan(
			&o.ID,
			&o.WidgetID,
			&o.TransactionID,
			&o.CustomerID,
			&o.StatusID,
			&o.Quantity,
			&o.Amount,
			&o.CreatedAt,
			&o.UpdatedAt,
			&o.Widget.ID,
			&o.Widget.Name,
			&o.Transaction.ID,
			&o.Transaction.Amount,
			&o.Transaction.Currency,
			&o.Transaction.LastFour,
			&o.Transaction.ExpiryMonth,
			&o.Transaction.ExpiryYear,
			&o.Transaction.PaymentIntent,
			&o.Transaction.BankReturnCode,
			&o.Customer.ID,
			&o.Customer.FirstName,
			&o.Customer.LastName,
			&o.Customer.Email,
		)
		if err != nil {
			return nil,err
		}

		orders = append(orders, &o)
	}
	return orders, nil
}

func (m *DbModel) GetOrderByID(orderID int) (Order, error) {
	ctx,cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()

	query := `
		select
			o.id, o.widget_id, o.transaction_id, o.customer_id, 
			o.status_id, o.quantity, o.amount, o.created_at,
			o.updated_at, w.id, w.name, t.id, t.amount, t.currency,
			t.last_four, t.expiry_month, t.expiry_year, t.payment_intent,
			t.bank_return_code, c.id, c.first_name, c.last_name, c.email	
		
		from
			orders o
			left join widgets w on (o.widget_id = w.id)
			left join transactions t on (o.transaction_id = t.id)
			left join customers c on (o.customer_id = c.id)
		where
			o.id = ?
	`

	row := m.DB.QueryRowContext(ctx, query, orderID)
	
	var o Order
	err := row.Scan(
		&o.ID,
		&o.WidgetID,
		&o.TransactionID,
		&o.CustomerID,
		&o.StatusID,
		&o.Quantity,
		&o.Amount,
		&o.CreatedAt,
		&o.UpdatedAt,
		&o.Widget.ID,
		&o.Widget.Name,
		&o.Transaction.ID,
		&o.Transaction.Amount,
		&o.Transaction.Currency,
		&o.Transaction.LastFour,
		&o.Transaction.ExpiryMonth,
		&o.Transaction.ExpiryYear,
		&o.Transaction.PaymentIntent,
		&o.Transaction.BankReturnCode,
		&o.Customer.ID,
		&o.Customer.FirstName,
		&o.Customer.LastName,
		&o.Customer.Email,
	)
	if err != nil {
		return o,err
	}

	
	return o, nil
}

func (m *DbModel) UpdateOrderStatus(id, statusID int) error {
	ctx,cancel := context.WithTimeout(context.Background(), 3 * time.Second)
	defer cancel()

	stmt := "update orders set status_id=? where id = ?"

	_, err := m.DB.ExecContext(ctx,stmt, statusID, id)
	if err !=  nil {
		return err
	}
	return nil
}