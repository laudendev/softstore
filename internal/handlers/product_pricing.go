package handlers

import (
	"database/sql"
	"fmt"

	"softstore/internal/db"
	"softstore/internal/models"
	"softstore/internal/payments"
)

// deviceDiscountTiers maps a device count to its discount fraction off
// the per-device base price. Tiers are stepped (not a smooth formula) so
// the pricing is easy to state plainly to a buyer: "10-14 devices: 25%
// off," etc. Capped at 35% — even a 24-device purchase remains solidly
// profitable, since there's no real per-device cost increase behind the
// scenes to justify discounting further.
func deviceDiscountTiers(seats int64) float64 {
	switch {
	case seats <= 1:
		return 0
	case seats == 2:
		return 0.10
	case seats <= 4:
		return 0.15
	case seats <= 9:
		return 0.20
	case seats <= 14:
		return 0.25
	default: // 15-24
		return 0.35
	}
}

// GetOrCreatePriceForSeats returns the Stripe Price ID a buyer should be
// charged for purchasing the given product at the given seat count. The
// product's own stripe_price_id (registered at creation time) is always
// the 1-seat price. For any other seat count, this looks up an existing
// product_prices row; if none exists yet, it creates one on the fly via
// the payment provider (base price * seats) and records it for reuse by
// future buyers picking the same tier — so a given product+seats
// combination only ever gets ONE genuinely equivalent Stripe Price,
// never a fresh duplicate per purchase.
func GetOrCreatePriceForSeats(conn *sql.DB, provider payments.Provider, product *models.Product, seats int64) (string, error) {
	if seats <= 0 {
		seats = 1
	}
	if seats == 1 {
		return product.StripePriceID, nil
	}

	existing, err := db.GetProductPrice(conn, product.ID, seats)
	if err == nil {
		return existing.StripePriceID, nil
	}
	if err != sql.ErrNoRows {
		return "", fmt.Errorf("look up existing product price: %w", err)
	}

	discount := deviceDiscountTiers(seats)
	perDeviceCents := int64(float64(product.PriceCents) * (1 - discount))
	totalCents := perDeviceCents * seats

	registered, err := provider.AddPrice(payments.AdditionalPrice{
		ProviderProductID: product.StripeProductID,
		PriceCents:        totalCents,
		Currency:          "usd",
	})

	if err != nil {
		return "", fmt.Errorf("create provider price for %d seats: %w", seats, err)
	}

	if _, err := db.CreateProductPrice(conn, product.ID, seats, registered.ProviderItemID); err != nil {
		return "", fmt.Errorf("record new product price: %w", err)
	}

	return registered.ProviderItemID, nil
}
