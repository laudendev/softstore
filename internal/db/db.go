package db

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		return nil, err
	}
	if err := migrate(conn); err != nil {
		return nil, err
	}
	return conn, nil
}

func migrate(conn *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		slug TEXT NOT NULL UNIQUE,
		description TEXT NOT NULL DEFAULT '',
		price_cents INTEGER NOT NULL,
		stripe_price_id TEXT NOT NULL,
		stripe_product_id TEXT NOT NULL DEFAULT '',
		product_code TEXT NOT NULL,
        stub_url TEXT NOT NULL DEFAULT '',
		tax_code TEXT NOT NULL,
		seats INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS carts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		token TEXT NOT NULL UNIQUE,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS cart_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		cart_id INTEGER NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
		product_id INTEGER NOT NULL REFERENCES products(id),
		quantity INTEGER NOT NULL DEFAULT 1,
		seats INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(cart_id, product_id, seats)
	);

	CREATE TABLE IF NOT EXISTS product_prices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		product_id INTEGER NOT NULL REFERENCES products(id),
		seats INTEGER NOT NULL,
		stripe_price_id TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(product_id, seats)
	);


	CREATE TABLE IF NOT EXISTS subscribers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		status TEXT NOT NULL DEFAULT 'pending',
		confirm_token TEXT NOT NULL,
		unsubscribe_token TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		confirmed_at DATETIME
	);`
	_, err := conn.Exec(schema)
	if err != nil {
		return err
	}
	if err := addSubscriberSourceColumn(conn); err != nil {
		return err
	}
	return addProductLongDescriptionColumn(conn)
}

// addSubscriberSourceColumn adds subscribers.source on databases created
// before the column existed. Existing rows default to 'shop'. Safe to run on
// every start: it does nothing when the column is already present.
func addSubscriberSourceColumn(conn *sql.DB) error {
	var n int
	if err := conn.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info('subscribers') WHERE name = 'source'`,
	).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err := conn.Exec(`ALTER TABLE subscribers ADD COLUMN source TEXT NOT NULL DEFAULT 'shop'`)
	return err
}

// addProductLongDescriptionColumn adds the description shown only on the shop
// page. The short description stays separate because checkout builds the
// Stripe line-item text from it. Safe to run on every start: it does nothing
// when the column is already present.
func addProductLongDescriptionColumn(conn *sql.DB) error {
	var n int
	if err := conn.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info('products') WHERE name = 'long_description'`,
	).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	_, err := conn.Exec(`ALTER TABLE products ADD COLUMN long_description TEXT NOT NULL DEFAULT ''`)
	return err
}
