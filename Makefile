SHELL := /bin/bash

SERVER_DIR := server
VUE_DIR := static
REACT_DIR := static-react

.PHONY: dev \
	lint lint-backend lint-vue lint-react \
	lint-fix lint-fix-backend lint-fix-vue lint-fix-react \
	typecheck typecheck-vue typecheck-react \
	test test-backend test-vue test-react \
	build build-backend build-vue build-react \
	check-react-contract \
	verify

dev:
	@command -v air >/dev/null 2>&1 || { echo "air is not installed. Install it before running 'make dev'."; exit 1; }
	@command -v pnpm >/dev/null 2>&1 || { echo "pnpm is not installed. Install it before running 'make dev'."; exit 1; }
	@set -u; \
	trap 'status=$$?; kill $$frontend_pid $$backend_pid 2>/dev/null || true; wait $$frontend_pid $$backend_pid 2>/dev/null || true; exit $$status' INT TERM EXIT; \
	(cd static && pnpm dev) & frontend_pid=$$!; \
	air -c .air.toml & backend_pid=$$!; \
	wait -n $$frontend_pid $$backend_pid; \
	status=$$?; \
	kill $$frontend_pid $$backend_pid 2>/dev/null || true; \
	wait $$frontend_pid $$backend_pid 2>/dev/null || true; \
	exit $$status

# ---- Backend (server) ----

lint-backend:
	cd $(SERVER_DIR) && golangci-lint run ./...

lint-fix-backend:
	cd $(SERVER_DIR) && golangci-lint run --fix ./...

test-backend:
	cd $(SERVER_DIR) && go test ./...

build-backend:
	cd $(SERVER_DIR) && go build ./...

# ---- Vue frontend (static) ----

lint-vue:
	cd $(VUE_DIR) && pnpm lint .

lint-fix-vue:
	cd $(VUE_DIR) && pnpm fix

typecheck-vue:
	cd $(VUE_DIR) && pnpm exec vue-tsc --noEmit

test-vue:
	cd $(VUE_DIR) && pnpm test:unit

build-vue:
	cd $(VUE_DIR) && pnpm build

# ---- React frontend (static-react) ----

lint-react:
	cd $(REACT_DIR) && pnpm lint

lint-fix-react:
	cd $(REACT_DIR) && pnpm lint --fix

typecheck-react:
	cd $(REACT_DIR) && pnpm exec tsc -b

test-react:
	cd $(REACT_DIR) && pnpm test

check-react-contract:
	cd $(REACT_DIR) && pnpm check:api-contract

build-react:
	cd $(REACT_DIR) && pnpm build

# ---- Aggregates ----

lint: lint-backend lint-vue lint-react

lint-fix: lint-fix-backend lint-fix-vue lint-fix-react

typecheck: typecheck-vue typecheck-react

test: test-backend test-vue test-react

build: build-backend build-vue build-react

verify: lint typecheck test build check-react-contract
