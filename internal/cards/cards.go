package cards

import (
	"github.com/stripe/stripe-go/v72"
	"github.com/stripe/stripe-go/v72/paymentintent"
	"github.com/stripe/stripe-go/v72/paymentmethod"
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