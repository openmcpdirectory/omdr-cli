package main

import (
	"os"

	"github.com/openmcpdirectory/omdr/internal/cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
