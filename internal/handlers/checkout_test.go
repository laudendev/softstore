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
	if len(call.LineItems) != 1 {
		t.Fatalf("expected 1 line item, got %d", len(call.LineItems))
	}
	if call.LineItems[0].ProviderItemID != "price_abc123" {
		t.Errorf("expected provider item id 'price_abc123', got %q", call.LineItems[0].ProviderItemID)
	}
	if call.LineItems[0].Quantity != 1 {
		t.Errorf("expected quantity 1, got %d", call.LineItems[0].Quantity)
	}
	if call.Metadata["product"] != "TWDG" {
		t.Errorf("expected metadata product 'TWDG', got %q", call.Metadata["product"])
	}
	if call.Metadata["seats"] != "1" {
		t.Errorf("expected metadata seats '1', got %q", call.Metadata["seats"])
	}
	if call.SuccessURL != "https://store.example.com/thank-you?session_id={CHECKOUT_SESSION_ID}" {
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

func TestCartCheckoutSuccess(t *testing.T) {
	conn := newTestDBWithSchema(t)
	mock := mockprovider.New()

	p1 := &models.Product{
		Name: "Widget A", Slug: "widget-a", PriceCents: 1000,
		StripePriceID: "price_a", ProductCode: "WGTA", TaxCode: "txcd_10202000",
	}
	p2 := &models.Product{
		Name: "Widget B", Slug: "widget-b", PriceCents: 2000,
		StripePriceID: "price_b", ProductCode: "WGTB", TaxCode: "txcd_10202000",
	}
	if err := db.CreateProduct(conn, p1); err != nil {
		t.Fatalf("seed p1: %v", err)
	}
	if err := db.CreateProduct(conn, p2); err != nil {
		t.Fatalf("seed p2: %v", err)
	}

	cart, err := db.GetOrCreateCart(conn, "test-token")
	if err != nil {
		t.Fatalf("get or create cart: %v", err)
	}
	if err := db.AddCartItem(conn, cart.ID, p1.ID, 1, 1); err != nil {
		t.Fatalf("add p1 to cart: %v", err)
	}
	if err := db.AddCartItem(conn, cart.ID, p2.ID, 1, 2); err != nil {
		t.Fatalf("add p2 to cart: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
	req.AddCookie(&http.Cookie{Name: "softstore_cart", Value: "test-token"})
	w := httptest.NewRecorder()

	CartCheckout(conn, mock, "https://store.example.com")(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect status %d, got %d", http.StatusSeeOther, w.Code)
	}

	if len(mock.StartPurchaseCalls) != 1 {
		t.Fatalf("expected 1 StartPurchase call, got %d", len(mock.StartPurchaseCalls))
	}
	call := mock.StartPurchaseCalls[0]
	if len(call.LineItems) != 2 {
		t.Fatalf("expected 2 line items, got %d", len(call.LineItems))
	}
	if call.Metadata["cart_token"] != "test-token" {
		t.Errorf("expected cart_token metadata 'test-token', got %q", call.Metadata["cart_token"])
	}
}

func TestCartCheckoutEmptyCart(t *testing.T) {
	conn := newTestDBWithSchema(t)
	mock := mockprovider.New()

	req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
	req.AddCookie(&http.Cookie{Name: "softstore_cart", Value: "empty-token"})
	w := httptest.NewRecorder()

	CartCheckout(conn, mock, "https://store.example.com")(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d for empty cart, got %d", http.StatusBadRequest, w.Code)
	}
	if len(mock.StartPurchaseCalls) != 0 {
		t.Errorf("expected no StartPurchase call for empty cart, got %d", len(mock.StartPurchaseCalls))
	}
}

func TestCartCheckoutUsesCorrectSeatTierPrice(t *testing.T) {
	conn := newTestDBWithSchema(t)
	mock := mockprovider.New()

	p := &models.Product{
		Name: "Team License", Slug: "team-license", PriceCents: 1000,
		StripePriceID: "price_base", StripeProductID: "prod_base",
		ProductCode: "TEAM", TaxCode: "txcd_10202000",
	}
	if err := db.CreateProduct(conn, p); err != nil {
		t.Fatalf("seed product: %v", err)
	}

	cart, err := db.GetOrCreateCart(conn, "seat-tier-token")
	if err != nil {
		t.Fatalf("get or create cart: %v", err)
	}
	if err := db.AddCartItem(conn, cart.ID, p.ID, 3, 1); err != nil {
		t.Fatalf("add 3-seat item to cart: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/checkout", nil)
	req.AddCookie(&http.Cookie{Name: "softstore_cart", Value: "seat-tier-token"})
	w := httptest.NewRecorder()

	CartCheckout(conn, mock, "https://store.example.com")(w, req)

	if w.Code != http.StatusSeeOther {
		t.Fatalf("expected redirect, got %d: %s", w.Code, w.Body.String())
	}

	if len(mock.StartPurchaseCalls) != 1 {
		t.Fatalf("expected 1 StartPurchase call, got %d", len(mock.StartPurchaseCalls))
	}
	lineItems := mock.StartPurchaseCalls[0].LineItems
	if len(lineItems) != 1 {
		t.Fatalf("expected 1 line item, got %d", len(lineItems))
	}
	if lineItems[0].ProviderItemID == "price_base" {
		t.Error("expected the 3-seat tier price, not the base 1-seat price")
	}

	if len(mock.AddPriceCalls) != 1 {
		t.Fatalf("expected 1 AddPrice call for the 3-seat tier, got %d", len(mock.AddPriceCalls))
	}
	if mock.AddPriceCalls[0].PriceCents != 3000 {
		t.Errorf("expected 3000 cents (1000 * 3), got %d", mock.AddPriceCalls[0].PriceCents)
	}
}
