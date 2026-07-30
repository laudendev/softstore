package db

import (
	"database/sql"
	"errors"
	"testing"

	"softstore/internal/models"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := Open(":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory test db: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestCreateAndListProducts(t *testing.T) {
	conn := newTestDB(t)

	p := &models.Product{
		Name:          "Test Widget",
		Slug:          "test-widget",
		Description:   "A widget for testing.",
		PriceCents:    1999,
		StripePriceID: "price_test123",
		ProductCode:   "TWDG",
		StubURL:       "https://example.com/stub.zip",
		TaxCode:       "txcd_10202000",
	}

	if err := CreateProduct(conn, p); err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}
	if p.ID == 0 {
		t.Error("expected CreateProduct to populate ID, got 0")
	}

	products, err := ListProducts(conn)
	if err != nil {
		t.Fatalf("ListProducts failed: %v", err)
	}
	if len(products) != 1 {
		t.Fatalf("expected 1 product, got %d", len(products))
	}
	if products[0].Name != "Test Widget" {
		t.Errorf("expected name 'Test Widget', got %q", products[0].Name)
	}
	if products[0].PriceCents != 1999 {
		t.Errorf("expected price 1999, got %d", products[0].PriceCents)
	}
}

func TestCreateProductDuplicateSlug(t *testing.T) {
	conn := newTestDB(t)

	p1 := &models.Product{Name: "First", Slug: "same-slug", PriceCents: 100, StripePriceID: "price_1", ProductCode: "AAAA", TaxCode: "txcd_10202000"}
	p2 := &models.Product{Name: "Second", Slug: "same-slug", PriceCents: 200, StripePriceID: "price_2", ProductCode: "BBBB", TaxCode: "txcd_10202000"}

	if err := CreateProduct(conn, p1); err != nil {
		t.Fatalf("first CreateProduct failed: %v", err)
	}
	if err := CreateProduct(conn, p2); err == nil {
		t.Error("expected second CreateProduct with duplicate slug to fail, but it succeeded")
	}
}

func TestListProductsEmpty(t *testing.T) {
	conn := newTestDB(t)

	products, err := ListProducts(conn)
	if err != nil {
		t.Fatalf("ListProducts on empty db failed: %v", err)
	}
	if len(products) != 0 {
		t.Errorf("expected 0 products, got %d", len(products))
	}
}

func TestListProductsOrderedByCreatedDesc(t *testing.T) {
	conn := newTestDB(t)

	first := &models.Product{Name: "First", Slug: "first", PriceCents: 100, StripePriceID: "price_1", ProductCode: "AAAA", TaxCode: "txcd_10202000"}
	second := &models.Product{Name: "Second", Slug: "second", PriceCents: 200, StripePriceID: "price_2", ProductCode: "BBBB", TaxCode: "txcd_10202000"}

	if err := CreateProduct(conn, first); err != nil {
		t.Fatalf("create first failed: %v", err)
	}
	if err := CreateProduct(conn, second); err != nil {
		t.Fatalf("create second failed: %v", err)
	}

	products, err := ListProducts(conn)
	if err != nil {
		t.Fatalf("ListProducts failed: %v", err)
	}
	if len(products) != 2 {
		t.Fatalf("expected 2 products, got %d", len(products))
	}
	// Most recently created should come first.
	if products[0].Slug != "second" {
		t.Errorf("expected most recent product 'second' first, got %q", products[0].Slug)
	}
}

func TestGetProductBySlug(t *testing.T) {
	conn := newTestDB(t)

	p := &models.Product{Name: "Findable", Slug: "findable", PriceCents: 500, StripePriceID: "price_x", ProductCode: "FIND", TaxCode: "txcd_10202000"}
	if err := CreateProduct(conn, p); err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}

	found, err := GetProductBySlug(conn, "findable")
	if err != nil {
		t.Fatalf("GetProductBySlug failed: %v", err)
	}
	if found.Name != "Findable" {
		t.Errorf("expected name 'Findable', got %q", found.Name)
	}
}

func TestGetProductBySlugNotFound(t *testing.T) {
	conn := newTestDB(t)

	_, err := GetProductBySlug(conn, "does-not-exist")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows for missing slug, got %v", err)
	}
}

func TestGetProductByStripePriceID(t *testing.T) {
	conn := newTestDB(t)

	p := &models.Product{Name: "Priced", Slug: "priced", PriceCents: 700, StripePriceID: "price_lookup_me", ProductCode: "PRCD", TaxCode: "txcd_10202000"}
	if err := CreateProduct(conn, p); err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}

	found, err := GetProductByStripePriceID(conn, "price_lookup_me")
	if err != nil {
		t.Fatalf("GetProductByStripePriceID failed: %v", err)
	}
	if found.ProductCode != "PRCD" {
		t.Errorf("expected product code 'PRCD', got %q", found.ProductCode)
	}
}

func TestGetProductByStripePriceIDNotFound(t *testing.T) {
	conn := newTestDB(t)

	_, err := GetProductByStripePriceID(conn, "price_does_not_exist")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows for missing price id, got %v", err)
	}
}
