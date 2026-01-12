package logger

import (
	"fmt"
	"os"
)

var verboseEnabled bool

func SetVerbose(enabled bool) {
	verboseEnabled = enabled
}

func Verbose(format string, args ...interface{}) {
	if verboseEnabled {
		fmt.Fprintf(os.Stderr, "[VERBOSE] "+format+"\n", args...)
	}
}

func Info(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

func Error(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}

func Success(format string, args ...interface{}) {
	fmt.Printf("✓ "+format+"\n", args...)
}

func Warning(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Warning: "+format+"\n", args...)
}
