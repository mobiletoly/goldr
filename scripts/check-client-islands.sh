#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf "error: required command not found: %s\n" "$1" >&2
    exit 1
  fi
}

note() {
  printf "\n==> %s\n" "$*"
}

require_cmd diff
require_cmd go
require_cmd npm
require_cmd node
require_cmd sed

node_version="$(node -p 'process.versions.node')"
node_major="${node_version%%.*}"
node_remainder="${node_version#*.}"
node_minor="${node_remainder%%.*}"
if (( node_major < 20 || (node_major == 20 && node_minor < 19) )); then
  printf "error: Node 20.19 or newer is required; found %s\n" "$node_version" >&2
  exit 1
fi

qualification_dir="$(mktemp -d)"
server_pid=""

cleanup() {
  if [[ -n "$server_pid" ]]; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
  rm -rf "$qualification_dir"
}
trap cleanup EXIT INT TERM

example_modules=(
  "examples/react_island"
  "examples/svelte_island"
)

for example_module in "${example_modules[@]}"; do
  example_name="${example_module##*/}"
  build_dir="$qualification_dir/$example_name-build"

  note "(cd $example_module && npm ci)"
  (cd "$example_module" && npm ci)

  note "(cd $example_module && npm run check)"
  (cd "$example_module" && npm run check)

  note "rebuild $example_module frontend into temporary output"
  (cd "$example_module" && npm exec vite build -- --outDir "$build_dir")

  note "compare $example_module committed frontend output"
  diff -ru "$example_module/assets/build" "$build_dir"

  note "check $example_module Goldr output and Go tests"
  (cd "$example_module" && go tool goldr generate --check)
  (cd "$example_module" && go tool goldr check)
  (cd "$example_module" && go test ./...)
done

note "install local Playwright Chromium"
(cd examples/react_island && npm exec playwright install chromium)

for example_module in "${example_modules[@]}"; do
  example_name="${example_module##*/}"
  binary="$qualification_dir/$example_name"
  log="$qualification_dir/$example_name.log"

  note "build and start $example_module"
  (cd "$example_module" && go build -o "$binary" .)
  "$binary" -addr 127.0.0.1:0 >"$log" 2>&1 &
  server_pid=$!

  base_url=""
  for _ in {1..100}; do
    base_url="$(sed -n 's/.*listening on \(http:\/\/[^ ]*\).*/\1/p' "$log" | tail -n 1)"
    if [[ -n "$base_url" ]]; then
      break
    fi
    if ! kill -0 "$server_pid" 2>/dev/null; then
      printf "error: %s stopped before qualification\n" "$example_module" >&2
      sed -n '1,120p' "$log" >&2
      exit 1
    fi
    sleep 0.05
  done
  if [[ -z "$base_url" ]]; then
    printf "error: timed out waiting for %s\n" "$example_module" >&2
    exit 1
  fi

  note "run $example_module Playwright lifecycle test"
  (cd "$example_module" && GOLDR_ISLAND_BASE_URL="$base_url" npm exec playwright test)

  kill "$server_pid"
  wait "$server_pid" 2>/dev/null || true
  server_pid=""
done

note "client-island qualification passed"
