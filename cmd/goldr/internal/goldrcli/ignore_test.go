package goldrcli

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestGoldrIgnoreExcludesFreshNestedCheckout(t *testing.T) {
	root := tempGenerateApp(t)
	requireRunSuccess(t, "generate", "--app-root", root)
	archive := filepath.Join(root, "tmp", "archive")
	writeTemplToolModule(t, archive, "example.com/archive")
	writeFile(t, archive, "app/routes/route.go", routeDeclarationSource("routes", "page", routeDeclarationOptions{Page: true}))
	writeFile(t, archive, "app/routes/page.templ", "package routes\n\ntempl PageView(label string) {<p>{ label }</p>}\n")
	requireRunSuccess(t, "generate", "--app-root", archive)
	requireRunSuccess(t, "check", "--app-root", archive)
	writeFile(t, root, ".gitignore", "/tmp/\n")
	writeFile(t, root, ".goldrignore", "# Archived review checkout\ntmp/\n")
	archiveOutput := filepath.Join(archive, "app/routes/page_templ.go")
	before := readFile(t, archiveOutput)
	writeFile(t, root, "tmp/missing.templ", "invalid archived input")
	writeFile(t, root, "tmp/orphan_templ.go", "archived orphan")
	maintained := filepath.Join(root, "app/routes/settings/page_templ.go")
	maintainedBefore := readFile(t, maintained)
	requireRunSuccess(t, "check", "--app-root", root)
	requireRunSuccess(t, "generate", "--check", "--app-root", root)
	writeFile(t, root, "app/routes/page.templ", "package routes\n\ntempl PageView() {<h1>Changed</h1>}\n")
	requireCheckFailureContains(t, root, checkCodeTemplGenerated, "app/routes/page_templ.go")
	requireCommandArgsFailureContains(t, []string{"generate", "--check", "--app-root", root}, "app/routes/page_templ.go")
	requireRunSuccess(t, "generate", "--app-root", root)
	requireRunSuccess(t, "check", "--app-root", root)
	requireRunSuccess(t, "check", "--app-root", archive)
	if readFile(t, archiveOutput) != before || readFile(t, maintained) != maintainedBefore {
		t.Fatal("generation changed archived output or fresh maintained output")
	}
	if got := readFile(t, filepath.Join(root, "tmp/orphan_templ.go")); got != "archived orphan" {
		t.Fatal("generation/check changed an excluded orphan")
	}
	requireMissingFile(t, filepath.Join(root, "tmp/missing_templ.go"))
}

func TestGoldrIgnoreFilePairsAndRootSelection(t *testing.T) {
	root := tempGenerateApp(t)
	writeFile(t, root, ".goldrignore", "notes/scratch.templ\nnotes/missing_templ.go\n")
	writeFile(t, root, "notes/scratch.templ", "invalid archived input")
	writeFile(t, root, "notes/scratch_templ.go", "archived output")
	writeFile(t, root, "notes/missing.templ", "invalid archived input")
	caller := t.TempDir()
	writeFile(t, caller, ".goldrignore", "../invalid\n")
	t.Chdir(caller)
	requireRunSuccess(t, "generate", "--app-root", root)
	requireRunSuccess(t, "check", "--app-root", root)
	requireRunSuccess(t, "generate", "--check", "--app-root", root)
	if got := readFile(t, filepath.Join(root, "notes/scratch_templ.go")); got != "archived output" {
		t.Fatal("excluded template output was rewritten")
	}
	requireMissingFile(t, filepath.Join(root, "notes/missing_templ.go"))
}

func TestGoldrIgnoreAllTemplatesWithoutTool(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/notempltool\n\ngo 1.26.0\n")
	writeFile(t, root, "app/routes/route.go", routeDeclarationSource("routes", "page", routeDeclarationOptions{Page: true}))
	writeFile(t, root, ".goldrignore", "tmp/\n")
	writeFile(t, root, "tmp/page.templ", "invalid archived input")
	requireRunSuccess(t, "generate", "--app-root", root)
	requireRunSuccess(t, "check", "--app-root", root)
	requireRunSuccess(t, "generate", "--check", "--app-root", root)
}

func TestGoldrIgnoreErrorsBeforeGeneration(t *testing.T) {
	root := tempGenerateApp(t)
	requireRunSuccess(t, "generate", "--app-root", root)
	output := filepath.Join(root, "app/routes/page_templ.go")
	before := readFile(t, output)
	writeFile(t, root, "app/routes/page.templ", "package routes\n\ntempl PageView() {<h1>Changed</h1>}\n")
	writeFile(t, root, ".goldrignore", "# comment\n../archive\n")
	requireCheckFailureContains(t, root, checkCodeTemplGenerated, ".goldrignore:2:")
	for _, args := range [][]string{{"generate", "--app-root", root}, {"generate", "--check", "--app-root", root}} {
		requireCommandArgsFailureContains(t, args, "goldr generate:", ".goldrignore:2:")
	}
	if readFile(t, output) != before {
		t.Fatal("invalid ignore file allowed template writes")
	}
}

func TestGoldrIgnoreCommandHelp(t *testing.T) {
	for _, command := range []string{"check", "generate", "dev"} {
		output := requireGoldrSuccessOutput(t, command+" help", command, "--help")
		if !strings.Contains(output, ".goldrignore") {
			t.Fatalf("%s help does not document .goldrignore", command)
		}
	}
}
