# API Reference

Every HTTP route registered in `cmd/server/main.go`, grouped by audience. Routes use Go
1.22+ method-prefixed `http.ServeMux` patterns (e.g. `POST /checkout/{slug}`); there is
no separate router or framework.

For the service-to-service contract with Quartermaster specifically (request/response
shapes, checkout metadata), see
[ARCHITECTURE.md § The softstore ⇄ Quartermaster contract](ARCHITECTURE.md#the-softstore--quartermaster-contract) —
the internal routes are also listed here for completeness.

## Conventions

- Routes returning **HTML** render the full page (extends `layout.html`) unless noted
  as an **HTMX fragment**, which returns a partial swapped into the existing page.
- Form-encoded routes read `application/x-www-form-urlencoded` bodies via
  `r.ParseForm()` / `r.FormValue(...)`; JSON routes are noted explicitly.
- `seats` form values are parsed by `parseSeatsForm` (`internal/handlers/cart.go`):
  missing, empty, zero, negative, or unparseable defaults to `1`; anything above `6` is
  clamped to `6`. There is no error response for an out-of-range value — it's silently
  clamped.
- Errors are plain-text `http.Error` responses (no JSON error envelope) except where
  noted.

## Public / customer-facing routes

| Method | Path | Handler | Notes |
|---|---|---|---|
| `GET` | `/` | `Shop` | Renders the catalog page. |
| `GET` | `/cart` | `GetCart` | HTMX fragment: cart drawer contents for the caller's cart cookie. |
| `POST` | `/cart/add/{slug}` | `AddToCart` | Form: `seats` (optional). Adds 1 unit at the given seat tier; lazily creates the Stripe Price for that tier if new. HTMX fragment + out-of-band `#cart-count` badge update. |
| `POST` | `/cart/remove/{slug}` | `RemoveFromCart` | Form: `seats` (optional, must match the line item's tier). HTMX fragment + badge update. |
| `POST` | `/checkout/{slug}` | `Checkout` | Form: `seats` (optional). Single-item "buy now" — bypasses the cart. `303` redirect to the Stripe Checkout URL. |
| `POST` | `/checkout` | `CartCheckout` | Builds one Stripe Checkout Session from every item in the caller's cart. `400` if the cart is empty. `303` redirect to Stripe. |
| `GET` | `/thank-you` | `ThankYou` | Query: `session_id`. Renders the post-checkout page shell in a loading state. |
| `GET` | `/session-status/{session_id}` | `SessionStatus` | HTMX fragment. Polls Quartermaster and renders either the loading state or the receipt (line items, tax, total, license keys). A poll failure renders the loading state again rather than erroring. |
| `GET` | `/legal/terms` | `LegalPage` | Static Terms of Service page. |
| `GET` | `/legal/privacy` | `LegalPage` | Static Privacy Policy page. |
| `GET` | `/legal/eula` | `LegalPage` | Static EULA page. |
| `GET` | `/legal/refunds` | `LegalPage` | Static Refund Policy page. |
| `GET` | `/legal/cookies` | `LegalPage` | Static Cookie Policy page. |
| `GET` | `/health` | inline in `main.go` | Returns `200 ok` (plain text). No dependency checks — liveness only, not readiness. |
| `GET` | `/static/*` | `http.FileServer` | Serves `web/static` (CSS, images) from the embedded FS. |

## Admin routes

All routes under `/admin/products/*` (except login) are wrapped in
`handlers.RequireAdmin`, which checks the `softstore_admin_session` cookie and redirects
to `/admin/login` if missing/invalid/expired. There is exactly one admin account,
configured via `ADMIN_USERNAME` / `ADMIN_PASSWORD_HASH` — no user table, no
registration flow.

| Method | Path | Handler | Notes |
|---|---|---|---|
| `GET` | `/admin/login` | `AdminLoginForm` | Renders the login form. |
| `POST` | `/admin/login` | `AdminLoginSubmit` | Form: `username`, `password`. On success, sets the signed session cookie and redirects to `/admin/products/new`. On failure, re-renders the form with an error — no redirect, no lockout/rate-limiting. |
| `POST` | `/admin/logout` | `AdminLogout` | Clears the session cookie, redirects to `/admin/login`. |
| `GET` | `/admin/products/new` | `AdminNew` (behind `RequireAdmin`) | Renders the "add product" form. |
| `POST` | `/admin/products` | `AdminCreateProduct` (behind `RequireAdmin`) | Form: `name`, `slug`, `description`, `price` (dollars, e.g. `19.99`), `product_code` (exactly 4 characters), `seats` (whole number ≥ 1, this is the product's *base* seat count, not a tier), `tax_code`, `stub_url`. Registers a new Stripe Product + 1-seat Price, then inserts the `products` row. Returns an HTML fragment (`<p class="error">…</p>` or `<p class="success">…</p>`), not a redirect. |

## Internal routes (service-to-service)

Guarded by `handlers.RequireInternalSecret`, which compares the `X-Internal-Secret`
header against `INTERNAL_API_SECRET` using `crypto/subtle.ConstantTimeCompare`. These
exist for Quartermaster to call over the WireGuard tunnel and are not meant to be
internet-reachable — see
[ARCHITECTURE.md § Security model](ARCHITECTURE.md#security-model).

### `GET /internal/products/by-price/{price_id}`

Resolves a Stripe Price ID — either a product's default 1-seat price or a cached
seat-tier price from `product_prices` — back to softstore's product code and the seat
count that price actually represents.

```
GET /internal/products/by-price/price_1AbC...
X-Internal-Secret: <INTERNAL_API_SECRET>
```

```json
200 OK
{
  "product_code": "SOFT",
  "name": "Widget Pro",
  "seats": 3
}
```

`404` if no product or `product_prices` row matches that Price ID.

### `POST /internal/cart/clear`

Deletes a cart (and its items) by cart token, so a fulfilled cart doesn't linger and
reappear on the buyer's next visit.

```
POST /internal/cart/clear
X-Internal-Secret: <INTERNAL_API_SECRET>
Content-Type: application/json

{ "cart_token": "..." }
```

`200 OK` on success. `400 Bad Request` if the body is missing or `cart_token` is empty.
