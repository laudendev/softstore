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
	default: // 5-6
		return 0.20
	}
}

// UpdateCheckoutProductDescription sets the Stripe product's name and
// description to reflect the specific device count and volume discount
// about to be purchased, so Stripe's own checkout page shows this
// clearly rather than a generic product name. Only meaningful for
// multi-device purchases — a 1-device purchase uses the product's
// normal registered name/description as-is, so this is a no-op then.
func UpdateCheckoutProductDescription(provider payments.Provider, product *models.Product, seats int64) error {
	if seats <= 1 {
		return nil
	}
	discount := deviceDiscountTiers(seats)
	name := fmt.Sprintf("%s (%d devices)", product.Name, seats)
	description := fmt.Sprintf("%d-device license for %s", seats, product.Description)
	if discount > 0 {
		description = fmt.Sprintf("%s — %.0f%% volume discount applied", description, discount*100)
	}
	return provider.UpdateProductDescription(product.StripeProductID, name, description)
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

	name := fmt.Sprintf("%s (%d devices)", product.Name, seats)
	description := fmt.Sprintf("%d-device license for %s", seats, product.Description)
	if discount > 0 {
		description = fmt.Sprintf("%s — %.0f%% volume discount applied", description, discount*100)
	}

	// Each seat tier gets its own dedicated Stripe Product+Price, created
	// once and reused forever — not a Price attached to the shared base
	// Product. Sharing one Product across tiers and mutating its
	// description at checkout time doesn't work: a cart containing
	// multiple tiers of the same product needs each line item to show
	// its own tier correctly at the same time, which a single mutable
	// Product description can never do.
	registered, err := provider.RegisterItem(payments.SellableItem{
		Name:        name,
		Description: description,
		PriceCents:  totalCents,
		Currency:    "usd",
		TaxCategory: product.TaxCode,
	})
	if err != nil {
		return "", fmt.Errorf("create provider product+price for %d seats: %w", seats, err)
	}
	
	if _, err := db.CreateProductPrice(conn, product.ID, seats, registered.ProviderItemID); err != nil {
		return "", fmt.Errorf("record new product price: %w", err)
	}

	return registered.ProviderItemID, nil
}
