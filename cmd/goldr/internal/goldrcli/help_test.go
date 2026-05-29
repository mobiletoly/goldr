package goldrcli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRunHelp(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no args"},
		{name: "help", args: []string{"help"}},
		{name: "long help", args: []string{"--help"}},
		{name: "short help", args: []string{"-h"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, stdout, stderr := runGoldr(t, tt.args...)

			if code != 0 {
				t.Fatalf("Run() exit code = %d, want 0", code)
			}
			if !strings.Contains(stdout, "USAGE:") {
				t.Fatalf("stdout = %q, want usage text", stdout)
			}
			if !strings.Contains(stdout, "init") {
				t.Fatalf("stdout = %q, want init command", stdout)
			}
			for _, want := range []string{
				"Common workflow:",
				"go tool goldr generate",
				"go tool goldr check",
				"go test ./...",
				"dev",
				`Use "go tool goldr routes" to inspect the route tree before editing routes.`,
				`Use "go tool goldr assets" for asset-only checks`,
			} {
				if !strings.Contains(stdout, want) {
					t.Fatalf("stdout = %q, want %q", stdout, want)
				}
			}
			for _, futureCommand := range []string{"new", "build"} {
				if strings.Contains(stdout, "\n   "+futureCommand+" ") {
					t.Fatalf("stdout = %q, must not mention future command %q", stdout, futureCommand)
				}
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
		})
	}
}

func TestRunDevHelpExplainsProductionFaithfulLoop(t *testing.T) {
	requireGoldrOutputContains(
		t,
		[]string{"dev", "--help"},
		"goldr dev [--app-root <dir>] [--cmd-dir <dir>] [--app-url <url>] [--proxy-addr <host:port>] [--cmd <command>]",
		"--cmd-dir",
		"templ watch mode",
		"assets.Path",
		"assets.FS",
		"assets/build",
		"not assets/src",
	)
}

func TestRunInitHelp(t *testing.T) {
	code, stdout, stderr := runGoldr(t, "init", "--help")

	if code != 0 {
		t.Fatalf("Run(init --help) exit code = %d, want 0; stderr = %q", code, stderr)
	}
	if !strings.Contains(stdout, "goldr init [--app-root <dir>]") {
		t.Fatalf("stdout = %q, want init usage", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestRunGenerateHelpExplainsGeneratedFilesAndTemplBoundary(t *testing.T) {
	requireGoldrOutputContains(
		t,
		[]string{"generate", "--help"},
		"app/routes/goldr_gen.go",
		"app/routes/**/goldr_gen.go when route packages need generated helpers",
		"app/internal/goldrinspect/goldr_gen.go",
		"app/urls/goldr_gen.go",
		"app/mounts/<mount>/goldr_gen.go for referenced Kit mount subtrees",
		"assets/goldr_assets_gen.go when assets/build exists",
		"go tool templ generate -path .",
		"fingerprints assets/build into assets/dist",
		"verify templ files when present and goldr-generated files",
	)
}

func TestRunCheckHelpExplainsReadOnlyScope(t *testing.T) {
	requireGoldrOutputContains(
		t,
		[]string{"check", "--help"},
		"Read-only validation",
		"route naming",
		"goldr-generated file freshness",
		"templ-generated file freshness",
		"Goldr-managed asset freshness",
		"go tool goldr generate",
		"does not run tests",
		"or write files",
	)
}

func TestRunVersion(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "version", args: []string{"goldr", "version"}},
		{name: "long version", args: []string{"goldr", "--version"}},
		{name: "dash version", args: []string{"goldr", "-version"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			code := Run(context.Background(), tt.args, &stdout, &stderr, "dev")

			if code != 0 {
				t.Fatalf("Run() exit code = %d, want 0", code)
			}
			if got := stdout.String(); got != "goldr dev\n" {
				t.Fatalf("stdout = %q, want %q", got, "goldr dev\n")
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr = %q, want empty", stderr.String())
			}
		})
	}
}

func TestRunUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(context.Background(), []string{"goldr", "unknown"}, &stdout, &stderr, "dev")

	if code != 2 {
		t.Fatalf("Run() exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
	errText := stderr.String()
	if !strings.Contains(errText, `goldr: unknown command "unknown"`) {
		t.Fatalf("stderr = %q, want unknown-command error", errText)
	}
	if !strings.Contains(errText, "USAGE:") {
		t.Fatalf("stderr = %q, want usage text", errText)
	}
}

func TestRunRoutesShowsSubcommandHelp(t *testing.T) {
	requireGoldrOutputContains(
		t,
		[]string{"routes"},
		"goldr routes <command> [options]",
		"Read-only inspection",
		"Use before editing routes",
		"go tool goldr routes explain /users/7",
		"do not write generated files",
		"list",
		"layouts",
		"explain",
		"refs",
	)
}

func TestRunRoutesListHelpShowsRootFlag(t *testing.T) {
	requireGoldrOutputContains(
		t,
		[]string{"routes", "list", "--help"},
		"goldr routes list [--app-root <dir>] [--mount <path>] [--json]",
		"generated URL helper expressions",
		"stable route inventory",
		"--app-root string",
		"--mount string",
		"--json",
	)
}

func TestRunRoutesRefsHelpShowsRootAndJSONFlags(t *testing.T) {
	requireGoldrOutputContains(
		t,
		[]string{"routes", "refs", "--help"},
		"goldr routes refs [--app-root <dir>] [--json]",
		"direct HTMX request attributes",
		"source-level reference inventory",
		"--app-root string",
		"--json",
	)
}

func TestRunAssetsShowsSubcommandHelp(t *testing.T) {
	requireGoldrOutputContains(
		t,
		[]string{"assets"},
		"goldr assets <command> [options]",
		"assets/build -> assets/dist",
		"Goldr does not compile Tailwind",
		"asset-only checks",
		"dist",
		"check",
		"clean",
		"list",
	)
}

func TestRunAssetsDistHelpExplainsFinalFileBoundary(t *testing.T) {
	requireGoldrOutputContains(
		t,
		[]string{"assets", "dist", "--help"},
		"Reads final files from assets/build",
		"assets/goldr_assets_gen.go",
		"final safe-cache step",
		"does not run asset compilers",
	)
}

func requireGoldrOutputContains(t *testing.T, args []string, wants ...string) {
	t.Helper()

	stdout := requireGoldrSuccessOutput(t, strings.Join(args, " "), args...)
	for _, want := range wants {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout = %q, want %q", stdout, want)
		}
	}
}
