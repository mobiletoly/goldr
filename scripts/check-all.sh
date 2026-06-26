#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

check_tools_dir="tools/check"

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

run_in() {
  local dir="$1"
  shift

  note "(cd $dir && $*)"
  (cd "$dir" && "$@")
}

check_tool_path() {
  local name="$1"

  (cd "$check_tools_dir" && go tool -n "$name")
}

run_check_tool() {
  local name="$1"
  shift

  local tool
  tool="$(check_tool_path "$name")"

  note "$name $*"
  "$tool" "$@"
}

fail() {
  printf "error: %s\n" "$*" >&2
  exit 1
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

check_readme_ascii_allow_star() {
  note "check README ASCII text"

  # README.md may use U+2B50 in the public GitHub star callout. Keep every
  # other README character ASCII so smart quotes, em dashes, and invisible
  # Unicode still fail the gate.
  local star
  star="$(printf '\342\255\220')"

  local matches
  local status

  set +e
  matches="$(sed "s/${star}//g" README.md | LC_ALL=C grep -n '[^[:print:][:space:]]')"
  status=$?
  set -e

  if [[ "$status" -eq 0 ]]; then
    printf "%s\n" "$matches"
    fail "check README ASCII text failed"
  fi

  if [[ "$status" -ne 1 ]]; then
    fail "check README ASCII text scan failed"
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

  run_check_tool golangci-lint run
}

check_gopls_hints() {
  if [[ "${GOLDR_SKIP_GOPLS_HINTS:-0}" == "1" ]]; then
    note "skip gopls hints (GOLDR_SKIP_GOPLS_HINTS=1)"
    return
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
  local gopls

  gopls="$(check_tool_path gopls)"

  set +e
  output="$("$gopls" check -severity=hint "${go_files[@]}" 2>&1)"
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

check_cli_module_self_contained() {
  note "check CLI module has no root module dependency"

  local deps
  deps="$(cd cmd/goldr && go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./... | sort -u)"
  if printf "%s\n" "$deps" | grep -x 'github.com/mobiletoly/goldr' >/dev/null; then
    printf "%s\n" "$deps"
    fail "CLI module depends on root module"
  fi
}

check_downstream_tool_install() {
  note "check downstream go get -tool goldr"

  local tmp
  tmp="$(mktemp -d)"
  local status=0

  (
    cd "$tmp"
    go mod init example.com/goldr-tool-probe >/dev/null
    go mod edit -replace github.com/mobiletoly/goldr/cmd/goldr="$repo_root/cmd/goldr"
    go get -tool github.com/mobiletoly/goldr/cmd/goldr
    go tool goldr --help >/dev/null
  ) || status=$?

  rm -rf "$tmp"
  return "$status"
}

require_cmd go
require_cmd gofmt
require_cmd git

run_in "$check_tools_dir" go mod tidy -diff
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

run_in cmd/goldr go mod tidy -diff
run_in cmd/goldr go list ./...
check_cli_module_self_contained
run_in cmd/goldr go test ./...
run_in cmd/goldr go vet ./...

if [[ "${GOLDR_SKIP_RACE:-0}" == "1" ]]; then
  note "skip cmd/goldr go test -race (GOLDR_SKIP_RACE=1)"
else
  run_in cmd/goldr go test -race ./...
fi

run_in cmd/goldr go run . --help
check_downstream_tool_install

example_modules=(
  "examples/full_feature"
  "examples/chat"
  "examples/kit_routes"
)

for example_module in "${example_modules[@]}"; do
  run_in "$example_module" go mod tidy -diff
  run_in "$example_module" go tool goldr generate --check
  run_in "$example_module" go tool goldr check
  run_in "$example_module" go test ./...
done

layout_map_unicode_pathspecs=(
  "."
  ":(exclude)README.md"
  ":(exclude)cmd/goldr/internal/goldrcli/routes/layouts.go"
  ":(exclude)cmd/goldr/internal/goldrcli/app_test.go"
  ":(exclude)docs/user/cli.md"
  ":(exclude)docs/spec/a0008-route-layout-map.md"
)
check_no_git_grep_matches "check ASCII text" '[^[:print:][:space:]]' "${layout_map_unicode_pathspecs[@]}"
check_readme_ascii_allow_star
check_no_git_grep_matches "check trailing whitespace" '[[:blank:]]$'
check_golangci_lint
check_gopls_hints
check_govulncheck

printf "\nall checks passed\n"
