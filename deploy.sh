#!/bin/bash
set -euo pipefail

echo "Testing..."
go test ./...

echo "Building softstore..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o softstore-linux ./cmd/server

echo "Deploying softstore..."
scp softstore-linux softstore-deploy:/opt/softstore-deploy/softstore-linux.new
ssh softstore-deploy "mv /opt/softstore-deploy/softstore-linux.new /opt/softstore-deploy/softstore-linux"

echo "Waiting for redeploy..."
sleep 3

echo "Restarting service..."
ssh softstore-deploy "sudo systemctl restart softstore"

echo "Waiting for restart..."
sleep 3

echo "Verifying softstore..."
ssh softstore-deploy "systemctl is-active softstore && systemctl show softstore -p ActiveEnterTimestamp"

echo "Done."
