# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

softstore is a storefront for selling downloadable software and digital books, built in Go with HTMX + server-rendered HTML templates. It owns the product catalog, cart, admin management, and Stripe Checkout initiation only — it never generates, stores, or delivers license keys. That's handled by a separate service, **Quartermaster**, which listens to Stripe webhooks after checkout completes. softstore talks to Quartermaster over an internal HTTP API (WireGuard-only in production) to poll fulfillment status for the thank-you page, and Quartermaster calls back into softstore's `/internal/*` endpoints (shared-secret authenticated) to look up product metadata and clear carts after purchase.

Stack: Go 1.26+, HTMX, SQLite via `modernc.org/sqlite` (pure-Go, no CGO), Stripe (Checkout, Products, Prices, Tax).

## Commands

```bash
# Run the dev server (required env vars below)
go run ./cmd/server

# Run all tests
go test ./...

# Run a single package's tests
go test ./internal/handlers/...

# Run a single test
go test ./internal/handlers/ -run TestCheckout

# Build
go build ./...

# One-time operational script: backfill Stripe tax_behavior on existing prices
STRIPE_SECRET_KEY=sk_... go run ./cmd/backfill-tax-behavior
```

Required environment variables (see `internal/config`): `STRIPE_SECRET_KEY`, `SESSION_SECRET` (base64, 32 random bytes), `ADMIN_USERNAME`, `ADMIN_PASSWORD_HASH` (bcrypt), `BASE_URL`, `INTERNAL_API_SECRET`, `QUARTERMASTER_INTERNAL_URL`. `SECURE_COOKIES=true` enables secure-only cookies (auth session + cart); local dev typically runs HTTP-only.

`deploy.sh` is the production deploy path: runs `go test ./...`, cross-compiles for linux/amd64 (`CGO_ENABLED=0`), scp's the binary to `softstore-deploy`, and restarts the systemd service. In production the server listens on `:38217` behind Caddy (not `:8443`/TLS directly — that's local-only via `localhost-cert.pem`/`localhost-key.pem`, not committed).

## Architecture

**Provider seam.** All payment operations go through the `payments.Provider` interface (`internal/payments/payments.go`) — `RegisterItem`, `AddPrice`, `UpdateProductDescription`, `StartPurchase`. Handlers never import the Stripe SDK directly; they depend only on this interface. The real implementation is `internal/payments/stripeprovider`; `internal/payments/mockprovider` is a hand-rolled test double (records calls, overridable behavior via `*Func` fields) used throughout the handler tests instead of hitting Stripe.

**Seat-tier pricing.** A product's `stripe_price_id` (set at creation) is always its 1-seat price. Multi-seat tiers (2–6 devices) are created lazily and cached: `handlers.GetOrCreatePriceForSeats` (`internal/handlers/product_pricing.go`) checks the `product_prices` table for an existing Stripe Price for that product+seat combination, and if absent, computes a discounted total (`deviceDiscountTiers`: 10% at 2 seats, 15% at 3–4, 20% at 5–6) and registers a **dedicated Stripe Product+Price** for that tier via the provider, then persists it for reuse. This is deliberately not a shared Product with a mutated description — a cart can contain multiple tiers of the same product simultaneously, and each line item needs to show its own tier at checkout, which one mutable Product can't do.

**Routing.** All routes are registered inline in `cmd/server/main.go` on a single `http.ServeMux`, using Go 1.22+ method-prefixed patterns (e.g. `"POST /checkout/{slug}"`). Handler constructors are closures that take their dependencies (`*sql.DB`, `payments.Provider`, parsed templates) and return an `http.HandlerFunc` — there's no framework or DI container. Two middleware wrappers compose over handlers: `handlers.RequireAdmin` (session-cookie auth, redirects to `/admin/login`) and `handlers.RequireInternalSecret` (constant-time shared-secret check for Quartermaster-facing endpoints).

**Templates.** Parsed once at startup in `main.go` via `template.ParseFS` against the embedded `web.Templates` FS (`web/embed.go`). Each page composes `layout.html` with its own fragment(s); legal pages share a cloned `legal_layout.html` base so each gets its own `template.Template` instance. HTMX fragments (e.g. `cart_drawer.html`, `session_status_fragment.html`) are parsed separately for partial-page swaps.

**Data layer.** `internal/db` wraps `database/sql` over SQLite; `db.Open` runs an idempotent `CREATE TABLE IF NOT EXISTS` migration inline (no migration framework). Four tables: `products`, `carts`, `cart_items` (unique on cart+product+seats, so the same product at different seat counts is a separate line), `product_prices` (unique on product+seats, the seat-tier cache described above). `internal/models` holds plain structs plus display helpers (e.g. `PriceDollars()`, `CartItem.LineTotalCents()`) — no ORM.

**Cart identity vs. admin auth.** Two separate cookie schemes, both in dedicated packages: `internal/cartsession` issues an anonymous random-token cookie (`SameSite=Lax`, 30 days) to identify a guest's cart with no login required; `internal/auth` issues an HMAC-signed, expiring session token (`SameSite=Strict`, 24h) for the single hardcoded admin user. Both packages expose a package-level `SecureCookies` bool toggled once at startup from `config.SecureCookies()`.

**Cross-service contract.** `internal/quartermaster/client.go` is the outbound half of the Quartermaster integration — it polls `GET {QUARTERMASTER_INTERNAL_URL}/internal/sessions/{id}/status` for receipt/fulfillment data shown on the thank-you page. The inbound half is `internal/handlers/internal_api.go`, exposing `GET /internal/products/by-price/{price_id}` and `POST /internal/cart/clear` for Quartermaster to call back into softstore. Changing the shape of `quartermaster.SessionStatus` or the metadata sent in `payments.PurchaseRequest.Metadata` (`product` code, `seats`, `cart_token`, `product_codes`) is a cross-repo contract change — Quartermaster's webhook handler depends on it.
