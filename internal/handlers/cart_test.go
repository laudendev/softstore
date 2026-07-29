package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"softstore/internal/db"
	"softstore/internal/models"
)

func TestAddToCartCreatesCartAndCookie(t *testing.T) {
	conn := newTestDBWithSchema(t)

	p := &models.Product{
		Name: "Test Widget", Slug: "test-widget", PriceCents: 1999,
		StripePriceID: "price_abc123", ProductCode: "TWDG", TaxCode: "txcd_10202000",
	}
	if err := db.CreateProduct(conn, p); err != nil {
		t.Fatalf("failed to seed product: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/cart/add/test-widget", nil)
	req.SetPathValue("slug", "test-widget")
	w := httptest.NewRecorder()

	AddToCart(conn)(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "softstore_cart" {
		t.Fatalf("expected softstore_cart cookie to be set, got %v", cookies)
	}

	body := w.Body.String()
	if body != `<span id="cart-count">1</span>` {
		t.Errorf("expected cart-count fragment showing 1, got %q", body)
	}
}

func TestAddToCartIncrementsExistingCart(t *testing.T) {
	conn := newTestDBWithSchema(t)

	p := &models.Product{
		Name: "Test Widget", Slug: "test-widget", PriceCents: 1999,
		StripePriceID: "price_abc123", ProductCode: "TWDG", TaxCode: "txcd_10202000",
	}
	if err := db.CreateProduct(conn, p); err != nil {
		t.Fatalf("failed to seed product: %v", err)
	}

	// First add — no cookie yet, one gets set.
	req1 := httptest.NewRequest(http.MethodPost, "/cart/add/test-widget", nil)
	req1.SetPathValue("slug", "test-widget")
	w1 := httptest.NewRecorder()
	AddToCart(conn)(w1, req1)
	cartCookie := w1.Result().Cookies()[0]

	// Second add — reuse the cookie, expect count to increment to 2.
	req2 := httptest.NewRequest(http.MethodPost, "/cart/add/test-widget", nil)
	req2.SetPathValue("slug", "test-widget")
	req2.AddCookie(cartCookie)
	w2 := httptest.NewRecorder()
	AddToCart(conn)(w2, req2)

	body := w2.Body.String()
	if body != `<span id="cart-count">2</span>` {
		t.Errorf("expected cart-count fragment showing 2, got %q", body)
	}
}

func TestAddToCartProductNotFound(t *testing.T) {
	conn := newTestDBWithSchema(t)

	req := httptest.NewRequest(http.MethodPost, "/cart/add/does-not-exist", nil)
	req.SetPathValue("slug", "does-not-exist")
	w := httptest.NewRecorder()

	AddToCart(conn)(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
