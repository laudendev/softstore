package db

import (
	"database/sql"

	"softstore/internal/models"
)

// GetProductPrice looks up the Stripe Price ID for a specific
// product+seats combination, if one has already been created.
func GetProductPrice(conn *sql.DB, productID, seats int64) (*models.ProductPrice, error) {
	var pp models.ProductPrice
	err := conn.QueryRow(
		`SELECT id, product_id, seats, stripe_price_id, created_at
		 FROM product_prices WHERE product_id = ? AND seats = ?`,
		productID, seats,
	).Scan(&pp.ID, &pp.ProductID, &pp.Seats, &pp.StripePriceID, &pp.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &pp, nil
}

// CreateProductPrice records a newly-created Stripe Price for a
// product+seats combination.
func CreateProductPrice(conn *sql.DB, productID, seats int64, stripePriceID string) (*models.ProductPrice, error) {
	res, err := conn.Exec(
		`INSERT INTO product_prices (product_id, seats, stripe_price_id) VALUES (?, ?, ?)`,
		productID, seats, stripePriceID,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &models.ProductPrice{ID: id, ProductID: productID, Seats: seats, StripePriceID: stripePriceID}, nil
}
