package handlers

import (
	"errors"
	"testing"

	"softstore/internal/db"
	"softstore/internal/models"
	"softstore/internal/payments"
	"softstore/internal/payments/mockprovider"
)

var errTestProviderDown = errors.New("provider is down")

func TestGetOrCreatePriceForSeatsReturnsBasePriceFor1Seat(t *testing.T) {
	conn := newTestDBWithSchema(t)
	mock := mockprovider.New()

	p := &models.Product{
		Name: "Widget", Slug: "widget", PriceCents: 1000,
		StripePriceID: "price_base", StripeProductID: "prod_base",
		ProductCode: "WGET", TaxCode: "txcd_10202000",
	}
	if err := db.CreateProduct(conn, p); err != nil {
		t.Fatalf("seed product: %v", err)
	}

	priceID, err := GetOrCreatePriceForSeats(conn, mock, p, 1)
	if err != nil {
		t.Fatalf("GetOrCreatePriceForSeats failed: %v", err)
	}
	if priceID != "price_base" {
		t.Errorf("expected base price id for 1 seat, got %q", priceID)
	}
	if len(mock.AddPriceCalls) != 0 {
		t.Error("expected provider NOT to be called for the 1-seat tier")
	}
}

func TestGetOrCreatePriceForSeatsCreatesNewTier(t *testing.T) {
	conn := newTestDBWithSchema(t)
	mock := mockprovider.New()

	p := &models.Product{
		Name: "Widget", Slug: "widget-2", PriceCents: 1000,
		StripePriceID: "price_base2", StripeProductID: "prod_base2",
		ProductCode: "WGT2", TaxCode: "txcd_10202000",
	}
	if err := db.CreateProduct(conn, p); err != nil {
		t.Fatalf("seed product: %v", err)
	}

	priceID, err := GetOrCreatePriceForSeats(conn, mock, p, 3)
	if err != nil {
		t.Fatalf("GetOrCreatePriceForSeats failed: %v", err)
	}
	if priceID == "" || priceID == "price_base2" {
		t.Errorf("expected a distinct new price id for 3 seats, got %q", priceID)
	}
	if len(mock.AddPriceCalls) != 1 {
		t.Fatalf("expected 1 AddPrice call, got %d", len(mock.AddPriceCalls))
	}
	call := mock.AddPriceCalls[0]
	if call.PriceCents != 3000 {
		t.Errorf("expected 3000 cents (1000 * 3 seats), got %d", call.PriceCents)
	}
	if call.ProviderProductID != "prod_base2" {
		t.Errorf("expected product id 'prod_base2', got %q", call.ProviderProductID)
	}

	saved, err := db.GetProductPrice(conn, p.ID, 3)
	if err != nil {
		t.Fatalf("expected product_prices row to be saved: %v", err)
	}
	if saved.StripePriceID != priceID {
		t.Errorf("expected saved price id to match returned price id, got %q vs %q", saved.StripePriceID, priceID)
	}
}

func TestGetOrCreatePriceForSeatsReusesExistingTier(t *testing.T) {
	conn := newTestDBWithSchema(t)
	mock := mockprovider.New()

	p := &models.Product{
		Name: "Widget", Slug: "widget-3", PriceCents: 1000,
		StripePriceID: "price_base3", StripeProductID: "prod_base3",
		ProductCode: "WGT3", TaxCode: "txcd_10202000",
	}
	if err := db.CreateProduct(conn, p); err != nil {
		t.Fatalf("seed product: %v", err)
	}

	first, err := GetOrCreatePriceForSeats(conn, mock, p, 5)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	second, err := GetOrCreatePriceForSeats(conn, mock, p, 5)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if first != second {
		t.Errorf("expected the same price id on repeat calls, got %q and %q", first, second)
	}
	if len(mock.AddPriceCalls) != 1 {
		t.Errorf("expected AddPrice to be called only once (first time), got %d calls", len(mock.AddPriceCalls))
	}
}

func TestGetOrCreatePriceForSeatsProviderError(t *testing.T) {
	conn := newTestDBWithSchema(t)
	mock := mockprovider.New()
	mock.AddPriceFunc = func(payments.AdditionalPrice) (payments.RegisteredItem, error) {
		return payments.RegisteredItem{}, errTestProviderDown
	}

	p := &models.Product{
		Name: "Widget", Slug: "widget-4", PriceCents: 1000,
		StripePriceID: "price_base4", StripeProductID: "prod_base4",
		ProductCode: "WGT4", TaxCode: "txcd_10202000",
	}
	if err := db.CreateProduct(conn, p); err != nil {
		t.Fatalf("seed product: %v", err)
	}

	_, err := GetOrCreatePriceForSeats(conn, mock, p, 2)
	if err == nil {
		t.Error("expected an error when the provider fails")
	}

	if _, lookupErr := db.GetProductPrice(conn, p.ID, 2); lookupErr == nil {
		t.Error("expected no product_prices row to be saved when provider call fails")
	}
}
