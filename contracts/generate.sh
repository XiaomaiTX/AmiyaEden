#!/usr/bin/env bash
set -euo pipefail
cd -- "$(dirname -- "$0")"
bundle=$(mktemp -d)/openapi.json
cleanup() {
  if [[ -f "$bundle" && $(realpath "$bundle") == "$bundle" ]]; then rm -- "$bundle"; fi
  rmdir -- "${bundle%/*}"
}
trap cleanup EXIT
node validate.cjs "$bundle"
mkdir -p ../server/internal/apicontract
go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.5.1 -config go-codegen.yaml -o ../server/internal/apicontract/api.gen.go "$bundle"
OPENAPI_BUNDLE="$bundle" pnpm run generate:ts
