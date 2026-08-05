package models

import (
	"fmt"
	"time"
)

type Product struct {
	ID              int64
	Name            string
	Slug            string
	Description     string
	PriceCents      int64
	StripePriceID   string
	StripeProductID string
	ProductCode     string
	StubURL         string
	TaxCode         string
	PreviewVideoURL string
	Seats           int64
	CreatedAt       time.Time
}

func (p Product) PriceDollars() string {
	return formatCents(p.PriceCents)
}

func formatCents(cents int64) string {
	return fmt.Sprintf("%.2f", float64(cents)/100)
}
