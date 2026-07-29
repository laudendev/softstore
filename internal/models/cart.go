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
	Product    Product
	CreatedAt  time.Time
}

// TotalCents sums quantity * price across every item in the cart.
func (c Cart) TotalCents() int64 {
	var total int64
	for _, item := range c.Items {
		total += item.Quantity * item.Product.PriceCents
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
