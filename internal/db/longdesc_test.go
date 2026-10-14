package db

import (
	"path/filepath"
	"testing"

	"softstore/internal/models"
)

func TestLongDescriptionMigrationAndList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "t.db")
	conn, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	p := &models.Product{Name: "N", Slug: "n", Description: "short", PriceCents: 1600,
		StripePriceID: "price_x", ProductCode: "POPF", TaxCode: "txcd_10202000", Seats: 1}
	if err := CreateProduct(conn, p); err != nil {
		t.Fatal(err)
	}
	list, err := ListProducts(conn)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v, %d rows", err, len(list))
	}
	if list[0].LongDescription != "" || list[0].Description != "short" {
		t.Fatalf("unexpected defaults: %+v", list[0])
	}
	if _, err := conn.Exec("UPDATE products SET long_description = ? WHERE slug = ?", "line one\nline two", "n"); err != nil {
		t.Fatal(err)
	}
	conn.Close()

	conn, err = Open(path) // migration must be idempotent
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	list, err = ListProducts(conn)
	if err != nil || len(list) != 1 {
		t.Fatalf("relist: %v, %d rows", err, len(list))
	}
	if list[0].LongDescription != "line one\nline two" || list[0].Description != "short" {
		t.Fatalf("unexpected row after reopen: %+v", list[0])
	}
}
