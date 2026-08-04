package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"strings"

	"softstore/internal/db"
	"softstore/internal/models"
)

func TestGetProductByPriceSuccess(t *testing.T) {
	conn := newTestDBWithSchema(t)

	p := &models.Product{
		Name: "Test Widget", Slug: "test-widget", PriceCents: 1999,
		StripePriceID: "price_abc123", ProductCode: "TWDG", TaxCode: "txcd_10202000",
	}
	if err := db.CreateProduct(conn, p); err != nil {
		t.Fatalf("failed to seed product: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/internal/products/by-price/price_abc123", nil)
	req.SetPathValue("price_id", "price_abc123")
	w := httptest.NewRecorder()

	GetProductByPrice(conn)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp productByPriceResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ProductCode != "TWDG" {
		t.Errorf("expected product code 'TWDG', got %q", resp.ProductCode)
	}
}

func TestGetProductByPriceNotFound(t *testing.T) {
	conn := newTestDBWithSchema(t)

	req := httptest.NewRequest(http.MethodGet, "/internal/products/by-price/price_missing", nil)
	req.SetPathValue("price_id", "price_missing")
	w := httptest.NewRecorder()

	GetProductByPrice(conn)(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestClearCartSuccess(t *testing.T) {
	conn := newTestDBWithSchema(t)
	p := &models.Product{
		Name: "Test Widget", Slug: "test-widget", PriceCents: 1999,
		StripePriceID: "price_abc123", ProductCode: "TWDG", TaxCode: "txcd_10202000",
	}
	if err := db.CreateProduct(conn, p); err != nil {
		t.Fatalf("failed to seed product: %v", err)
	}

	cart, err := db.GetOrCreateCart(conn, "clear-me")
	if err != nil {
		t.Fatalf("get or create cart: %v", err)
	}
	if err := db.AddCartItem(conn, cart.ID, p.ID, 1); err != nil {
		t.Fatalf("add cart item: %v", err)
	}

	body := strings.NewReader(`{"cart_token":"clear-me"}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/cart/clear", body)
	w := httptest.NewRecorder()

	ClearCart(conn)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	got, err := db.GetCartWithItems(conn, "clear-me")
	if err != nil {
		t.Fatalf("get cart with items: %v", err)
	}
	if len(got.Items) != 0 {
		t.Errorf("expected 0 items after clear, got %d", len(got.Items))
	}
}

func TestClearCartMissingToken(t *testing.T) {
	conn := newTestDBWithSchema(t)

	body := strings.NewReader(`{}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/cart/clear", body)
	w := httptest.NewRecorder()

	ClearCart(conn)(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}
