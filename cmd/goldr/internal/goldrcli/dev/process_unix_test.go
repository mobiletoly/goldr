// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package dev

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestConfigureDevCommandStopsDescendantProcessesOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDevCommandSignalHelper$")
	command.Env = append(os.Environ(), "GOLDR_TEST_DEV_SIGNAL_HELPER=1")
	configureDevCommand(command)
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	})

	scanner := bufio.NewScanner(stdout)
	if !scanner.Scan() {
		t.Fatalf("read descendant PID: %v", scanner.Err())
	}
	descendantPID, err := strconv.Atoi(scanner.Text())
	if err != nil {
		t.Fatalf("parse descendant PID: %v", err)
	}

	cancel()
	_ = command.Wait()
	deadline := time.Now().Add(3 * time.Second)
	for processExists(descendantPID) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if processExists(descendantPID) {
		t.Fatalf("descendant process %d survived command cancellation", descendantPID)
	}
}

func TestStopDevCommandKillsRecordedProcessGroup(t *testing.T) {
	command := exec.Command("/bin/sleep", "30")
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		_, _ = command.Process.Wait()
	})
	wrapperPath := filepath.Join(t.TempDir(), "wrapper")
	if err := os.WriteFile(devCommandPIDPath(wrapperPath), []byte(strconv.Itoa(command.Process.Pid)+"\n"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := stopDevCommand(wrapperPath); err != nil {
		t.Fatalf("stopDevCommand() error = %v", err)
	}
	_, _ = command.Process.Wait()
	deadline := time.Now().Add(3 * time.Second)
	for processExists(command.Process.Pid) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if processExists(command.Process.Pid) {
		t.Fatalf("process group %d survived cleanup", command.Process.Pid)
	}
}

func TestDevTerminationSignalsIncludeSIGTERM(t *testing.T) {
	for _, signal := range devTerminationSignals() {
		if signal == syscall.SIGTERM {
			return
		}
	}
	t.Fatalf("devTerminationSignals() = %v, want SIGTERM", devTerminationSignals())
}

func TestRunDevChecksReloadDirectoriesBeforeTempl(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read mode-000 directories")
	}
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/devapp\n\ngo 1.26.3\n")
	writeFile(t, root, "app/routes/page.go", "package routes\n")
	writeFile(t, root, "app/routes/page.templ", "package routes\n\ntempl PageView() {}\n")
	content := filepath.Join(root, "content")
	blocked := filepath.Join(content, "blocked")
	if err := os.MkdirAll(blocked, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(blocked, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(blocked, 0755)

	fakeBin := t.TempDir()
	goLog := filepath.Join(t.TempDir(), "go.log")
	writeFakeStartupGo(t, fakeBin)
	t.Setenv("PATH", fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GOLDR_TEST_GO_MOD", filepath.Join(root, "go.mod"))
	t.Setenv("GOLDR_TEST_GO_LOG", goLog)

	err := runDev(context.Background(), devOptions{
		root:        root,
		reloadPaths: []string{content},
		appURL:      defaultDevAppURL,
		proxyAddr:   defaultDevProxyAddr,
		command:     defaultDevCommand,
	}, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "reload directory") || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("runDev() error = %v, want inaccessible reload directory error", err)
	}
	log, readErr := os.ReadFile(goLog)
	if readErr != nil {
		t.Fatalf("read fake go log: %v", readErr)
	}
	if strings.Contains(string(log), "tool templ --help") {
		t.Fatalf("go invocations = %q, templ lookup must not run before reload directory validation", log)
	}
}

func TestRunDevSIGTERMDuringTemplCheckCleansStartupResources(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "go.mod", "module example.com/devapp\n\ngo 1.26.3\n")
	writeFile(t, root, "app/routes/page.go", "package routes\n")
	writeFile(t, root, "app/routes/page.templ", "package routes\n\ntempl PageView() {}\n")
	content := filepath.Join(root, "content")
	if err := os.Mkdir(content, 0755); err != nil {
		t.Fatal(err)
	}

	fakeBin := t.TempDir()
	writeFakeStartupGo(t, fakeBin)
	wrapperDir := t.TempDir()
	templPIDPath := filepath.Join(t.TempDir(), "templ.pid")
	command := exec.Command(os.Args[0], "-test.run=^TestRunDevStartupSignalHelper$")
	command.Env = append(os.Environ(),
		"GOLDR_TEST_DEV_STARTUP_SIGNAL_HELPER=1",
		"GOLDR_TEST_DEV_ROOT="+root,
		"GOLDR_TEST_DEV_RELOAD_PATH="+content,
		"GOLDR_TEST_GO_MOD="+filepath.Join(root, "go.mod"),
		"GOLDR_TEST_TEMPL_PID="+templPIDPath,
		"PATH="+fakeBin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"TMPDIR="+wrapperDir,
	)
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- command.Wait()
	}()
	finished := false
	t.Cleanup(func() {
		if !finished {
			_ = command.Process.Kill()
			<-done
		}
	})

	templPID := waitForPIDFile(t, templPIDPath)
	t.Cleanup(func() {
		_ = syscall.Kill(templPID, syscall.SIGKILL)
	})
	if err := command.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("signal helper: %v", err)
	}

	var waitErr error
	select {
	case waitErr = <-done:
		finished = true
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for reload-only startup to stop")
	}
	wrapperPaths, err := filepath.Glob(filepath.Join(wrapperDir, "goldr-dev-*.sh"))
	if err != nil {
		t.Fatal(err)
	}
	availabilityCheckAlive := processExists(templPID)
	if waitErr != nil || len(wrapperPaths) != 0 || availabilityCheckAlive {
		t.Fatalf("SIGTERM cleanup: wait error = %v, wrappers = %v, templ check alive = %t", waitErr, wrapperPaths, availabilityCheckAlive)
	}
}

func TestRunDevStartupSignalHelper(t *testing.T) {
	if os.Getenv("GOLDR_TEST_DEV_STARTUP_SIGNAL_HELPER") != "1" {
		return
	}
	if err := runDev(context.Background(), devOptions{
		root:        os.Getenv("GOLDR_TEST_DEV_ROOT"),
		reloadPaths: []string{os.Getenv("GOLDR_TEST_DEV_RELOAD_PATH")},
		appURL:      defaultDevAppURL,
		proxyAddr:   defaultDevProxyAddr,
		command:     defaultDevCommand,
	}, io.Discard, io.Discard); err != nil {
		t.Fatalf("runDev() error after startup cancellation = %v", err)
	}
}

func writeFakeStartupGo(t *testing.T, binDir string) {
	t.Helper()
	path := filepath.Join(binDir, "go")
	script := `#!/bin/sh
if [ -n "${GOLDR_TEST_GO_LOG:-}" ]; then
  printf '%s\n' "$*" >> "$GOLDR_TEST_GO_LOG"
fi
if [ "$1" = "env" ] && [ "$2" = "GOMOD" ]; then
  printf '%s\n' "$GOLDR_TEST_GO_MOD"
  exit 0
fi
if [ "$1" = "tool" ] && [ "$2" = "templ" ] && [ "$3" = "--help" ]; then
  if [ -n "${GOLDR_TEST_TEMPL_PID:-}" ]; then
    printf '%s\n' "$$" > "$GOLDR_TEST_TEMPL_PID"
    exec /bin/sleep 30
  fi
  exit 23
fi
exit 24
`
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("write fake go: %v", err)
	}
}

func waitForPIDFile(t *testing.T, path string) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		content, err := os.ReadFile(path)
		if err == nil {
			pid, err := strconv.Atoi(strings.TrimSpace(string(content)))
			if err != nil {
				t.Fatalf("parse PID file %s: %v", path, err)
			}
			return pid
		}
		if !os.IsNotExist(err) {
			t.Fatalf("read PID file %s: %v", path, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for PID file %s", path)
	return 0
}

func TestResolveReloadPathsRejectsNonRegularFile(t *testing.T) {
	root, err := os.MkdirTemp("/tmp", "goldr-reload-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(root)
	})
	path := filepath.Join(root, "content.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Skipf("Unix sockets are unavailable: %v", err)
	}
	defer listener.Close()

	_, err = resolveReloadPaths([]string{path})
	if err == nil || !strings.Contains(err.Error(), "regular file or directory") {
		t.Fatalf("resolveReloadPaths() error = %v, want non-regular path error", err)
	}
}

func TestResolveReloadPathsRejectsInaccessiblePath(t *testing.T) {
	root := t.TempDir()
	blocked := filepath.Join(root, "blocked")
	if err := os.Mkdir(blocked, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(blocked, "page.html")
	if err := os.WriteFile(path, []byte("page"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(blocked, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(blocked, 0700)

	_, err := resolveReloadPaths([]string{path})
	if err == nil {
		t.Skip("filesystem permits access through a mode-000 directory")
	}
	if !strings.Contains(err.Error(), "--reload-path") {
		t.Fatalf("resolveReloadPaths() error = %v, want reload-path context", err)
	}
}

func TestResolveReloadPathsRejectsUnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can read mode-000 files")
	}
	root := t.TempDir()
	path := filepath.Join(root, "page.html")
	if err := os.WriteFile(path, []byte("page"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(path, 0600)

	_, err := resolveReloadPaths([]string{path})
	if err == nil || !strings.Contains(err.Error(), "read --reload-path") {
		t.Fatalf("resolveReloadPaths() error = %v, want unreadable file error", err)
	}
}

func TestDevCommandSignalHelper(t *testing.T) {
	if os.Getenv("GOLDR_TEST_DEV_SIGNAL_HELPER") != "1" {
		return
	}
	child := exec.Command("/bin/sleep", "30")
	if err := child.Start(); err != nil {
		os.Exit(2)
	}
	fmt.Println(child.Process.Pid)
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt, syscall.SIGTERM)
	<-interrupts
	_ = child.Process.Kill()
	_ = child.Wait()
	os.Exit(0)
}

func processExists(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}
