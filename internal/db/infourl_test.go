package db

import (
	"path/filepath"
	"testing"

	"softstore/internal/models"
)

func TestInfoURLMigrationAndList(t *testing.T) {
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
	if err != nil || len(list) != 1 || list[0].InfoURL != "" {
		t.Fatalf("defaults: %v %+v", err, list)
	}
	if _, err := conn.Exec("UPDATE products SET info_url = ? WHERE slug = ?", "https://example.com/", "n"); err != nil {
		t.Fatal(err)
	}
	conn.Close()

	conn, err = Open(path) // migration must be idempotent
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	list, err = ListProducts(conn)
	if err != nil || len(list) != 1 || list[0].InfoURL != "https://example.com/" {
		t.Fatalf("after reopen: %v %+v", err, list)
	}
}
