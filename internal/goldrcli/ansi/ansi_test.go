package ansi

import (
	"bytes"
	"os"
	"testing"
)

func TestAnsiStyleWrapsEnabledText(t *testing.T) {
	style := New(true)

	if got, want := style.Green("page"), greenCode+"page"+resetCode; got != want {
		t.Fatalf("green() = %q, want %q", got, want)
	}
	if got := style.Bold(""); got != "" {
		t.Fatalf("bold(empty) = %q, want empty", got)
	}
}

func TestAnsiStyleLeavesDisabledTextPlain(t *testing.T) {
	var style Style

	if got := style.Magenta("action"); got != "action" {
		t.Fatalf("magenta(disabled) = %q, want plain text", got)
	}
}

func TestAnsiStyleForWriterDisablesNonTerminalWriters(t *testing.T) {
	var buffer bytes.Buffer
	if style := ForWriter(&buffer); style.enabled {
		t.Fatal("ForWriter(buffer).enabled = true, want false")
	}

	file, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			t.Fatalf("Close(temp file) error = %v", err)
		}
	}()

	if style := ForWriter(file); style.enabled {
		t.Fatal("ForWriter(regular file).enabled = true, want false")
	}
}

func TestAnsiEnvAllowsColorDisablesOptOuts(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T)
	}{
		{
			name: "no color set empty",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("NO_COLOR", "")
			},
		},
		{
			name: "no color set value",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("NO_COLOR", "1")
			},
		},
		{
			name: "dumb terminal",
			setup: func(t *testing.T) {
				t.Helper()
				t.Setenv("TERM", "dumb")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)
			if envAllowsColor() {
				t.Fatal("envAllowsColor() = true, want false")
			}
		})
	}
}
