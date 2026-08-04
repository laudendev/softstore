package handlers

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"softstore/internal/db"
	"softstore/internal/models"
	"softstore/web"
)

func newTestCartTmpl(t *testing.T) *template.Template {
	t.Helper()
	tmpl, err := template.ParseFS(web.Templates, "templates/cart_drawer.html")
	if err != nil {
		t.Fatalf("failed to parse cart_drawer.html: %v", err)
	}
	return tmpl
}

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

	AddToCart(conn, newTestCartTmpl(t))(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "softstore_cart" {
		t.Fatalf("expected softstore_cart cookie to be set, got %v", cookies)
	}

	body := w.Body.String()
	if !strings.Contains(body, `id="cart-count" hx-swap-oob="true">1<`) {
		t.Errorf("expected cart-count oob swap showing 1, got %q", body)
	}
	if !strings.Contains(body, "Test Widget") {
		t.Errorf("expected drawer content to include product name, got %q", body)
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
	AddToCart(conn, newTestCartTmpl(t))(w1, req1)
	cartCookie := w1.Result().Cookies()[0]

	// Second add — reuse the cookie, expect count to increment to 2.
	req2 := httptest.NewRequest(http.MethodPost, "/cart/add/test-widget", nil)
	req2.SetPathValue("slug", "test-widget")
	req2.AddCookie(cartCookie)
	w2 := httptest.NewRecorder()
	AddToCart(conn, newTestCartTmpl(t))(w2, req2)

	body := w2.Body.String()
	if !strings.Contains(body, `id="cart-count" hx-swap-oob="true">2<`) {
		t.Errorf("expected cart-count oob swap showing 2, got %q", body)
	}
}

func TestAddToCartProductNotFound(t *testing.T) {
	conn := newTestDBWithSchema(t)

	req := httptest.NewRequest(http.MethodPost, "/cart/add/does-not-exist", nil)
	req.SetPathValue("slug", "does-not-exist")
	w := httptest.NewRecorder()

	AddToCart(conn, newTestCartTmpl(t))(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestRemoveFromCartUpdatesCartCountBadge(t *testing.T) {
	conn := newTestDBWithSchema(t)

	p := &models.Product{
		Name: "Test Widget", Slug: "test-widget", PriceCents: 1999,
		StripePriceID: "price_abc123", ProductCode: "TWDG", TaxCode: "txcd_10202000",
	}
	if err := db.CreateProduct(conn, p); err != nil {
		t.Fatalf("failed to seed product: %v", err)
	}

	// Add the item first, keep its cookie.
	addReq := httptest.NewRequest(http.MethodPost, "/cart/add/test-widget", nil)
	addReq.SetPathValue("slug", "test-widget")
	addW := httptest.NewRecorder()
	AddToCart(conn, newTestCartTmpl(t))(addW, addReq)
	cartCookie := addW.Result().Cookies()[0]

	// Now remove it, using the same cart.
	removeReq := httptest.NewRequest(http.MethodPost, "/cart/remove/test-widget", nil)
	removeReq.SetPathValue("slug", "test-widget")
	removeReq.AddCookie(cartCookie)
	removeW := httptest.NewRecorder()
	RemoveFromCart(conn, newTestCartTmpl(t))(removeW, removeReq)

	body := removeW.Body.String()
	if !strings.Contains(body, `id="cart-count" hx-swap-oob="true">0<`) {
		t.Errorf("expected cart-count oob swap showing 0 after removal, got %q", body)
	}
}
