package models

import "time"

// ProductPrice maps a product and a seat-tier to the specific Stripe
// Price ID that charges base_price * seats for that combination. Each
// product's 1-seat tier is its "default" price, created alongside the
// product itself; other tiers (2, 3, or a custom count up to 24) are
// created on demand the first time a buyer selects them, then reused.
type ProductPrice struct {
	ID            int64
	ProductID     int64
	Seats         int64
	StripePriceID string
	CreatedAt     time.Time
}
