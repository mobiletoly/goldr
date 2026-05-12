// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package ansi

import (
	"io"
	"os"
)

const (
	resetCode   = "\x1b[0m"
	boldCode    = "\x1b[1m"
	dimCode     = "\x1b[2m"
	cyanCode    = "\x1b[36m"
	greenCode   = "\x1b[32m"
	yellowCode  = "\x1b[33m"
	magentaCode = "\x1b[35m"
)

type Style struct {
	enabled bool
}

func New(enabled bool) Style {
	return Style{enabled: enabled}
}

func ForWriter(writer io.Writer) Style {
	if !envAllowsColor() {
		return Style{}
	}

	file, ok := writer.(*os.File)
	if !ok {
		return Style{}
	}
	info, err := file.Stat()
	if err != nil {
		return Style{}
	}
	return Style{enabled: info.Mode()&os.ModeCharDevice != 0}
}

func envAllowsColor() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	return os.Getenv("TERM") != "dumb"
}

func (style Style) Bold(text string) string {
	return style.wrap(boldCode, text)
}

func (style Style) Dim(text string) string {
	return style.wrap(dimCode, text)
}

func (style Style) Cyan(text string) string {
	return style.wrap(cyanCode, text)
}

func (style Style) Green(text string) string {
	return style.wrap(greenCode, text)
}

func (style Style) Yellow(text string) string {
	return style.wrap(yellowCode, text)
}

func (style Style) Magenta(text string) string {
	return style.wrap(magentaCode, text)
}

func (style Style) wrap(code string, text string) string {
	if !style.enabled || text == "" {
		return text
	}
	return code + text + resetCode
}
