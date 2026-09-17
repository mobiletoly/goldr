// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package dev

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func devTerminationSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}

func configureDevCommand(command *exec.Cmd) {
	command.Cancel = func() error {
		return command.Process.Signal(os.Interrupt)
	}
	command.WaitDelay = 3 * time.Second
}

func stopDevCommand(wrapperPath string) error {
	content, err := os.ReadFile(devCommandPIDPath(wrapperPath))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(content)))
	if err != nil || pid <= 1 {
		return fmt.Errorf("invalid app process group in %s", devCommandPIDPath(wrapperPath))
	}
	for _, signal := range []syscall.Signal{syscall.SIGTERM, syscall.SIGKILL} {
		if err := syscall.Kill(-pid, signal); err != nil && !errors.Is(err, syscall.ESRCH) {
			return fmt.Errorf("signal app process group %d: %w", pid, err)
		}
	}
	return nil
}
