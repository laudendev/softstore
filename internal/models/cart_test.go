package models

import "testing"

func TestLineTotalCentsAccountsForSeats(t *testing.T) {
	item := CartItem{
		Quantity: 2,
		Seats:    3,
		Product:  Product{PriceCents: 1000},
	}
	got := item.LineTotalCents()
	want := int64(6000) // 1000 * 3 seats * 2 quantity
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
			{Quantity: 1, Seats: 1, Product: Product{PriceCents: 500}},  // 500
			{Quantity: 2, Seats: 3, Product: Product{PriceCents: 1000}}, // 6000
		},
	}
	got := cart.TotalCents()
	want := int64(6500)
	if got != want {
		t.Errorf("expected %d, got %d", want, got)
	}
}
