package main

import (
	"fmt"
	"log"
	"net/http"

	"softstore/internal/db"
	"softstore/internal/models"
)

func main() {
	database, err := db.Open("softstore.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	// Temporary: seed one product to prove the DB layer works.
	seed := &models.Product{
		Name:        "Test Widget CLI",
		Slug:        "test-widget-cli",
		Description: "A sample product for testing.",
		PriceCents:  1999,
		FilePath:    "files/test-widget-cli.zip",
	}
	if err := db.CreateProduct(database, seed); err != nil {
		log.Println("seed insert (may already exist):", err)
	}

	products, err := db.ListProducts(database)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("products in db: %d", len(products))
	for _, p := range products {
		log.Printf("  - %s (%s) $%.2f", p.Name, p.Slug, float64(p.PriceCents)/100)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
