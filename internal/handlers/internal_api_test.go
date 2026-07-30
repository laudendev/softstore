package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
