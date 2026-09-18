#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

minimum_go_version="$(awk '$1 == "go" { print $2; exit }' go.mod)"
if [[ -z "$minimum_go_version" ]]; then
  printf "error: root go.mod has no Go version\n" >&2
  exit 1
fi

actual_go_version="$(go env GOVERSION)"
if [[ "$actual_go_version" != "go${minimum_go_version}" ]]; then
  printf "error: minimum Go check requires go%s, running %s\n" "$minimum_go_version" "$actual_go_version" >&2
  exit 1
fi

printf "==> %s\n" "$(go version)"
printf "==> go test ./...\n"
go test ./...

printf "==> (cd cmd/goldr && go test ./...)\n"
(cd cmd/goldr && go test ./...)

for module in examples/*/go.mod; do
  example_dir="${module%/go.mod}"
  printf "==> (cd %s && go test ./...)\n" "$example_dir"
  (cd "$example_dir" && go test ./...)
done

printf "minimum Go compatibility checks passed\n"
