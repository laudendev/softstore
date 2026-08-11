package models

import "time"

type Cart struct {
	ID        int64
	Token     string
	CreatedAt time.Time
	Items     []CartItem
}

// CartItem is a cart_items row joined with its product's display fields,
// so templates can render name/price without a second query per item.
type CartItem struct {
	ID         int64
	CartID     int64
	ProductID  int64
	Quantity   int64
	Seats      int64
	Product    Product
	CreatedAt  time.Time
}

// DeviceDiscountTiers maps a device count to its discount fraction off
// the per-device base price. Shared between cart display math and the
// checkout pricing logic (internal/handlers/product_pricing.go), so
// both layers apply the exact same discount — this is the single
// source of truth for the tier table.
func DeviceDiscountTiers(seats int64) float64 {
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

// LineTotalCents is one cart item's total price: discounted per-device
// price * seats * quantity — e.g. a $10 product at 3 seats (15% off)
// and quantity 2 costs (10 * 0.85 * 3) * 2 = $51, not $60.
func (ci CartItem) LineTotalCents() int64 {
	seats := ci.Seats
	if seats <= 0 {
		seats = 1
	}
	discount := DeviceDiscountTiers(seats)
	perDeviceCents := int64(float64(ci.Product.PriceCents) * (1 - discount))
	return perDeviceCents * seats * ci.Quantity
}

// LineTotalDollars formats LineTotalCents as a dollar string.
func (ci CartItem) LineTotalDollars() string {
	return formatCents(ci.LineTotalCents())
}

func (c Cart) TotalCents() int64 {
	var total int64
	for _, item := range c.Items {
		total += item.LineTotalCents()
	}
	return total
}

// TotalDollars formats TotalCents as a dollar string, matching Product.PriceDollars.
func (c Cart) TotalDollars() string {
	return formatCents(c.TotalCents())
}

// ItemCount sums quantities across all items (not just distinct products).
func (c Cart) ItemCount() int64 {
	var count int64
	for _, item := range c.Items {
		count += item.Quantity
	}
	return count
}
