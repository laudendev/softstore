package db

import (
	"database/sql"

	"softstore/internal/models"
)

func CreateProduct(conn *sql.DB, p *models.Product) error {
	res, err := conn.Exec(
		`INSERT INTO products (name, slug, description, price_cents, stripe_price_id, product_code, stub_url, tax_code)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Name, p.Slug, p.Description, p.PriceCents, p.StripePriceID, p.ProductCode, p.StubURL, p.TaxCode,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = id
	return nil
}

func ListProducts(conn *sql.DB) ([]models.Product, error) {
	rows, err := conn.Query(
		`SELECT id, name, slug, description, price_cents, stripe_price_id, product_code, stub_url, tax_code, created_at
		 FROM products ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.PriceCents, &p.StripePriceID, &p.ProductCode, &p.StubURL, &p.TaxCode, &p.CreatedAt); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}
