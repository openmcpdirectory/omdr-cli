package main

import (
	"os"

	"github.com/openmcpdirectory/omdr-cli/internal/cli/cmd"
)

// OMDR CLI
// Author: Asman Mirza <asman@omdr.dev>
// Copyright (c) 2026 OMDR Team

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
