// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"os"

	"github.com/mobiletoly/goldr/cmd/goldr/internal/goldrcli"
)

const version = "dev"

func main() {
	os.Exit(goldrcli.Run(context.Background(), os.Args, os.Stdout, os.Stderr, version))
}
