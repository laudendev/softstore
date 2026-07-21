# softstore

A storefront for selling downloadable software and digital books, built in Go with HTMX. Handles product catalog, admin management, and Stripe Checkout. License issuance and delivery are handled by a separate service ([Quartermaster](https://github.com/laudendev/quartermaster)) via Stripe webhooks.

## Stack

- Go 1.26+
- HTMX + server-rendered HTML templates
- SQLite (via [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite), pure-Go, no CGO)
- Stripe (Checkout, Products, Prices, Tax)

## Architecture

softstore owns the product catalog and checkout initiation only. It creates Stripe Products/Prices via API when an admin adds a product, and redirects customers to Stripe Checkout with the metadata Quartermaster's webhook expects (`product` code, `seats` count). softstore never generates, stores, or delivers license keys — that's handled entirely by Quartermaster after Stripe's webhook fires.

## Local development

Requires Go 1.26+, and a Stripe test-mode account.

```bash
# required environment variables (see internal/config)
export STRIPE_SECRET_KEY='sk_test_...'
export SESSION_SECRET='<base64, 32 random bytes>'
export ADMIN_USERNAME='...'
export ADMIN_PASSWORD_HASH='<bcrypt hash>'

go run ./cmd/server
```

The server listens on `:8443` via HTTPS, using a local self-signed certificate (`localhost-cert.pem` / `localhost-key.pem`, generated separately, not committed).

## License

[PolyForm Noncommercial 1.0.0](LICENSE) — free for noncommercial use. Commercial use requires separate authorization.
