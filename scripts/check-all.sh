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

run() {
  note "$*"
  "$@"
}

fail() {
  printf "error: %s\n" "$*" >&2
  exit 1
}

optional_missing() {
  local name="$1"

  if [[ "${GOLDR_REQUIRE_OPTIONAL_TOOLS:-0}" == "1" ]]; then
    fail "optional command is required but not installed: $name"
  fi

  note "skip $name (not installed)"
}

check_gofmt() {
  note "check gofmt"

  local files
  files="$(gofmt -l .)"

  if [[ -n "$files" ]]; then
    printf "%s\n" "$files"
    fail "Go files need gofmt"
  fi
}

check_no_git_grep_matches() {
  local label="$1"
  local pattern="$2"
  shift 2

  note "$label"

  local pathspecs=("$@")
  if [[ "${#pathspecs[@]}" -eq 0 ]]; then
    pathspecs=(".")
  fi

  local matches
  local status

  set +e
  matches="$(LC_ALL=C git grep -nI --untracked --exclude-standard -e "$pattern" -- "${pathspecs[@]}")"
  status=$?
  set -e

  if [[ "$status" -eq 0 ]]; then
    printf "%s\n" "$matches"
    fail "$label failed"
  fi

  if [[ "$status" -ne 1 ]]; then
    fail "$label scan failed"
  fi
}

check_golangci_lint() {
  if [[ "${GOLDR_SKIP_GOLANGCI_LINT:-0}" == "1" ]]; then
    note "skip golangci-lint (GOLDR_SKIP_GOLANGCI_LINT=1)"
    return
  fi

  if [[ ! -f ".golangci.yml" && ! -f ".golangci.yaml" && ! -f ".golangci.toml" && ! -f ".golangci.json" ]]; then
    note "skip golangci-lint (no config)"
    return
  fi

  if ! command -v golangci-lint >/dev/null 2>&1; then
    optional_missing "golangci-lint"
    return
  fi

  run golangci-lint run
}

check_gopls_hints() {
  if [[ "${GOLDR_SKIP_GOPLS_HINTS:-0}" == "1" ]]; then
    note "skip gopls hints (GOLDR_SKIP_GOPLS_HINTS=1)"
    return
  fi

  if ! command -v gopls >/dev/null 2>&1; then
    fail "gopls is required for modernization hint checks"
  fi

  note "gopls check -severity=hint"

  local go_files=()
  local path

  while IFS= read -r path; do
    if [[ -e "$path" ]]; then
      go_files+=("$path")
    fi
  done < <(git ls-files --cached --others --exclude-standard -- '*.go')

  if [[ "${#go_files[@]}" -eq 0 ]]; then
    return
  fi

  local output
  local status

  set +e
  output="$(gopls check -severity=hint "${go_files[@]}" 2>&1)"
  status=$?
  set -e

  if [[ -n "$output" ]]; then
    printf "%s\n" "$output"
    fail "gopls hint diagnostics found"
  fi

  if [[ "$status" -ne 0 ]]; then
    fail "gopls check failed"
  fi
}

check_govulncheck() {
  if [[ "${GOLDR_SKIP_GOVULNCHECK:-0}" == "1" ]]; then
    note "skip govulncheck (GOLDR_SKIP_GOVULNCHECK=1)"
    return
  fi

  if [[ "${GOLDR_RUN_GOVULNCHECK:-0}" != "1" ]]; then
    note "skip govulncheck (set GOLDR_RUN_GOVULNCHECK=1)"
    return
  fi

  if ! command -v govulncheck >/dev/null 2>&1; then
    fail "govulncheck requested but not installed"
  fi

  run govulncheck ./...
}

require_cmd go
require_cmd gofmt
require_cmd git

check_gofmt
run go mod tidy -diff
run go list ./...
run go test ./...
run go vet ./...

if [[ "${GOLDR_SKIP_RACE:-0}" == "1" ]]; then
  note "skip go test -race (GOLDR_SKIP_RACE=1)"
else
  run go test -race ./...
fi

layout_map_unicode_pathspecs=(
  "."
  ":(exclude)internal/goldrcli/routes/layouts.go"
  ":(exclude)internal/goldrcli/app_test.go"
  ":(exclude)docs/user/cli.md"
  ":(exclude)docs/spec/a0008-route-layout-map.md"
)
check_no_git_grep_matches "check ASCII text" '[^[:print:][:space:]]' "${layout_map_unicode_pathspecs[@]}"
check_no_git_grep_matches "check trailing whitespace" '[[:blank:]]$'
check_golangci_lint
check_gopls_hints
check_govulncheck

printf "\nall checks passed\n"
