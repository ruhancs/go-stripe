package cards

import (

	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/customer"
	"github.com/stripe/stripe-go/v72/paymentintent"
	"github.com/stripe/stripe-go/v72/paymentmethod"
	"github.com/stripe/stripe-go/v72/refund"
	"github.com/stripe/stripe-go/v72/sub"
)

type Card struct {
	Secret string
	Key string
	Currency string
}

type Transaction struct {
	TransactionStatusId int
	Amount int
	Currrency string
	LastFour string
	BankReturnCode string
}

func (c *Card) Charge(currency string, amount int) (*stripe.PaymentIntent, string, error) {
	return c.CreatePaymenIntent(currency,amount)
}

func (c *Card) CreatePaymenIntent(currency string, amount int) (*stripe.PaymentIntent, string, error) {
	stripe.Key = c.Secret //secret key do stripe

	//create payment intent
	params := &stripe.PaymentIntentParams{
		//converter o int para 64 para o stripe
		Amount: stripe.Int64(int64(amount)),
		Currency: stripe.String(currency),
	}

	//se quiser adicionar alguma informacao na transacao
	//params.AddMetadata("key","value")

	paymentIntent,err := paymentintent.New(params)
	if err!= nil {
		msg := ""
		if stripeErr, ok := err.(*stripe.Error); ok {
			msg = cardErrorMsg(stripeErr.Code)
		}
		return nil, msg, err
	}
	return paymentIntent, "", nil
}

//pegar o metodo de pagamento pelo payment Intent Id
func (c *Card) GetPaymentMethod( s string) (*stripe.PaymentMethod, error) {
	stripe.Key = c.Secret

	paymentMeth,err := paymentmethod.Get(s, nil)
	if err != nil{
		return nil, err
	}
	return paymentMeth,nil
}

//pegar um payment intent existent pelo id
func(c *Card) GetPaymentIntent(id string) (*stripe.PaymentIntent, error) {
	stripe.Key = c.Secret

	paymentInt,err := paymentintent.Get(id,nil)
	if err != nil{
		return nil, err
	}
	return paymentInt, nil
}

//subscrever o customer no plano
func(c *Card) SubscribeToPlan(cust *stripe.Customer, plan, email, last4, cardType string) (*stripe.Subscription, error) {
	stripeCustomerID := cust.ID
	items := []*stripe.SubscriptionItemsParams{
		{Plan: stripe.String(plan)},
	}

	params := &stripe.SubscriptionParams{
		Customer: stripe.String(stripeCustomerID),
		Items: items,
	}

	params.AddMetadata("last_four", last4)
	params.AddMetadata("card_type", cardType)
	params.AddExpand("latest_invoice.payment_intent")
	subscription, err := sub.New(params)
	if err != nil {
		return nil, err
	}
	return subscription,nil
}

//criar um customer no dashbioard do stripe
func (c *Card) CreateCustomer(paymentMethod, email string) (*stripe.Customer, string, error) {
	stripe.Key = c.Secret
	customerParams := &stripe.CustomerParams{
		PaymentMethod: stripe.String(paymentMethod),
		Email: stripe.String(email),
		InvoiceSettings: &stripe.CustomerInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(paymentMethod),
		},
	}

	cust,err := customer.New(customerParams)
	if err != nil {
		msg:= ""
		if stripeErr, ok := err.(*stripe.Error); ok {
			msg = cardErrorMsg(stripeErr.Code)
		}
		return nil, msg, err
	}
	return cust, "", nil
}

func(c *Card) Refunds(pi string, amount int) error {
	stripe.Key = c.Secret
	amountToRefund := int64(amount)

	refundParams := &stripe.RefundParams{
		Amount: &amountToRefund,
		PaymentIntent: &pi,
	}

	_, err := refund.New(refundParams)
	if err != nil {
		return err
	}

	return nil
}

func (c *Card) CancelSubscription(subID string) error {
	stripe.Key = c.Secret

	params := &stripe.SubscriptionParams{
		CancelAtPeriodEnd: stripe.Bool(true),
	}

	_, err := sub.Update(subID, params)
	if err != nil {
		return err
	}
	return nil
}

func cardErrorMsg(code stripe.ErrorCode) string {
	var msg = ""

	switch code {
	case stripe.ErrorCodeCardDeclined:
		msg = "Your Card was declined"
	case stripe.ErrorCodeExpiredCard:
		msg = "Your Card is expired"
	case stripe.ErrorCodeIncorrectCVC:
		msg = "Incorrect CVC code"
	case stripe.ErrorCodeIncorrectZip:
		msg = "Incorrect Zip/Postal code"
	case stripe.ErrorCodeAmountTooLarge:
		msg = "Amount too large to charge to your card"
	case stripe.ErrorCodeAmountTooSmall:
		msg = "Amount too smal to charge to your card"
	case stripe.ErrorCodeBalanceInsufficient:
		msg = "balance insuficient"
	case stripe.ErrorCodePostalCodeInvalid:
		msg = "your postal code is invalid"
	default:
		msg = "Your Card was declined"
	}
	return msg
}
