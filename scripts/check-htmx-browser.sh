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
server_pids=()

cleanup() {
  for server_pid in "${server_pids[@]}"; do
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  done
  rm -rf "$qualification_dir"
}
trap cleanup EXIT INT TERM

note "install HTMX browser test dependencies"
(cd tests/browser && npm ci)

note "type-check HTMX browser tests"
(cd tests/browser && npm run check)

start_example() {
  local name="$1"
  local example_dir="$2"
  local binary="$qualification_dir/$name"
  local log="$qualification_dir/$name.log"
  local server_pid

  note "build and start $example_dir"
  (cd "$repo_root/$example_dir" && go build -o "$binary" .)
  "$binary" -addr 127.0.0.1:0 >"$log" 2>&1 &
  server_pid="$!"
  server_pids+=("$server_pid")

  for _ in {1..100}; do
    local base_url
    base_url="$(sed -n 's/.*listening on \(http:\/\/[^ ]*\).*/\1/p' "$log" | tail -n 1)"
    if [[ -n "$base_url" ]]; then
      started_base_url="$base_url"
      return
    fi
    if ! kill -0 "$server_pid" 2>/dev/null; then
      printf "error: %s stopped before qualification\n" "$example_dir" >&2
      sed -n '1,120p' "$log" >&2
      exit 1
    fi
    sleep 0.05
  done

  printf "error: timed out waiting for %s\n" "$example_dir" >&2
  exit 1
}

started_base_url=""
start_example chat examples/chat
chat_base_url="$started_base_url"
start_example full_feature examples/full_feature
full_feature_base_url="$started_base_url"

note "install local Playwright Chromium"
(cd tests/browser && npm exec playwright install chromium)

note "run HTMX browser qualification"
(cd tests/browser && \
  GOLDR_CHAT_BASE_URL="$chat_base_url" \
  GOLDR_FULL_FEATURE_BASE_URL="$full_feature_base_url" \
  npm run test:browser)

note "HTMX browser qualification passed"
