#!/usr/bin/env bash
set -euo pipefail
cd -- "$(dirname -- "$0")"
bash generate.sh
(cd ../server && go test ./bootstrap -run '^TestStartupV1RoutesMatchOpenAPI$' -count=1 -v)
paths=(../server/internal/apicontract ../static/src/api/generated ../static-react/src/api/generated)
# Compare to the index so an already-staged, reproducible migration can be checked locally too.
git diff --exit-code -- "${paths[@]}"
if [[ -n $(git ls-files --others --exclude-standard -- "${paths[@]}") ]]; then
  git ls-files --others --exclude-standard -- "${paths[@]}"
  echo 'New generated files must be committed.' >&2
  exit 1
fi
