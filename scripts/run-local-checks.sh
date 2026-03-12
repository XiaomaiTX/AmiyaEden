#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SERVER_DIR="$ROOT_DIR/server"
STATIC_DIR="$ROOT_DIR/static"

printf '[run-local-checks] backend tests\n'
(
  cd "$SERVER_DIR"
  go test ./...
)

printf '[run-local-checks] backend build\n'
(
  cd "$SERVER_DIR"
  go build ./...
)

printf '[run-local-checks] frontend lint\n'
(
  cd "$STATIC_DIR"
  pnpm lint
)

printf '[run-local-checks] frontend build\n'
(
  cd "$STATIC_DIR"
  pnpm build
)
