package db

import (
	"database/sql"

	"softstore/internal/models"
)

func CreateProduct(conn *sql.DB, p *models.Product) error {
	seats := p.Seats
	if seats <= 0 {
		seats = 1
	}
	res, err := conn.Exec(
		`INSERT INTO products (name, slug, description, price_cents, stripe_price_id, stripe_product_id, product_code, stub_url, tax_code, seats)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Name, p.Slug, p.Description, p.PriceCents, p.StripePriceID, p.StripeProductID, p.ProductCode, p.StubURL, p.TaxCode, seats,
	)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	p.ID = id
	p.Seats = seats
	return nil
}

func GetProductBySlug(conn *sql.DB, slug string) (*models.Product, error) {
	var p models.Product
	err := conn.QueryRow(
		`SELECT id, name, slug, description, price_cents, stripe_price_id, stripe_product_id, product_code, stub_url, tax_code, seats, created_at
		 FROM products WHERE slug = ?`,
		slug,
	).Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.PriceCents, &p.StripePriceID, &p.StripeProductID, &p.ProductCode, &p.StubURL, &p.TaxCode, &p.Seats, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// GetProductByStripePriceID looks up a product by any of its Stripe
// Price IDs — either the product's own default (1-seat) price, or a
// price recorded in product_prices for a different seat tier. When the
// match comes from product_prices, the returned Product's Seats field
// is overridden to reflect that specific tier's seat count, since the
// caller needs to know how many seats THIS price actually represents,
// not the product's base/default seat count.
func GetProductByStripePriceID(conn *sql.DB, priceID string) (*models.Product, error) {
	var p models.Product
	err := conn.QueryRow(
		`SELECT id, name, slug, description, price_cents, stripe_price_id, stripe_product_id, product_code, stub_url, tax_code, seats, created_at
		 FROM products WHERE stripe_price_id = ?`,
		priceID,
	).Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.PriceCents, &p.StripePriceID, &p.StripeProductID, &p.ProductCode, &p.StubURL, &p.TaxCode, &p.Seats, &p.CreatedAt)
	if err == nil {
		return &p, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Not the product's default price — check whether it's a recorded
	// seat-tier price instead.
	var tierSeats int64
	err = conn.QueryRow(
		`SELECT p.id, p.name, p.slug, p.description, p.price_cents, p.stripe_price_id, p.stripe_product_id, p.product_code, p.stub_url, p.tax_code, pp.seats, p.created_at
		 FROM product_prices pp
		 JOIN products p ON p.id = pp.product_id
		 WHERE pp.stripe_price_id = ?`,
		priceID,
	).Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.PriceCents, &p.StripePriceID, &p.StripeProductID, &p.ProductCode, &p.StubURL, &p.TaxCode, &tierSeats, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	p.Seats = tierSeats
	return &p, nil
}

func ListProducts(conn *sql.DB) ([]models.Product, error) {
	rows, err := conn.Query(
		`SELECT id, name, slug, description, price_cents, stripe_price_id, stripe_product_id, product_code, stub_url, tax_code, seats, created_at
		 FROM products ORDER BY created_at DESC, id DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var p models.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Slug, &p.Description, &p.PriceCents, &p.StripePriceID, &p.StripeProductID, &p.ProductCode, &p.StubURL, &p.TaxCode, &p.Seats, &p.CreatedAt); err != nil {
			return nil, err
		}
		products = append(products, p)
	}
	return products, rows.Err()
}

