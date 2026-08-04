package db

import (
	"database/sql"

	"softstore/internal/models"
)

// GetOrCreateCart returns the cart for the given session token, creating
// one if it doesn't exist yet. Token is expected to be a random opaque
// string set by cookie middleware.
func GetOrCreateCart(conn *sql.DB, token string) (*models.Cart, error) {
	var c models.Cart
	err := conn.QueryRow(
		`SELECT id, token, created_at FROM carts WHERE token = ?`, token,
	).Scan(&c.ID, &c.Token, &c.CreatedAt)

	if err == nil {
		return &c, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	res, err := conn.Exec(`INSERT INTO carts (token) VALUES (?)`, token)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &models.Cart{ID: id, Token: token}, nil
}

// AddCartItem adds quantity of a product to a cart, or increments the
// existing row's quantity if the product is already in the cart.
func AddCartItem(conn *sql.DB, cartID, productID, quantity int64) error {
	_, err := conn.Exec(
		`INSERT INTO cart_items (cart_id, product_id, quantity)
		 VALUES (?, ?, ?)
		 ON CONFLICT(cart_id, product_id) DO UPDATE SET quantity = quantity + excluded.quantity`,
		cartID, productID, quantity,
	)
	return err
}

// GetCartWithItems loads a cart and its items, each joined with product
// display fields.
func GetCartWithItems(conn *sql.DB, token string) (*models.Cart, error) {
	cart, err := GetOrCreateCart(conn, token)
	if err != nil {
		return nil, err
	}

	rows, err := conn.Query(
		`SELECT ci.id, ci.cart_id, ci.product_id, ci.quantity, ci.created_at,
		        p.id, p.name, p.slug, p.description, p.price_cents,
		        p.stripe_price_id, p.product_code, p.stub_url, p.tax_code, p.created_at
		 FROM cart_items ci
		 JOIN products p ON p.id = ci.product_id
		 WHERE ci.cart_id = ?
		 ORDER BY ci.created_at ASC, ci.id ASC`,
		cart.ID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item models.CartItem
		if err := rows.Scan(
			&item.ID, &item.CartID, &item.ProductID, &item.Quantity, &item.CreatedAt,
			&item.Product.ID, &item.Product.Name, &item.Product.Slug, &item.Product.Description,
			&item.Product.PriceCents, &item.Product.StripePriceID, &item.Product.ProductCode,
			&item.Product.StubURL, &item.Product.TaxCode, &item.Product.CreatedAt,
		); err != nil {
			return nil, err
		}
		cart.Items = append(cart.Items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return cart, nil
}

// RemoveCartItem deletes a single product from a cart.
func RemoveCartItem(conn *sql.DB, cartID, productID int64) error {
	_, err := conn.Exec(
		`DELETE FROM cart_items WHERE cart_id = ? AND product_id = ?`,
		cartID, productID,
	)
	return err
}

// ClearCart deletes all items from a cart, identified by its session
// token. Used after a successful purchase to empty the cart so a
// completed order doesn't linger and reappear on the next visit.
func ClearCart(conn *sql.DB, token string) error {
	_, err := conn.Exec(
		`DELETE FROM cart_items WHERE cart_id = (SELECT id FROM carts WHERE token = ?)`,
		token,
	)
	return err
}
