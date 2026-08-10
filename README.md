# softstore

A storefront for selling downloadable software and digital books, built in Go with HTMX. Handles product catalog, cart, admin management, and Stripe Checkout. License issuance and delivery are handled by a separate service ([Quartermaster](https://github.com/laudendev/quartermaster)) via Stripe webhooks.

## Features

- Product catalog with per-product Stripe-backed pricing and tax categories
- Multi-device seat tiers (1–6 devices) with automatic volume discounts, priced and cached on demand
- Persistent HTMX cart (cookie-based, no login required) supporting mixed products and seat tiers in one checkout
- Single-item "buy now" and full-cart checkout, both via Stripe Checkout with automatic tax and required billing address
- Live thank-you page that polls for fulfillment and reveals the receipt (including license keys) once Quartermaster completes it
- Minimal cookie-session admin area for adding products, guarded separately from customer/cart cookies
- Legal pages (Terms, Privacy, EULA, Refunds, Cookies) served from the same layout
- Light/dark theme support (`prefers-color-scheme`)

## Stack

- Go 1.26+
- HTMX + server-rendered HTML templates
- SQLite (via [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite), pure-Go, no CGO)
- Stripe (Checkout, Products, Prices, Tax)

## Architecture

softstore owns the product catalog, cart, and checkout initiation only. It creates Stripe Products/Prices via API when an admin adds a product (and lazily for each multi-seat tier a buyer selects), then redirects customers to Stripe Checkout with the metadata Quartermaster's webhook expects (`product` code, `cart_token`, `product_codes`). softstore never generates, stores, or delivers license keys — that's handled entirely by Quartermaster after Stripe's webhook fires. The two services talk to each other only over small internal HTTP APIs (shared-secret authenticated, WireGuard-only in production): Quartermaster resolves a Stripe Price ID to a product/seat count and clears a buyer's cart after fulfillment; softstore polls Quartermaster for a checkout session's fulfillment status to render the receipt.

## Documentation

| Doc | Covers |
|---|---|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | System overview, sequence diagrams for checkout/fulfillment/thank-you polling, seat-tier pricing algorithm, database schema, package layout, security model, and the full softstore ⇄ Quartermaster API contract. |
| [docs/API.md](docs/API.md) | Every HTTP route — public, admin, and internal — with request/response shapes. |
| [docs/CONFIGURATION.md](docs/CONFIGURATION.md) | Every environment variable, what it's for, and how to generate the secrets. |
| [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) | What `deploy.sh` does, host prerequisites, and example systemd/reverse-proxy setup. |

## Local development

Requires Go 1.26+, and a Stripe test-mode account.

```bash
# required environment variables — see docs/CONFIGURATION.md for what each one
# does and how to generate SESSION_SECRET / ADMIN_PASSWORD_HASH / INTERNAL_API_SECRET
export STRIPE_SECRET_KEY='sk_test_...'
export SESSION_SECRET='...'
export ADMIN_USERNAME='...'
export ADMIN_PASSWORD_HASH='...'
export BASE_URL='https://localhost:8443'
export INTERNAL_API_SECRET='...'
export QUARTERMASTER_INTERNAL_URL='http://...'

go run ./cmd/server
```

The server listens on plain HTTP on `:38217`, expecting a reverse proxy (Caddy in production) to terminate TLS in front of it. `localhost-cert.pem` / `localhost-key.pem` are provided for local tooling that wants a self-signed cert (e.g. a local reverse proxy) but are not used directly by `cmd/server`.

## License

[PolyForm Noncommercial 1.0.0](LICENSE) — free for noncommercial use. Commercial use requires separate authorization. [Quartermaster](https://github.com/laudendev/quartermaster) is licensed under the same terms.
