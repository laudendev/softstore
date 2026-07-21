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
	ProductCode     string
	StubURL         string
	TaxCode         string
	PreviewVideoURL string
	CreatedAt       time.Time
}


func (p Product) PriceDollars() string {
	return fmt.Sprintf("%.2f", float64(p.PriceCents)/100)
}
