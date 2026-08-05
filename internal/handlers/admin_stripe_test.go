package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"softstore/internal/db"
	"softstore/internal/payments"
	"softstore/internal/payments/mockprovider"
)

func TestAdminCreateProductSuccess(t *testing.T) {
	conn := newTestDBWithSchema(t)
	mock := mockprovider.New()

	form := url.Values{
		"name":         {"Test Widget"},
		"slug":         {"test-widget"},
		"description":  {"A widget."},
		"price":        {"19.99"},
		"product_code": {"TWDG"},
		"stub_url":     {"https://example.com/stub.zip"},
		"tax_code":     {"txcd_10202000"},
		"seats":       {"1"},
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/products", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	AdminCreateProduct(conn, mock)(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Created \"Test Widget\" successfully") {
		t.Errorf("expected success message, got body: %s", w.Body.String())
	}

	// Confirm the provider was called with correctly translated data.
	if len(mock.RegisterItemCalls) != 1 {
		t.Fatalf("expected 1 RegisterItem call, got %d", len(mock.RegisterItemCalls))
	}
	call := mock.RegisterItemCalls[0]
	if call.Name != "Test Widget" {
		t.Errorf("expected provider call name 'Test Widget', got %q", call.Name)
	}
	if call.PriceCents != 1999 {
		t.Errorf("expected provider call price 1999 cents, got %d", call.PriceCents)
	}
	if call.TaxCategory != "txcd_10202000" {
		t.Errorf("expected tax category txcd_10202000, got %q", call.TaxCategory)
	}

	// Confirm it was actually persisted locally too.
	saved, err := db.GetProductBySlug(conn, "test-widget")
	if err != nil {
		t.Fatalf("expected product to be saved, got error: %v", err)
	}
	if saved.StripePriceID != "mock_item_id" {
		t.Errorf("expected saved StripePriceID to be the provider's returned ID, got %q", saved.StripePriceID)
	}
}

func TestAdminCreateProductInvalidProductCode(t *testing.T) {
	conn := newTestDBWithSchema(t)
	mock := mockprovider.New()

	form := url.Values{
		"name":         {"Test Widget"},
		"slug":         {"test-widget"},
		"price":        {"19.99"},
		"product_code": {"TOOLONG"}, // not exactly 4 chars
		"tax_code":     {"txcd_10202000"},
		"seats":       {"1"},
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/products", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	AdminCreateProduct(conn, mock)(w, req)

	if !strings.Contains(w.Body.String(), "must be exactly 4 characters") {
		t.Errorf("expected product code validation error, got: %s", w.Body.String())
	}
	if len(mock.RegisterItemCalls) != 0 {
		t.Error("expected provider NOT to be called when validation fails")
	}
}

func TestAdminCreateProductInvalidPrice(t *testing.T) {
	conn := newTestDBWithSchema(t)
	mock := mockprovider.New()

	form := url.Values{
		"name":         {"Test Widget"},
		"slug":         {"test-widget"},
		"price":        {"not-a-number"},
		"product_code": {"TWDG"},
		"tax_code":     {"txcd_10202000"},
		"seats":       {"1"},
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/products", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	AdminCreateProduct(conn, mock)(w, req)

	if !strings.Contains(w.Body.String(), "Invalid price") {
		t.Errorf("expected invalid price error, got: %s", w.Body.String())
	}
	if len(mock.RegisterItemCalls) != 0 {
		t.Error("expected provider NOT to be called when price is invalid")
	}
}

func TestAdminCreateProductProviderError(t *testing.T) {
	conn := newTestDBWithSchema(t)
	mock := mockprovider.New()
	mock.RegisterItemFunc = func(payments.SellableItem) (payments.RegisteredItem, error) {
		return payments.RegisteredItem{}, errors.New("provider is down")
	}

	form := url.Values{
		"name":         {"Test Widget"},
		"slug":         {"test-widget"},
		"price":        {"19.99"},
		"product_code": {"TWDG"},
		"tax_code":     {"txcd_10202000"},
		"seats":       {"1"},
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/products", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	AdminCreateProduct(conn, mock)(w, req)

	if !strings.Contains(w.Body.String(), "Failed to register product") {
		t.Errorf("expected provider error message, got: %s", w.Body.String())
	}

	// Confirm nothing was saved locally when the provider call fails.
	_, err := db.GetProductBySlug(conn, "test-widget")
	if err == nil {
		t.Error("expected product NOT to be saved when provider registration fails")
	}
}

func TestAdminCreateProductWithMultipleSeats(t *testing.T) {
	conn := newTestDBWithSchema(t)
	mock := mockprovider.New()

	form := url.Values{
		"name":         {"Team License"},
		"slug":         {"team-license"},
		"price":        {"49.99"},
		"product_code": {"TEAM"},
		"tax_code":     {"txcd_10202000"},
		"seats":        {"5"},
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/products", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	AdminCreateProduct(conn, mock)(w, req)

	if !strings.Contains(w.Body.String(), "Created \"Team License\" successfully") {
		t.Fatalf("expected success message, got: %s", w.Body.String())
	}

	saved, err := db.GetProductBySlug(conn, "team-license")
	if err != nil {
		t.Fatalf("expected product to be saved: %v", err)
	}
	if saved.Seats != 5 {
		t.Errorf("expected saved product to have 5 seats, got %d", saved.Seats)
	}
}

func TestAdminCreateProductRejectsInvalidSeats(t *testing.T) {
	conn := newTestDBWithSchema(t)
	mock := mockprovider.New()

	form := url.Values{
		"name":         {"Bad Seats"},
		"slug":         {"bad-seats"},
		"price":        {"9.99"},
		"product_code": {"BADS"},
		"tax_code":     {"txcd_10202000"},
		"seats":        {"0"},
	}

	req := httptest.NewRequest(http.MethodPost, "/admin/products", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	AdminCreateProduct(conn, mock)(w, req)

	if !strings.Contains(w.Body.String(), "Seats must be") {
		t.Errorf("expected seats validation error, got: %s", w.Body.String())
	}
	if len(mock.RegisterItemCalls) != 0 {
		t.Error("expected provider NOT to be called when seats is invalid")
	}
}
