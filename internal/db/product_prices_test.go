package db

import (
	"database/sql"
	"errors"
	"testing"

	"softstore/internal/models"
)

func TestCreateAndGetProductPrice(t *testing.T) {
	conn := newTestDB(t)

	p := &models.Product{
		Name: "Team License", Slug: "team-license", PriceCents: 4999,
		StripePriceID: "price_base", ProductCode: "TEAM", TaxCode: "txcd_10202000",
	}
	if err := CreateProduct(conn, p); err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}

	created, err := CreateProductPrice(conn, p.ID, 3, "price_3seat")
	if err != nil {
		t.Fatalf("CreateProductPrice failed: %v", err)
	}
	if created.ID == 0 {
		t.Error("expected ID to be populated")
	}

	found, err := GetProductPrice(conn, p.ID, 3)
	if err != nil {
		t.Fatalf("GetProductPrice failed: %v", err)
	}
	if found.StripePriceID != "price_3seat" {
		t.Errorf("expected 'price_3seat', got %q", found.StripePriceID)
	}
}

func TestGetProductPriceNotFound(t *testing.T) {
	conn := newTestDB(t)

	p := &models.Product{
		Name: "Team License", Slug: "team-license-2", PriceCents: 4999,
		StripePriceID: "price_base2", ProductCode: "TEA2", TaxCode: "txcd_10202000",
	}
	if err := CreateProduct(conn, p); err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}

	_, err := GetProductPrice(conn, p.ID, 7)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
}

func TestCreateProductPriceDuplicateRejected(t *testing.T) {
	conn := newTestDB(t)

	p := &models.Product{
		Name: "Team License", Slug: "team-license-3", PriceCents: 4999,
		StripePriceID: "price_base3", ProductCode: "TEA3", TaxCode: "txcd_10202000",
	}
	if err := CreateProduct(conn, p); err != nil {
		t.Fatalf("CreateProduct failed: %v", err)
	}

	if _, err := CreateProductPrice(conn, p.ID, 2, "price_first"); err != nil {
		t.Fatalf("first CreateProductPrice failed: %v", err)
	}
	if _, err := CreateProductPrice(conn, p.ID, 2, "price_second"); err == nil {
		t.Error("expected duplicate product_id+seats to be rejected")
	}
}
