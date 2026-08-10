# Deployment

This describes how `deploy.sh` deploys softstore, and what a host needs to have set up
before it can be used. The systemd unit and reverse-proxy config referenced here are
**not committed to this repository** (they live only on the deploy host) — the examples
below are illustrative templates, not the literal production files.

## What `deploy.sh` does

`deploy.sh` is the only deploy path in this repo. Run from a machine with SSH access to
the production host (aliased `softstore-deploy` in the deployer's SSH config):

```bash
./deploy.sh
```

Step by step:

1. **`go test ./...`** — the deploy aborts immediately (`set -euo pipefail`) if any test
   fails.
2. **Cross-compile** a static Linux binary:
   `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o softstore-linux ./cmd/server`.
   `CGO_ENABLED=0` works because the only native-feeling dependency, SQLite, is the
   pure-Go `modernc.org/sqlite` driver — there's no C toolchain requirement on the
   target host.
3. **Upload** the binary via `scp` to `/opt/softstore-deploy/softstore-linux.new`, then
   `ssh` in to atomically `mv` it over the running binary's path
   (`/opt/softstore-deploy/softstore-linux`) — the rename avoids serving a
   partially-written binary if the upload is interrupted.
4. **Restart**: `sudo systemctl restart softstore` on the remote host.
5. **Verify**: `systemctl is-active softstore` and
   `systemctl show softstore -p ActiveEnterTimestamp`, so the script's own output
   confirms the restart actually succeeded rather than just assuming it did.

The `sleep 3` calls between upload/restart and restart/verify are fixed pauses, not a
health-check poll — verification can in principle run before the new process is fully
up.

## Host prerequisites

`deploy.sh` assumes the following already exist on `softstore-deploy` — it does not
create any of them:

- An SSH host entry named `softstore-deploy` resolvable from the deploying machine.
- `/opt/softstore-deploy/` writable by the SSH user.
- A systemd service unit named `softstore` (see example below) that the deploying
  user can restart via passwordless or interactive `sudo`.
- A reverse proxy terminating TLS in front of the app. `main.go` binds plain HTTP on
  `:38217` and logs `listening on :38217 (http, behind caddy)` — softstore itself never
  handles TLS in production.
- Network reachability to Quartermaster's internal API, scoped to the WireGuard tunnel
  between the two services (see [ARCHITECTURE.md](ARCHITECTURE.md)) — `QUARTERMASTER_INTERNAL_URL`
  should point at an address only reachable over that tunnel, not the public internet.

## Example systemd unit

Illustrative only — adapt paths, user, and the env var list (all seven required
variables from [CONFIGURATION.md](CONFIGURATION.md)) to your host:

```ini
# /etc/systemd/system/softstore.service
[Unit]
Description=softstore
After=network.target

[Service]
Type=simple
User=softstore
WorkingDirectory=/opt/softstore-deploy
ExecStart=/opt/softstore-deploy/softstore-linux
Restart=on-failure
RestartSec=2
EnvironmentFile=/opt/softstore-deploy/softstore.env

[Install]
WantedBy=multi-user.target
```

`EnvironmentFile` should point at a root-readable-only file (`chmod 600`) containing
`KEY=value` lines for every variable in [CONFIGURATION.md](CONFIGURATION.md) — keep
secrets out of the unit file itself and out of shell history.

The service's working directory is where `softstore.db` (SQLite) will be created on
first run — `db.Open("softstore.db")` in `cmd/server/main.go` uses a path relative to
the process's working directory, not an absolute one.

## Example reverse proxy (Caddy)

Illustrative only:

```
store.example.com {
	reverse_proxy localhost:38217
}
```

Caddy's automatic HTTPS handles certificate issuance/renewal; softstore only ever sees
plain HTTP from `localhost`. Set `SECURE_COOKIES=true` in the environment either way —
the cookies are marked `Secure` based on that flag, independent of what's terminating
TLS in front of the app.

## Data

- `softstore.db` is a single SQLite file with no separate migration step — schema is
  created idempotently (`CREATE TABLE IF NOT EXISTS`) on every startup by
  `internal/db.Open`. Back it up with a regular file copy (or `sqlite3 .backup`) rather
  than any application-level export.
- softstore holds no license keys or customer PII beyond what Stripe Checkout itself
  collects — the catalog/cart database is comparatively low-stakes to lose versus
  Quartermaster's fulfillment records.

## Rollback

`deploy.sh` has no built-in rollback. To revert, redeploy an older commit through the
same script (rebuild + re-upload), or manually restore the previous
`/opt/softstore-deploy/softstore-linux` binary from a backup and
`systemctl restart softstore`.

## One-time operational scripts

`cmd/backfill-tax-behavior` is a standalone one-off script (not part of the deployed
service) for correcting `tax_behavior` on Stripe Prices created before Stripe Tax was
enabled. Run it manually, once, against production data when needed:

```bash
STRIPE_SECRET_KEY=sk_live_... SOFTSTORE_DB_PATH=/opt/softstore-deploy/softstore.db \
  go run ./cmd/backfill-tax-behavior
```
