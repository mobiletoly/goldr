// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldrcli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/mobiletoly/goldr/internal/goldrcli/appfs"
	"github.com/urfave/cli/v3"
)

const (
	devRootFlag      = "root"
	devAppURLFlag    = "app-url"
	devProxyAddrFlag = "proxy-addr"
	devCommandFlag   = "cmd"

	defaultDevAppURL    = "http://127.0.0.1:8080"
	defaultDevProxyAddr = "127.0.0.1:7331"
	defaultDevCommand   = "go run ."
)

type devOptions struct {
	root      string
	appURL    string
	proxyAddr string
	command   string
}

type devConfig struct {
	root            string
	appURL          string
	proxyBind       string
	proxyPort       int
	command         string
	goldrExecutable string
	wrapperPath     string
}

func devCommand() *cli.Command {
	return &cli.Command{
		Name:        "dev",
		Usage:       "run live reload for a goldr app",
		UsageText:   "goldr dev [--root <dir>] [--app-url <url>] [--proxy-addr <host:port>] [--cmd <command>]",
		Description: devDescription,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        devRootFlag,
				Value:       ".",
				Usage:       "app root directory",
				HideDefault: false,
			},
			&cli.StringFlag{
				Name:        devAppURLFlag,
				Value:       defaultDevAppURL,
				Usage:       "app server URL that proxy forwards to",
				HideDefault: false,
			},
			&cli.StringFlag{
				Name:        devProxyAddrFlag,
				Value:       defaultDevProxyAddr,
				Usage:       "proxy listen address",
				HideDefault: false,
			},
			&cli.StringFlag{
				Name:        devCommandFlag,
				Value:       defaultDevCommand,
				Usage:       "app command executed from the app root",
				HideDefault: false,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return runDev(ctx, devOptions{
				root:      cmd.String(devRootFlag),
				appURL:    cmd.String(devAppURLFlag),
				proxyAddr: cmd.String(devProxyAddrFlag),
				command:   cmd.String(devCommandFlag),
			}, cmd.Root().Writer, cmd.Root().ErrWriter)
		},
	}
}

const devDescription = `Runs a local development loop using templ watch mode.

goldr dev keeps development on the production asset path: templates keep using assets.Path, apps keep serving assets.FS, and changes under assets/build run goldr assets dist before the app restarts.

Run app-owned tools such as Tailwind separately so they write final browser-ready files into assets/build. goldr dev watches assets/build, not assets/src.`

func runDev(ctx context.Context, options devOptions, stdout, stderr io.Writer) error {
	config, err := resolveDevConfig(ctx, options)
	if err != nil {
		return fmt.Errorf("goldr dev: %w", err)
	}
	defer func() {
		_ = os.Remove(config.wrapperPath)
	}()

	if err := checkTemplTool(ctx, config.root); err != nil {
		return fmt.Errorf("goldr dev: %w", err)
	}

	if _, err := fmt.Fprintf(stdout, "goldr dev proxy listening on http://%s\n", net.JoinHostPort(config.proxyBind, strconv.Itoa(config.proxyPort))); err != nil {
		return fmt.Errorf("write proxy URL: %w", err)
	}

	command := exec.CommandContext(ctx, "go", templArgs(config)...)
	command.Dir = config.root
	command.Stdin = os.Stdin
	command.Stdout = stdout
	command.Stderr = stderr
	command.Env = os.Environ()
	if err := command.Run(); err != nil {
		return fmt.Errorf("templ live reload failed: %w", err)
	}
	return nil
}

func resolveDevConfig(ctx context.Context, options devOptions) (devConfig, error) {
	appRoot, err := appfs.ResolveExistingDir(options.root)
	if err != nil {
		return devConfig{}, fmt.Errorf("resolve --root %q: %w", options.root, err)
	}
	if _, err := appPathsForResolvedRoot(ctx, appRoot); err != nil {
		return devConfig{}, err
	}
	if err := appfs.RequireDir(appfs.RoutesDir(appRoot)); err != nil {
		return devConfig{}, err
	}

	appURL, err := validateDevAppURL(options.appURL)
	if err != nil {
		return devConfig{}, err
	}
	proxyBind, proxyPort, err := parseDevProxyAddr(options.proxyAddr)
	if err != nil {
		return devConfig{}, err
	}
	if strings.TrimSpace(options.command) == "" {
		return devConfig{}, errors.New("--cmd must not be empty")
	}
	goldrExecutable, err := os.Executable()
	if err != nil {
		return devConfig{}, fmt.Errorf("resolve current executable: %w", err)
	}

	config := devConfig{
		root:            appRoot,
		appURL:          appURL,
		proxyBind:       proxyBind,
		proxyPort:       proxyPort,
		command:         options.command,
		goldrExecutable: goldrExecutable,
	}
	wrapperPath, err := writeDevWrapper(config)
	if err != nil {
		return devConfig{}, err
	}
	config.wrapperPath = wrapperPath
	return config, nil
}

func validateDevAppURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid --app-url %q: %w", raw, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("--app-url must use http or https")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("--app-url must include a host")
	}
	return parsed.String(), nil
}

func parseDevProxyAddr(raw string) (string, int, error) {
	host, portText, err := net.SplitHostPort(raw)
	if err != nil {
		return "", 0, fmt.Errorf("invalid --proxy-addr %q: %w", raw, err)
	}
	if host == "" {
		return "", 0, fmt.Errorf("--proxy-addr must include a host")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port <= 0 || port > 65535 {
		return "", 0, fmt.Errorf("--proxy-addr port must be between 1 and 65535")
	}
	return host, port, nil
}

func checkTemplTool(ctx context.Context, root string) error {
	command := exec.CommandContext(ctx, "go", "tool", "templ", "generate", "--help")
	command.Dir = root
	command.Stdout = io.Discard
	command.Stderr = io.Discard
	if err := command.Run(); err != nil {
		return fmt.Errorf("go tool templ is not available; add it with: go get -tool github.com/a-h/templ/cmd/templ@v0.3.1020")
	}
	return nil
}

func templArgs(config devConfig) []string {
	return []string{
		"tool",
		"templ",
		"generate",
		"-path", config.root,
		"-watch",
		"-watch-pattern", devWatchPattern(),
		"-ignore-pattern", devIgnorePattern(),
		"-proxy", config.appURL,
		"-proxybind", config.proxyBind,
		"-proxyport", strconv.Itoa(config.proxyPort),
		"-open-browser=false",
		"-cmd", config.wrapperPath,
	}
}

func devWatchPattern() string {
	separator := `[\\/]`
	return strings.Join([]string{
		`(.+\.go$)`,
		`(.+\.templ$)`,
		`(.+` + separator + `assets` + separator + `build` + separator + `.+)`,
	}, "|")
}

func devIgnorePattern() string {
	separator := `[\\/]`
	return strings.Join([]string{
		`(.+` + separator + `app` + separator + `routes` + separator + `goldr_gen\.go$)`,
		`(.+` + separator + `app` + separator + `urls` + separator + `goldr_gen\.go$)`,
		`(.+` + separator + `assets` + separator + `goldr_assets_gen\.go$)`,
		`(.+` + separator + `assets` + separator + `dist(` + separator + `.*)?$)`,
		`(.+` + separator + `assets` + separator + `\.goldr(` + separator + `.*)?$)`,
	}, "|")
}

func writeDevWrapper(config devConfig) (string, error) {
	return writeDevWrapperInTempDirs(config, devWrapperTempDirCandidates())
}

func writeDevWrapperInTempDirs(config devConfig, candidates []string) (string, error) {
	tempDir, err := selectDevWrapperTempDir(candidates)
	if err != nil {
		return "", err
	}
	if runtime.GOOS == "windows" {
		return writeWindowsDevWrapper(config, tempDir)
	}
	return writeUnixDevWrapper(config, tempDir)
}

func devWrapperTempDirCandidates() []string {
	candidates := []string{os.TempDir()}
	if runtime.GOOS == "windows" {
		if systemDrive := os.Getenv("SystemDrive"); systemDrive != "" {
			candidates = append(candidates, filepath.Join(systemDrive+`\`, "Temp"))
		}
	} else {
		candidates = append(candidates, "/tmp", "/var/tmp")
	}
	return candidates
}

func selectDevWrapperTempDir(candidates []string) (string, error) {
	seen := make(map[string]bool)
	for _, candidate := range candidates {
		if candidate == "" || devPathHasWhitespace(candidate) || seen[candidate] {
			continue
		}
		seen[candidate] = true
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, nil
		}
	}
	return "", errors.New("could not find a space-free temp directory for the dev wrapper; set TMPDIR to a path without whitespace")
}

func devPathHasWhitespace(path string) bool {
	return strings.ContainsAny(path, " \t\n\r\v\f")
}

func writeUnixDevWrapper(config devConfig, tempDir string) (string, error) {
	file, err := os.CreateTemp(tempDir, "goldr-dev-*.sh")
	if err != nil {
		return "", fmt.Errorf("create dev wrapper: %w", err)
	}
	path := file.Name()
	if err := file.Chmod(0755); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("chmod dev wrapper: %w", err)
	}
	content := strings.Join([]string{
		"#!/bin/sh",
		"set -eu",
		shellQuote(config.goldrExecutable) + " generate --root " + shellQuote(config.root),
		"if [ -d " + shellQuote(filepath.Join(config.root, "assets", "build")) + " ]; then",
		"  " + shellQuote(config.goldrExecutable) + " assets dist --root " + shellQuote(config.root),
		"fi",
		"cd " + shellQuote(config.root),
		"exec /bin/sh -c " + shellQuote(config.command),
		"",
	}, "\n")
	if _, err := file.WriteString(content); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("write dev wrapper: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("close dev wrapper: %w", err)
	}
	return path, nil
}

func writeWindowsDevWrapper(config devConfig, tempDir string) (string, error) {
	file, err := os.CreateTemp(tempDir, "goldr-dev-*.cmd")
	if err != nil {
		return "", fmt.Errorf("create dev wrapper: %w", err)
	}
	path := file.Name()
	content := strings.Join([]string{
		"@echo off",
		"setlocal",
		windowsQuote(config.goldrExecutable) + " generate --root " + windowsQuote(config.root) + " || exit /b %ERRORLEVEL%",
		"if exist " + windowsQuote(filepath.Join(config.root, "assets", "build")) + " (",
		"  " + windowsQuote(config.goldrExecutable) + " assets dist --root " + windowsQuote(config.root) + " || exit /b %ERRORLEVEL%",
		")",
		"cd /d " + windowsQuote(config.root) + " || exit /b %ERRORLEVEL%",
		"cmd /S /C " + windowsQuote(config.command),
		"",
	}, "\r\n")
	if _, err := file.WriteString(content); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("write dev wrapper: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("close dev wrapper: %w", err)
	}
	return path, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func windowsQuote(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}
