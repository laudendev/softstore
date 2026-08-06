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

// LineTotalCents is one cart item's total price: base price * seats *
// quantity — e.g. a $10 product at 3 seats, quantity 2, costs $60.
func (ci CartItem) LineTotalCents() int64 {
	seats := ci.Seats
	if seats <= 0 {
		seats = 1
	}
	return ci.Product.PriceCents * seats * ci.Quantity
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
