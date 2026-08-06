package db

import (
	"database/sql"
	"testing"

	"softstore/internal/models"
)

func seedProduct(t *testing.T, conn *sql.DB, slug string, priceCents int64) *models.Product {
	t.Helper()
	p := &models.Product{
		Name: slug, Slug: slug, PriceCents: priceCents,
		StripePriceID: "price_" + slug, ProductCode: slug, TaxCode: "txcd_10202000",
	}
	if err := CreateProduct(conn, p); err != nil {
		t.Fatalf("seed product %s: %v", slug, err)
	}
	return p
}

func TestGetOrCreateCartCreatesNew(t *testing.T) {
	conn := newTestDB(t)

	cart, err := GetOrCreateCart(conn, "token-abc")
	if err != nil {
		t.Fatalf("GetOrCreateCart failed: %v", err)
	}
	if cart.ID == 0 {
		t.Error("expected cart ID to be populated, got 0")
	}
	if cart.Token != "token-abc" {
		t.Errorf("expected token 'token-abc', got %q", cart.Token)
	}
}

func TestGetOrCreateCartReturnsExisting(t *testing.T) {
	conn := newTestDB(t)

	first, err := GetOrCreateCart(conn, "token-xyz")
	if err != nil {
		t.Fatalf("first GetOrCreateCart failed: %v", err)
	}

	second, err := GetOrCreateCart(conn, "token-xyz")
	if err != nil {
		t.Fatalf("second GetOrCreateCart failed: %v", err)
	}

	if second.ID != first.ID {
		t.Errorf("expected same cart ID for same token, got %d and %d", first.ID, second.ID)
	}
}

func TestAddCartItemIncrementsOnDuplicate(t *testing.T) {
	conn := newTestDB(t)
	p := seedProduct(t, conn, "widget", 1000)
	cart, _ := GetOrCreateCart(conn, "token-1")

	if err := AddCartItem(conn, cart.ID, p.ID, 1, 1); err != nil {
		t.Fatalf("first AddCartItem failed: %v", err)
	}
	if err := AddCartItem(conn, cart.ID, p.ID, 1, 1); err != nil {
		t.Fatalf("second AddCartItem failed: %v", err)
	}

	got, err := GetCartWithItems(conn, "token-1")
	if err != nil {
		t.Fatalf("GetCartWithItems failed: %v", err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("expected 1 distinct item, got %d", len(got.Items))
	}
	if got.Items[0].Quantity != 2 {
		t.Errorf("expected quantity 2 after two adds, got %d", got.Items[0].Quantity)
	}
}

func TestGetCartWithItemsJoinsProductData(t *testing.T) {
	conn := newTestDB(t)
	p := seedProduct(t, conn, "gadget", 2500)
	cart, _ := GetOrCreateCart(conn, "token-2")

	if err := AddCartItem(conn, cart.ID, p.ID, 1, 3); err != nil {
		t.Fatalf("AddCartItem failed: %v", err)
	}

	got, err := GetCartWithItems(conn, "token-2")
	if err != nil {
		t.Fatalf("GetCartWithItems failed: %v", err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got.Items))
	}
	item := got.Items[0]
	if item.Product.Name != "gadget" {
		t.Errorf("expected joined product name 'gadget', got %q", item.Product.Name)
	}
	if item.Quantity != 3 {
		t.Errorf("expected quantity 3, got %d", item.Quantity)
	}
	if got.TotalCents() != 7500 {
		t.Errorf("expected total 7500 cents, got %d", got.TotalCents())
	}
	if got.ItemCount() != 3 {
		t.Errorf("expected item count 3, got %d", got.ItemCount())
	}
}

func TestRemoveCartItem(t *testing.T) {
	conn := newTestDB(t)
	p := seedProduct(t, conn, "removable", 500)
	cart, _ := GetOrCreateCart(conn, "token-3")

	if err := AddCartItem(conn, cart.ID, p.ID, 1, 1); err != nil {
		t.Fatalf("AddCartItem failed: %v", err)
	}
	if err := RemoveCartItem(conn, cart.ID, p.ID, 1); err != nil {
		t.Fatalf("RemoveCartItem failed: %v", err)
	}

	got, err := GetCartWithItems(conn, "token-3")
	if err != nil {
		t.Fatalf("GetCartWithItems failed: %v", err)
	}
	if len(got.Items) != 0 {
		t.Errorf("expected 0 items after removal, got %d", len(got.Items))
	}
}

func TestClearCart(t *testing.T) {
	conn := newTestDB(t)
	p1 := seedProduct(t, conn, "item-one", 500)
	p2 := seedProduct(t, conn, "item-two", 700)
	cart, _ := GetOrCreateCart(conn, "token-clear")

	if err := AddCartItem(conn, cart.ID, p1.ID, 1, 1); err != nil {
		t.Fatalf("add p1: %v", err)
	}
	if err := AddCartItem(conn, cart.ID, p2.ID, 1, 2); err != nil {
		t.Fatalf("add p2: %v", err)
	}

	if err := ClearCart(conn, "token-clear"); err != nil {
		t.Fatalf("ClearCart failed: %v", err)
	}

	got, err := GetCartWithItems(conn, "token-clear")
	if err != nil {
		t.Fatalf("GetCartWithItems failed: %v", err)
	}
	if len(got.Items) != 0 {
		t.Errorf("expected 0 items after clear, got %d", len(got.Items))
	}
}

func TestClearCartNonexistentTokenNoError(t *testing.T) {
	conn := newTestDB(t)

	if err := ClearCart(conn, "no-such-token"); err != nil {
		t.Errorf("expected no error clearing a nonexistent cart, got: %v", err)
	}
}

func TestGetCartWithItemsIncludesStripeProductID(t *testing.T) {
	conn := newTestDB(t)
	p := &models.Product{
		Name: "widget", Slug: "widget-prod-id", PriceCents: 1000,
		StripePriceID: "price_x", StripeProductID: "prod_x_real",
		ProductCode: "WGT1", TaxCode: "txcd_10202000",
	}
	if err := CreateProduct(conn, p); err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}
	cart, _ := GetOrCreateCart(conn, "token-prod-id")
	if err := AddCartItem(conn, cart.ID, p.ID, 1, 1); err != nil {
		t.Fatalf("AddCartItem failed: %v", err)
	}

	got, err := GetCartWithItems(conn, "token-prod-id")
	if err != nil {
		t.Fatalf("GetCartWithItems failed: %v", err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(got.Items))
	}
	if got.Items[0].Product.StripeProductID != "prod_x_real" {
		t.Errorf("expected StripeProductID 'prod_x_real', got %q", got.Items[0].Product.StripeProductID)
	}
}

func TestRemoveCartItemOnlyAffectsMatchingSeatTier(t *testing.T) {
	conn := newTestDB(t)
	p := seedProduct(t, conn, "multi-tier-widget", 1000)
	cart, _ := GetOrCreateCart(conn, "token-multi-tier")

	if err := AddCartItem(conn, cart.ID, p.ID, 1, 1); err != nil {
		t.Fatalf("add 1-seat item: %v", err)
	}
	if err := AddCartItem(conn, cart.ID, p.ID, 2, 1); err != nil {
		t.Fatalf("add 2-seat item: %v", err)
	}

	if err := RemoveCartItem(conn, cart.ID, p.ID, 2); err != nil {
		t.Fatalf("RemoveCartItem failed: %v", err)
	}

	got, err := GetCartWithItems(conn, "token-multi-tier")
	if err != nil {
		t.Fatalf("GetCartWithItems failed: %v", err)
	}
	if len(got.Items) != 1 {
		t.Fatalf("expected 1 remaining item (the 1-seat tier), got %d", len(got.Items))
	}
	if got.Items[0].Seats != 1 {
		t.Errorf("expected the remaining item to be the 1-seat tier, got seats=%d", got.Items[0].Seats)
	}
}
