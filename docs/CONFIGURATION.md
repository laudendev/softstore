# Configuration Reference

softstore is configured entirely through environment variables, read once at startup by
`internal/config`. Every variable below except `SECURE_COOKIES` is required —
`config.requireEnv` calls `log.Fatalf` and the process exits immediately if any of them
is unset or empty, so there's no way to start the server in a partially-configured
state.

## Required

| Variable | Read by | Purpose |
|---|---|---|
| `STRIPE_SECRET_KEY` | `config.StripeSecretKey()` | Stripe API secret key (`sk_test_...` / `sk_live_...`). Assigned directly to `stripe.Key` in `main.go` — all Stripe SDK calls use it globally. |
| `SESSION_SECRET` | `config.SessionSecret()` | HMAC key for signing the admin session cookie (`internal/auth`). Treat as a long-lived secret: rotating it invalidates every existing admin session. |
| `ADMIN_USERNAME` | `config.AdminUsername()` | The single admin account's username. Compared in constant time against login submissions. |
| `ADMIN_PASSWORD_HASH` | `config.AdminPasswordHash()` | A bcrypt hash of the admin password (see [Generating secrets](#generating-secrets) below). Never store the plaintext password anywhere. |
| `BASE_URL` | `config.BaseURL()` | The externally-reachable origin (e.g. `https://store.example.com`), used to build Stripe Checkout's `success_url` and `cancel_url`. Must be a URL Stripe can send the customer's browser back to. |
| `INTERNAL_API_SECRET` | `config.InternalAPISecret()` | Static shared secret authenticating traffic between softstore and Quartermaster, checked via `crypto/subtle.ConstantTimeCompare` on both sides. Must be set to the **same value** in both services' environments. |
| `QUARTERMASTER_INTERNAL_URL` | `config.QuartermasterInternalURL()` | Base URL of Quartermaster's internal API (e.g. `http://10.20.0.2:6774`), used by softstore to poll checkout-session fulfillment status. In production this should only be reachable over the WireGuard tunnel between the two hosts, not the public internet. |

## Optional

| Variable | Read by | Default | Purpose |
|---|---|---|---|
| `SECURE_COOKIES` | `config.SecureCookies()` | unset → `false` | Set to exactly `"true"` (any other value, including unset, evaluates to `false`) to mark the admin session and cart cookies `Secure` (HTTPS-only). **Must** be `"true"` in production — see [ARCHITECTURE.md § Security model](ARCHITECTURE.md#security-model). Leave unset for local HTTP development. |

## Generating secrets

**`SESSION_SECRET`** — 32 random bytes, base64-encoded:

```bash
openssl rand -base64 32
```

**`INTERNAL_API_SECRET`** — any sufficiently random string, shared verbatim between
softstore and Quartermaster's environments:

```bash
openssl rand -hex 32
```

**`ADMIN_PASSWORD_HASH`** — a bcrypt hash. softstore already depends on
`golang.org/x/crypto/bcrypt`, so the simplest way to generate a matching hash is a
throwaway script run from inside this repo (so the dependency is available), using the
same `bcrypt.DefaultCost` the codebase uses elsewhere (see `internal/auth/auth_test.go`):

```bash
cat > hash_password.go <<'EOF'
package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	hash, err := bcrypt.GenerateFromPassword([]byte(os.Args[1]), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(hash))
}
EOF
go run hash_password.go 'your-password-here'
rm hash_password.go
```

Copy the printed hash (starts with `$2a$` or `$2b$`) into `ADMIN_PASSWORD_HASH`.

## Local development `.env`

`main.go` reads only from the process environment (via `os.Getenv`), not from a `.env`
file directly — export the variables in your shell, or use a tool like `direnv` or
`dotenv` to load a local `.env` before running `go run ./cmd/server`. `.env` is already
listed in `.gitignore`, so it's safe to keep secrets there locally.

```bash
export STRIPE_SECRET_KEY='sk_test_...'
export SESSION_SECRET='<output of: openssl rand -base64 32>'
export ADMIN_USERNAME='admin'
export ADMIN_PASSWORD_HASH='<output of the bcrypt snippet above>'
export BASE_URL='https://localhost:8443'
export INTERNAL_API_SECRET='<output of: openssl rand -hex 32>'
export QUARTERMASTER_INTERNAL_URL='http://localhost:6774'
# SECURE_COOKIES intentionally left unset for local HTTP development
```
