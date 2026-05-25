// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package templtool

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const InstallCommand = "go get -tool github.com/a-h/templ/cmd/templ@v0.3.1020"

func Require(ctx context.Context, root string) error {
	command := exec.CommandContext(ctx, "go", "tool", "templ", "--help")
	command.Dir = root
	if err := command.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return errors.New("go executable not found")
		}
		return fmt.Errorf("go tool templ is not available; add it with: %s", InstallCommand)
	}
	return nil
}

func Generate(ctx context.Context, root string) error {
	if err := Require(ctx, root); err != nil {
		return err
	}

	command := exec.CommandContext(ctx, "go", "tool", "templ", "generate", "-path", ".")
	command.Dir = root
	output, err := command.CombinedOutput()
	if err == nil {
		return nil
	}

	var message strings.Builder
	message.WriteString("templ generation failed")
	if trimmed := strings.TrimSpace(string(output)); trimmed != "" {
		message.WriteString("\n")
		message.WriteString(trimmed)
	}
	return errors.New(message.String())
}

func GenerateCheck(ctx context.Context, root string) error {
	if err := Require(ctx, root); err != nil {
		return err
	}

	command := exec.CommandContext(ctx, "go", "tool", "templ", "generate", "-check", "-path", ".")
	command.Dir = root
	output, err := command.CombinedOutput()
	if err == nil {
		return nil
	}

	var message strings.Builder
	message.WriteString("templ generated files are not up to date; run go tool goldr generate")
	if trimmed := strings.TrimSpace(string(output)); trimmed != "" {
		message.WriteString("\n")
		message.WriteString(trimmed)
	}
	return errors.New(message.String())
}
