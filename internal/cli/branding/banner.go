package branding

import (
	"fmt"
	"os"

	"github.com/openmcpdirectory/omdr-cli/internal/version"
)

const (
	Banner = `
   ██████  ███    ███ ██████  ██████
  ██    ██ ████  ████ ██   ██ ██   ██
  ██    ██ ██ ████ ██ ██   ██ ██████
  ██    ██ ██  ██  ██ ██   ██ ██   ██
   ██████  ██      ██ ██████  ██   ██
`

	BannerCompact = `
  ▄▄▄  █ █ █ █▄▄  ██▄
 █   █ █ █ █ █  █ █▄▀
 ▀▀▀▀  ▀ ▀ ▀ ▀▀▀  ▀ ▀
`

	BannerASCII = `
   ___  __  __ ___  ___
  / _ \|  \/  |   \| _ \
 | (_) | |\/| | |) |   /
  \___/|_|  |_|___/|_|_\
`
)

// ShowBanner displays the OMDR banner with version
func ShowBanner() {
	if ShouldShowBanner() {
		fmt.Println(GetBanner())
	}
}

// GetBanner returns the appropriate banner based on environment
func GetBanner() string {
	style := os.Getenv("OMDR_BANNER")

	var banner string
	switch style {
	case "compact":
		banner = BannerCompact
	case "ascii":
		banner = BannerASCII
	case "none", "off", "false":
		return ""
	default:
		banner = Banner
	}

	return fmt.Sprintf("%s\n  Open MCP Directory v%s\n", banner, version.GetVersion())
}

// ShouldShowBanner checks if banner should be displayed
func ShouldShowBanner() bool {
	// Don't show in CI/CD environments
	if os.Getenv("CI") != "" {
		return false
	}

	// Don't show if explicitly disabled
	if os.Getenv("OMDR_BANNER") == "none" ||
		os.Getenv("OMDR_BANNER") == "off" ||
		os.Getenv("OMDR_BANNER") == "false" {
		return false
	}

	// Don't show if output is not a terminal
	if !isTerminal() {
		return false
	}

	return true
}

// isTerminal checks if stdout is a terminal
func isTerminal() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
