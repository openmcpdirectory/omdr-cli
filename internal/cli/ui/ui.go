package ui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

func Success(msg string) {
	fmt.Printf("%s✓%s %s\n", colorGreen, colorReset, msg)
}

func Error(msg string) {
	fmt.Fprintf(os.Stderr, "%s✗%s %s\n", colorRed, colorReset, msg)
}

func Warn(msg string) {
	fmt.Printf("%s!%s %s\n", colorYellow, colorReset, msg)
}

func Info(msg string) {
	fmt.Printf("%s→%s %s\n", colorCyan, colorReset, msg)
}

func Bold(msg string) string {
	return colorBold + msg + colorReset
}

func Dim(msg string) string {
	return colorDim + msg + colorReset
}

// Table prints a formatted table with headers and rows.
func Table(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

// JSON outputs data as formatted JSON to stdout.
func JSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
