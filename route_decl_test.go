// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestKitRouteDefRejectsWrongReceiverAtCompileTime(t *testing.T) {
	repoRoot := goldrRepoRoot(t)
	root := t.TempDir()
	writeTempFile(t, root, "go.mod", "module route_decl_compile_test\n\ngo 1.26\n\nrequire github.com/mobiletoly/goldr v0.0.0\n\nreplace github.com/mobiletoly/goldr => "+filepath.ToSlash(repoRoot)+"\n")
	writeTempFile(t, root, "route.go", `package route_decl_compile_test

import (
	"net/http"

	"github.com/mobiletoly/goldr"
)

type portal struct{}
type kit struct{}
type otherKit struct{}

func newKit(*http.Request) kit { return kit{} }
func (otherKit) page(*http.Request) goldr.RouteResponse {
	return goldr.Text{Body: "wrong"}
}

var Route = goldr.KitRouteDef[kit]{
	New:  newKit,
	Page: goldr.KitPage(otherKit.page),
}
`)

	cmd := exec.CommandContext(context.Background(), "go", "test", "-mod=mod", ".")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("go test unexpectedly succeeded; output:\n%s", output)
	}
	if !strings.Contains(string(output), "cannot use goldr.KitPage(otherKit.page)") {
		t.Fatalf("go test output = %s, want wrong receiver type error", output)
	}
}

func goldrRepoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(file)
}

func writeTempFile(t *testing.T, root string, relPath string, content string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}
