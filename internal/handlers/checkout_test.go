package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"softstore/internal/db"
	"softstore/internal/models"
	"softstore/internal/payments"
	"softstore/internal/payments/mockprovider"
)

func TestCheckoutSuccess(t *testing.T) {
	conn := newTestDBWithSchema(t)
	mock := mockprovider.New()

	p := &models.Product{
		Name: "Test Widget", Slug: "test-widget", PriceCents: 1999,
		StripePriceID: "price_abc123", ProductCode: "TWDG", TaxCode: "txcd_10202000",
	}
	if err := db.CreateProduct(conn, p); err != nil {
		t.Fatalf("failed to seed product: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/checkout/test-widget", nil)
	req.SetPathValue("slug", "test-widget")
	w := httptest.NewRecorder()

	Checkout(conn, mock, "https://store.example.com")(w, req)

	if w.Code != http.StatusSeeOther {
		t.Errorf("expected redirect status %d, got %d", http.StatusSeeOther, w.Code)
	}
	if loc := w.Header().Get("Location"); loc != "https://mock-provider.test/checkout/session" {
		t.Errorf("expected redirect to mock checkout URL, got %q", loc)
	}

	if len(mock.StartPurchaseCalls) != 1 {
		t.Fatalf("expected 1 StartPurchase call, got %d", len(mock.StartPurchaseCalls))
	}
	call := mock.StartPurchaseCalls[0]
	if call.ProviderItemID != "price_abc123" {
		t.Errorf("expected provider item id 'price_abc123', got %q", call.ProviderItemID)
	}
	if call.Metadata["product"] != "TWDG" {
		t.Errorf("expected metadata product 'TWDG', got %q", call.Metadata["product"])
	}
	if call.Metadata["seats"] != "1" {
		t.Errorf("expected metadata seats '1', got %q", call.Metadata["seats"])
	}
	if call.SuccessURL != "https://store.example.com/thank-you" {
		t.Errorf("expected success URL to use provided base URL, got %q", call.SuccessURL)
	}
}

func TestCheckoutProductNotFound(t *testing.T) {
	conn := newTestDBWithSchema(t)
	mock := mockprovider.New()

	req := httptest.NewRequest(http.MethodPost, "/checkout/does-not-exist", nil)
	req.SetPathValue("slug", "does-not-exist")
	w := httptest.NewRecorder()

	Checkout(conn, mock, "https://store.example.com")(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	if len(mock.StartPurchaseCalls) != 0 {
		t.Error("expected provider NOT to be called for a nonexistent product")
	}
}

func TestCheckoutProviderError(t *testing.T) {
	conn := newTestDBWithSchema(t)
	mock := mockprovider.New()
	mock.StartPurchaseFunc = func(payments.PurchaseRequest) (payments.Purchase, error) {
		return payments.Purchase{}, errors.New("provider is down")
	}

	p := &models.Product{
		Name: "Test Widget", Slug: "test-widget", PriceCents: 1999,
		StripePriceID: "price_abc123", ProductCode: "TWDG", TaxCode: "txcd_10202000",
	}
	if err := db.CreateProduct(conn, p); err != nil {
		t.Fatalf("failed to seed product: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/checkout/test-widget", nil)
	req.SetPathValue("slug", "test-widget")
	w := httptest.NewRecorder()

	Checkout(conn, mock, "https://store.example.com")(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
