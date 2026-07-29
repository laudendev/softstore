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
		product_code TEXT NOT NULL,
        stub_url TEXT NOT NULL DEFAULT '',
		tax_code TEXT NOT NULL,
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
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(cart_id, product_id)
	);`
	_, err := conn.Exec(schema)
	return err
}
