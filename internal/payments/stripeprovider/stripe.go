package stripeprovider

import (
	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/checkout/session"
	"github.com/stripe/stripe-go/v82/price"
	"github.com/stripe/stripe-go/v82/product"

	"softstore/internal/payments"
)

// StripeProvider implements payments.Provider using the real Stripe API.
type StripeProvider struct{}

func New() *StripeProvider {
	return &StripeProvider{}
}

func (p *StripeProvider) RegisterItem(item payments.SellableItem) (payments.RegisteredItem, error) {
	stripeProd, err := product.New(&stripe.ProductParams{
		Name:        stripe.String(item.Name),
		Description: stripe.String(item.Description),
		TaxCode:     stripe.String(item.TaxCategory),
	})
	if err != nil {
		return payments.RegisteredItem{}, err
	}

	currency := item.Currency
	if currency == "" {
		currency = string(stripe.CurrencyUSD)
	}

	stripePrice, err := price.New(&stripe.PriceParams{
		Product:    stripe.String(stripeProd.ID),
		UnitAmount: stripe.Int64(item.PriceCents),
		Currency:   stripe.String(currency),
	})
	if err != nil {
		return payments.RegisteredItem{}, err
	}

	return payments.RegisteredItem{ProviderItemID: stripePrice.ID}, nil
}

func (p *StripeProvider) StartPurchase(req payments.PurchaseRequest) (payments.Purchase, error) {
	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(req.ProviderItemID),
				Quantity: stripe.Int64(req.Quantity),
			},
		},
		Metadata:   req.Metadata,
		SuccessURL: stripe.String(req.SuccessURL),
		CancelURL:  stripe.String(req.CancelURL),
	}

	s, err := session.New(params)
	if err != nil {
		return payments.Purchase{}, err
	}

	return payments.Purchase{RedirectURL: s.URL}, nil
}
