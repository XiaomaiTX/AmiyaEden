#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SERVER_DIR="$ROOT_DIR/server"
STATIC_DIR="$ROOT_DIR/static"
SERVER_CONFIG="$SERVER_DIR/config/config.yaml"
SERVER_CONFIG_EXAMPLE="$SERVER_DIR/config/config.example.yaml"
STATIC_ENV_LOCAL="$STATIC_DIR/.env.development.local"

log() {
  printf '[setup-local] %s\n' "$1"
}

warn() {
  printf '[setup-local] warning: %s\n' "$1" >&2
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    warn "missing required command: $1"
    return 1
  fi
}

log "repo root: $ROOT_DIR"

missing=0
require_cmd go || missing=1
require_cmd node || missing=1
require_cmd pnpm || missing=1

if [ "$missing" -ne 0 ]; then
  warn "install the missing tools first, then rerun this script"
fi

if [ ! -f "$SERVER_CONFIG" ]; then
  cp "$SERVER_CONFIG_EXAMPLE" "$SERVER_CONFIG"
  log "created backend config: $SERVER_CONFIG"
else
  log "backend config already exists: $SERVER_CONFIG"
fi

if [ ! -f "$STATIC_ENV_LOCAL" ]; then
  cat >"$STATIC_ENV_LOCAL" <<'EOF'
VITE_PORT=5173
VITE_VERSION=dev
VITE_WITH_CREDENTIALS=false
EOF
  log "created frontend local env: $STATIC_ENV_LOCAL"
else
  log "frontend local env already exists: $STATIC_ENV_LOCAL"
fi

if command -v go >/dev/null 2>&1; then
  log "downloading Go modules"
  (
    cd "$SERVER_DIR"
    go mod download
  )
else
  warn "skipped go mod download"
fi

if command -v pnpm >/dev/null 2>&1; then
  log "installing frontend packages"
  (
    cd "$STATIC_DIR"
    pnpm install
  )
else
  warn "skipped pnpm install"
fi

cat <<EOF

Next steps:
1. Edit $SERVER_CONFIG
2. Start PostgreSQL and Redis locally
3. Run backend:
   cd $SERVER_DIR && go run main.go
4. Run frontend:
   cd $STATIC_DIR && pnpm dev
5. Smoke test:
   curl http://localhost:8080/api/v1/sso/eve/scopes

Notes:
- The backend reads ./config/config.yaml automatically.
- The frontend proxy target is defined in $STATIC_DIR/.env.development.
- This app uses EVE SSO only; there is no local username/password login.
EOF
