package models

import "testing"

func TestLineTotalCentsAccountsForSeats(t *testing.T) {
	item := CartItem{
		Quantity: 2,
		Seats:    3,
		Product:  Product{PriceCents: 1000},
	}
	got := item.LineTotalCents()
	want := int64(5100) // 1000 * 0.85 (15% off at 3 seats) * 3 seats * 2 quantity
	if got != want {
		t.Errorf("expected %d, got %d", want, got)
	}
}

func TestLineTotalCentsDefaultsSeatsToOne(t *testing.T) {
	item := CartItem{
		Quantity: 2,
		Seats:    0, // unset, should behave as 1
		Product:  Product{PriceCents: 1000},
	}
	got := item.LineTotalCents()
	want := int64(2000)
	if got != want {
		t.Errorf("expected %d, got %d", want, got)
	}
}

func TestCartTotalCentsSumsMultipleMultiSeatItems(t *testing.T) {
	cart := Cart{
		Items: []CartItem{
			{Quantity: 1, Seats: 1, Product: Product{PriceCents: 500}},  // 500, no discount at 1 seat
			{Quantity: 2, Seats: 3, Product: Product{PriceCents: 1000}}, // 5100, 15% off at 3 seats
		},
	}
	got := cart.TotalCents()
	want := int64(5600)
	if got != want {
		t.Errorf("expected %d, got %d", want, got)
	}
}

