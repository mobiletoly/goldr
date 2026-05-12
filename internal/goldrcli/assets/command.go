// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package assets

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/mobiletoly/goldr/internal/goldrcli/appfs"
	"github.com/urfave/cli/v3"
)

const (
	rootFlag = "root"
	jsonFlag = "json"

	assetsDirName = "assets"
	buildDirName  = "build"
	distDirName   = "dist"
	stateDirName  = ".goldr"

	generatedFileName = "goldr_assets_gen.go"
	stateFileName     = "assets.json"

	stateVersion = "v0"
	urlPrefix    = "/assets/"
	hashLength   = 8
)

type options struct {
	root string
	json bool
}

type assetPaths struct {
	root          string
	assetsDir     string
	buildDir      string
	distDir       string
	generatedFile string
	stateFile     string
}

type manifestAsset struct {
	Name      string
	Path      string
	Source    string
	Dist      string
	Hash      string
	Size      int64
	Content   []byte
	DistRel   string
	SourceAbs string
	DistAbs   string
}

type stateFile struct {
	Version string       `json:"version"`
	Managed []stateAsset `json:"managed"`
}

type stateAsset struct {
	Logical string `json:"logical"`
	Dist    string `json:"dist"`
	Hash    string `json:"hash"`
}

// Command returns the goldr assets command group.
func Command() *cli.Command {
	return &cli.Command{
		Name:        "assets",
		Usage:       "fingerprint final static assets",
		UsageText:   "goldr assets <command> [options]",
		Description: assetsDescription,
		Commands: []*cli.Command{
			distCommand(),
			checkCommand(),
			cleanCommand(),
			listCommand(),
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return cli.ShowSubcommandHelp(cmd)
		},
	}
}

const assetsDescription = `Fingerprints final browser-ready files only:
  assets/build -> assets/dist
  assets/goldr_assets_gen.go

Goldr does not compile Tailwind, run npm, bundle JavaScript, minify files, optimize images, upload to a CDN, or register static handlers.

Use "go tool goldr assets dist" to write fingerprinted files, then "go tool goldr assets check" in CI.`

func distCommand() *cli.Command {
	return &cli.Command{
		Name:        "dist",
		Usage:       "build fingerprinted asset distribution",
		UsageText:   "goldr assets dist [--root <dir>]",
		Description: assetsDistDescription,
		Flags:       []cli.Flag{rootStringFlag()},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return runDist(options{root: cmd.String(rootFlag)})
		},
	}
}

const assetsDistDescription = `Reads final files from assets/build, copies them to assets/dist with content hashes in their filenames, and writes assets/goldr_assets_gen.go.

This is a final safe-cache step. It does not run asset compilers, bundlers, minifiers, or deployment tools.`

func checkCommand() *cli.Command {
	return &cli.Command{
		Name:        "check",
		Usage:       "check fingerprinted asset distribution",
		UsageText:   "goldr assets check [--root <dir>]",
		Description: assetsCheckDescription,
		Flags:       []cli.Flag{rootStringFlag()},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return runCheck(options{root: cmd.String(rootFlag)})
		},
	}
}

const assetsCheckDescription = `Read-only verification for fingerprinted assets.

Fails if assets/dist, assets/goldr_assets_gen.go, or assets/.goldr/assets.json is missing or stale for the current files in assets/build.`

func cleanCommand() *cli.Command {
	return &cli.Command{
		Name:        "clean",
		Usage:       "remove stale goldr-managed asset files",
		UsageText:   "goldr assets clean [--root <dir>]",
		Description: assetsCleanDescription,
		Flags:       []cli.Flag{rootStringFlag()},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return runClean(options{root: cmd.String(rootFlag)})
		},
	}
}

const assetsCleanDescription = `Removes stale files that goldr can prove it manages from assets/dist.

Clean is fail-closed: it uses goldr asset state and does not delete arbitrary files from assets/dist.`

func listCommand() *cli.Command {
	return &cli.Command{
		Name:        "list",
		Usage:       "list fingerprinted assets",
		UsageText:   "goldr assets list [--root <dir>] [--json]",
		Description: assetsListDescription,
		Flags: []cli.Flag{
			rootStringFlag(),
			&cli.BoolFlag{
				Name:        jsonFlag,
				Usage:       "print JSON output",
				HideDefault: true,
			},
		},
		Action: func(_ context.Context, cmd *cli.Command) error {
			return runList(options{
				root: cmd.String(rootFlag),
				json: cmd.Bool(jsonFlag),
			}, cmd.Root().Writer)
		},
	}
}

const assetsListDescription = `Lists the manifest goldr would generate from assets/build.

Use --json when scripts or agents need stable machine-readable asset metadata.`

func rootStringFlag() *cli.StringFlag {
	return &cli.StringFlag{
		Name:        rootFlag,
		Value:       ".",
		Usage:       "app root directory",
		HideDefault: false,
	}
}

func runDist(options options) error {
	paths, records, err := buildAssetManifest(options.root)
	if err != nil {
		return fmt.Errorf("goldr assets dist: %w", err)
	}

	if err := os.MkdirAll(paths.distDir, 0755); err != nil {
		return fmt.Errorf("goldr assets dist: %w", err)
	}
	for _, record := range records {
		if err := writeFileIfChanged(record.DistAbs, record.Content); err != nil {
			return fmt.Errorf("goldr assets dist: %w", err)
		}
	}

	source, err := generateAssetsSource(records)
	if err != nil {
		return fmt.Errorf("goldr assets dist: %w", err)
	}
	if err := writeFileIfChanged(paths.generatedFile, source); err != nil {
		return fmt.Errorf("goldr assets dist: %w", err)
	}

	state, err := mergedState(paths, records)
	if err != nil {
		return fmt.Errorf("goldr assets dist: %w", err)
	}
	if err := writeStateFile(paths.stateFile, state); err != nil {
		return fmt.Errorf("goldr assets dist: %w", err)
	}
	return nil
}

func runCheck(options options) error {
	paths, records, err := buildAssetManifest(options.root)
	if err != nil {
		return fmt.Errorf("goldr assets check: %w", err)
	}
	if err := appfs.RequireDir(paths.distDir); err != nil {
		return fmt.Errorf("goldr assets check: %w", err)
	}

	var stale []string
	for _, record := range records {
		existing, err := os.ReadFile(record.DistAbs)
		switch {
		case errors.Is(err, os.ErrNotExist):
			stale = append(stale, fmt.Sprintf("%s is missing", record.Dist))
		case err != nil:
			return fmt.Errorf("goldr assets check: %w", err)
		case !bytes.Equal(existing, record.Content):
			stale = append(stale, fmt.Sprintf("%s is stale", record.Dist))
		}
	}

	source, err := generateAssetsSource(records)
	if err != nil {
		return fmt.Errorf("goldr assets check: %w", err)
	}
	if err := checkFile(paths.generatedFile, source, &stale); err != nil {
		return fmt.Errorf("goldr assets check: %w", err)
	}
	if err := checkStateFile(paths, records, &stale); err != nil {
		return fmt.Errorf("goldr assets check: %w", err)
	}
	if len(stale) > 0 {
		return fmt.Errorf("goldr assets check: %s", strings.Join(stale, "\n"))
	}
	return nil
}

func runClean(options options) error {
	paths, records, err := buildAssetManifest(options.root)
	if err != nil {
		return fmt.Errorf("goldr assets clean: %w", err)
	}
	state, err := readStateFile(paths)
	if err != nil {
		return fmt.Errorf("goldr assets clean: %w", err)
	}

	current := stateAssetsByDist(currentStateAssets(records))
	for _, managed := range state.Managed {
		abs, err := managedDistPath(paths, managed)
		if err != nil {
			return fmt.Errorf("goldr assets clean: %w", err)
		}
		if _, ok := current[managed.Dist]; ok {
			continue
		}
		info, err := os.Lstat(abs)
		switch {
		case errors.Is(err, os.ErrNotExist):
			continue
		case err != nil:
			return fmt.Errorf("goldr assets clean: %w", err)
		case !info.Mode().IsRegular():
			return fmt.Errorf("goldr assets clean: %s is not a regular file", managed.Dist)
		}
		if err := os.Remove(abs); err != nil {
			return fmt.Errorf("goldr assets clean: %w", err)
		}
	}

	if err := writeStateFile(paths.stateFile, stateFile{
		Version: stateVersion,
		Managed: currentStateAssets(records),
	}); err != nil {
		return fmt.Errorf("goldr assets clean: %w", err)
	}
	return nil
}

func runList(options options, writer io.Writer) error {
	_, records, err := buildAssetManifest(options.root)
	if err != nil {
		return fmt.Errorf("goldr assets list: %w", err)
	}
	if options.json {
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(listRows(records)); err != nil {
			return fmt.Errorf("goldr assets list: %w", err)
		}
		return nil
	}
	if err := renderListTable(writer, records); err != nil {
		return fmt.Errorf("goldr assets list: %w", err)
	}
	return nil
}

func buildAssetManifest(root string) (assetPaths, []manifestAsset, error) {
	paths, err := assetPathsForRoot(root)
	if err != nil {
		return assetPaths{}, nil, err
	}
	records, err := scanBuildAssets(paths)
	if err != nil {
		return assetPaths{}, nil, err
	}
	return paths, records, nil
}

func assetPathsForRoot(root string) (assetPaths, error) {
	appRoot, err := appfs.ResolveExistingDir(root)
	if err != nil {
		return assetPaths{}, fmt.Errorf("resolve --root %q: %w", root, err)
	}
	assetsDir := filepath.Join(appRoot, assetsDirName)
	return assetPaths{
		root:          appRoot,
		assetsDir:     assetsDir,
		buildDir:      filepath.Join(assetsDir, buildDirName),
		distDir:       filepath.Join(assetsDir, distDirName),
		generatedFile: filepath.Join(assetsDir, generatedFileName),
		stateFile:     filepath.Join(assetsDir, stateDirName, stateFileName),
	}, nil
}

func scanBuildAssets(paths assetPaths) ([]manifestAsset, error) {
	if err := appfs.RequireDir(paths.buildDir); err != nil {
		return nil, err
	}

	var records []manifestAsset
	err := filepath.WalkDir(paths.buildDir, func(name string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if name == paths.buildDir {
			return nil
		}
		rel, err := filepath.Rel(paths.buildDir, name)
		if err != nil {
			return err
		}
		logical := filepath.ToSlash(rel)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symlink; asset inputs must be regular files", filepath.Join("assets", "build", filepath.FromSlash(logical)))
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s is not a regular file", filepath.Join("assets", "build", filepath.FromSlash(logical)))
		}
		content, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		hash := shortHash(content)
		distRel := fingerprintedPath(logical, hash)
		distAbs := filepath.Join(paths.distDir, filepath.FromSlash(distRel))
		sourceRel, err := rootRelativePath(paths.root, name)
		if err != nil {
			return err
		}
		distPath, err := rootRelativePath(paths.root, distAbs)
		if err != nil {
			return err
		}
		records = append(records, manifestAsset{
			Name:      logical,
			Path:      assetURLPath(distRel),
			Source:    sourceRel,
			Dist:      distPath,
			Hash:      hash,
			Size:      info.Size(),
			Content:   content,
			DistRel:   distRel,
			SourceAbs: name,
			DistAbs:   distAbs,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	slices.SortFunc(records, func(a, b manifestAsset) int {
		return strings.Compare(a.Name, b.Name)
	})
	for index := 1; index < len(records); index++ {
		if records[index-1].Name == records[index].Name {
			return nil, fmt.Errorf("logical asset %q is ambiguous", records[index].Name)
		}
	}
	return records, nil
}

func assetURLPath(distRel string) string {
	parts := strings.Split(distRel, "/")
	for index, part := range parts {
		parts[index] = url.PathEscape(part)
	}
	return urlPrefix + strings.Join(parts, "/")
}

func shortHash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])[:hashLength]
}

func fingerprintedPath(logical string, hash string) string {
	dir, file := path.Split(logical)
	ext := path.Ext(file)
	if ext == "" {
		return path.Join(dir, file+"."+hash)
	}
	stem := strings.TrimSuffix(file, ext)
	return path.Join(dir, stem+"."+hash+ext)
}

func rootRelativePath(root string, name string) (string, error) {
	rel, err := filepath.Rel(root, name)
	if err != nil {
		return "", err
	}
	return filepath.ToSlash(rel), nil
}

func generateAssetsSource(records []manifestAsset) ([]byte, error) {
	var builder strings.Builder
	builder.WriteString("// Code generated by goldr assets dist. DO NOT EDIT.\n")
	builder.WriteString("package assets\n\n")
	if len(records) > 0 {
		builder.WriteString("import (\n")
		builder.WriteString("\t\"embed\"\n")
		builder.WriteString("\t\"io/fs\"\n")
		builder.WriteString("\t\"maps\"\n")
		builder.WriteString(")\n\n")
		builder.WriteString("//go:embed dist/*\n")
		builder.WriteString("var embedded embed.FS\n\n")
	} else {
		builder.WriteString("import (\n")
		builder.WriteString("\t\"io/fs\"\n")
		builder.WriteString("\t\"maps\"\n")
		builder.WriteString(")\n\n")
	}
	builder.WriteString("type Asset struct {\n")
	builder.WriteString("\tName string\n")
	builder.WriteString("\tPath string\n")
	builder.WriteString("\tHash string\n")
	builder.WriteString("\tSize int64\n")
	builder.WriteString("}\n\n")
	builder.WriteString("var manifest = map[string]Asset{\n")
	for _, record := range records {
		fmt.Fprintf(&builder, "\t%q: {\n", record.Name)
		fmt.Fprintf(&builder, "\t\tName: %q,\n", record.Name)
		fmt.Fprintf(&builder, "\t\tPath: %q,\n", record.Path)
		fmt.Fprintf(&builder, "\t\tHash: %q,\n", record.Hash)
		fmt.Fprintf(&builder, "\t\tSize: %d,\n", record.Size)
		builder.WriteString("\t},\n")
	}
	builder.WriteString("}\n\n")
	builder.WriteString("func Path(name string) string {\n")
	builder.WriteString("\tpath, ok := Lookup(name)\n")
	builder.WriteString("\tif !ok {\n")
	builder.WriteString("\t\tpanic(\"unknown asset: \" + name)\n")
	builder.WriteString("\t}\n")
	builder.WriteString("\treturn path\n")
	builder.WriteString("}\n\n")
	builder.WriteString("func Lookup(name string) (string, bool) {\n")
	builder.WriteString("\tasset, ok := manifest[name]\n")
	builder.WriteString("\tif !ok {\n")
	builder.WriteString("\t\treturn \"\", false\n")
	builder.WriteString("\t}\n")
	builder.WriteString("\treturn asset.Path, true\n")
	builder.WriteString("}\n\n")
	builder.WriteString("func Manifest() map[string]Asset {\n")
	builder.WriteString("\tout := make(map[string]Asset, len(manifest))\n")
	builder.WriteString("\tmaps.Copy(out, manifest)\n")
	builder.WriteString("\treturn out\n")
	builder.WriteString("}\n\n")
	if len(records) > 0 {
		builder.WriteString("func FS() fs.FS {\n")
		builder.WriteString("\tfsys, err := fs.Sub(embedded, \"dist\")\n")
		builder.WriteString("\tif err != nil {\n")
		builder.WriteString("\t\tpanic(err)\n")
		builder.WriteString("\t}\n")
		builder.WriteString("\treturn fsys\n")
		builder.WriteString("}\n")
	} else {
		builder.WriteString("type emptyFS struct{}\n\n")
		builder.WriteString("func (emptyFS) Open(string) (fs.File, error) {\n")
		builder.WriteString("\treturn nil, fs.ErrNotExist\n")
		builder.WriteString("}\n\n")
		builder.WriteString("func FS() fs.FS {\n")
		builder.WriteString("\treturn emptyFS{}\n")
		builder.WriteString("}\n")
	}

	source, err := format.Source([]byte(builder.String()))
	if err != nil {
		return nil, err
	}
	return source, nil
}

func writeFileIfChanged(name string, content []byte) error {
	existing, err := os.ReadFile(name)
	if err == nil && bytes.Equal(existing, content) {
		return nil
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		return err
	}
	return os.WriteFile(name, content, 0644)
}

func checkFile(name string, content []byte, stale *[]string) error {
	existing, err := os.ReadFile(name)
	switch {
	case errors.Is(err, os.ErrNotExist):
		*stale = append(*stale, fmt.Sprintf("%s is missing", name))
	case err != nil:
		return err
	case !bytes.Equal(existing, content):
		*stale = append(*stale, fmt.Sprintf("%s is stale", name))
	}
	return nil
}

func mergedState(paths assetPaths, records []manifestAsset) (stateFile, error) {
	merged := stateFile{
		Version: stateVersion,
	}
	existing, err := readStateFile(paths)
	if err == nil {
		merged.Managed = append(merged.Managed, existing.Managed...)
	} else if !errors.Is(err, os.ErrNotExist) {
		return stateFile{}, err
	}
	for _, managed := range merged.Managed {
		if _, err := managedDistPath(paths, managed); err != nil {
			return stateFile{}, err
		}
	}

	byDist := stateAssetsByDist(merged.Managed)
	for _, managed := range currentStateAssets(records) {
		byDist[managed.Dist] = managed
	}
	merged.Managed = sortedStateAssets(byDist)
	return merged, nil
}

func readStateFile(paths assetPaths) (stateFile, error) {
	content, err := os.ReadFile(paths.stateFile)
	if err != nil {
		return stateFile{}, err
	}
	var state stateFile
	if err := json.Unmarshal(content, &state); err != nil {
		return stateFile{}, err
	}
	if state.Version != stateVersion {
		return stateFile{}, fmt.Errorf("%s has unsupported version %q", paths.stateFile, state.Version)
	}
	if state.Managed == nil {
		state.Managed = []stateAsset{}
	}
	for _, managed := range state.Managed {
		if managed.Logical == "" || managed.Dist == "" || managed.Hash == "" {
			return stateFile{}, fmt.Errorf("%s contains incomplete managed asset", paths.stateFile)
		}
	}
	return state, nil
}

func writeStateFile(name string, state stateFile) error {
	state.Managed = sortedStateAssets(stateAssetsByDist(state.Managed))
	content, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	return writeFileIfChanged(name, content)
}

func checkStateFile(paths assetPaths, records []manifestAsset, stale *[]string) error {
	state, err := readStateFile(paths)
	switch {
	case errors.Is(err, os.ErrNotExist):
		*stale = append(*stale, fmt.Sprintf("%s is missing", paths.stateFile))
		return nil
	case err != nil:
		return err
	}
	managed := stateAssetsByDist(state.Managed)
	for _, current := range currentStateAssets(records) {
		got, ok := managed[current.Dist]
		if !ok || got.Logical != current.Logical || got.Hash != current.Hash {
			*stale = append(*stale, fmt.Sprintf("%s is stale", paths.stateFile))
			return nil
		}
	}
	for _, entry := range state.Managed {
		if _, err := managedDistPath(paths, entry); err != nil {
			return err
		}
	}
	return nil
}

func currentStateAssets(records []manifestAsset) []stateAsset {
	managed := make([]stateAsset, 0, len(records))
	for _, record := range records {
		managed = append(managed, stateAsset{
			Logical: record.Name,
			Dist:    record.Dist,
			Hash:    record.Hash,
		})
	}
	return managed
}

func stateAssetsByDist(entries []stateAsset) map[string]stateAsset {
	byDist := make(map[string]stateAsset, len(entries))
	for _, entry := range entries {
		byDist[entry.Dist] = entry
	}
	return byDist
}

func sortedStateAssets(byDist map[string]stateAsset) []stateAsset {
	entries := make([]stateAsset, 0, len(byDist))
	for _, entry := range byDist {
		entries = append(entries, entry)
	}
	slices.SortFunc(entries, func(a, b stateAsset) int {
		return strings.Compare(a.Dist, b.Dist)
	})
	return entries
}

func managedDistPath(paths assetPaths, managed stateAsset) (string, error) {
	if filepath.IsAbs(managed.Dist) {
		return "", fmt.Errorf("managed asset %q must be relative", managed.Dist)
	}
	clean := path.Clean(managed.Dist)
	if clean != managed.Dist || !strings.HasPrefix(clean, "assets/dist/") {
		return "", fmt.Errorf("managed asset %q is outside assets/dist", managed.Dist)
	}
	abs := filepath.Join(paths.root, filepath.FromSlash(clean))
	rel, err := filepath.Rel(paths.distDir, abs)
	if err != nil {
		return "", err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("managed asset %q is outside assets/dist", managed.Dist)
	}
	return abs, nil
}

type listRow struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	Source string `json:"source"`
	Dist   string `json:"dist"`
	Hash   string `json:"hash"`
	Size   int64  `json:"size"`
}

func listRows(records []manifestAsset) []listRow {
	rows := make([]listRow, 0, len(records))
	for _, record := range records {
		rows = append(rows, listRow{
			Name:   record.Name,
			Path:   record.Path,
			Source: record.Source,
			Dist:   record.Dist,
			Hash:   record.Hash,
			Size:   record.Size,
		})
	}
	return rows
}

func renderListTable(writer io.Writer, records []manifestAsset) error {
	table := tabwriter.NewWriter(writer, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "Logical asset\tDist path\tSize"); err != nil {
		return err
	}
	for _, record := range records {
		if _, err := fmt.Fprintf(table, "%s\t%s\t%d\n", record.Name, record.Path, record.Size); err != nil {
			return err
		}
	}
	return table.Flush()
}
