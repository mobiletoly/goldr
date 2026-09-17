// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package dev

import (
	"os"
	"os/exec"
	"strconv"
	"time"
)

func devTerminationSignals() []os.Signal {
	return []os.Signal{os.Interrupt}
}

func configureDevCommand(command *exec.Cmd) {
	command.Cancel = func() error {
		return exec.Command("TASKKILL", "/T", "/F", "/PID", strconv.Itoa(command.Process.Pid)).Run()
	}
	command.WaitDelay = 3 * time.Second
}

func stopDevCommand(string) error {
	return nil
}
