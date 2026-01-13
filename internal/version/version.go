package version

import (
	"fmt"
	"time"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

func GetVersion() string {
	if Version == "dev" {
		return fmt.Sprintf("dev-%s", time.Now().Format("20060102"))
	}
	return Version
}

func GetFullVersion() string {
	return fmt.Sprintf("%s (commit: %s, built: %s)", GetVersion(), Commit, Date)
}

// GetCalVer returns calendar version format YYYY.MM.DD
func GetCalVer() string {
	if Version == "dev" {
		return time.Now().Format("2006.01.02")
	}
	return Version
}
