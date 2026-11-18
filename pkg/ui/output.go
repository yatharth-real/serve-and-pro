package ui

import (
	"encoding/json"
	"fmt"
	"os"
)

// PrintJSON marshals v and writes it to stdout. Dies on error.
func PrintJSON(v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error serializing JSON: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

// Row is a 2-column table row.
type Row = [2]string

// PrintTable prints a label/value table.
func PrintTable(rows []Row) {
	maxLabel := 0
	for _, r := range rows {
		if len(r[0]) > maxLabel {
			maxLabel = len(r[0])
		}
	}
	for _, r := range rows {
		fmt.Printf("  %-*s  %s\n", maxLabel, r[0], r[1])
	}
}

// Severity constants for status display.
const (
	SeverityOK    = "OK"
	SeverityInfo  = "INFO"
	SeverityWarn  = "WARN"
	SeverityError = "ERROR"
)

// StatusLine represents a single status check result.
type StatusLine struct {
	Label    string `json:"label"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// PrintStatusLines renders status lines to stdout.
func PrintStatusLines(lines []StatusLine, p *Palette) {
	for _, l := range lines {
		var sev string
		switch l.Severity {
		case SeverityOK:
			sev = p.Success("OK   ")
		case SeverityInfo:
			sev = p.Accent("INFO ")
		case SeverityWarn:
			sev = p.Warning("WARN ")
		case SeverityError:
			sev = p.Error("ERROR")
		default:
			sev = l.Severity
		}
		fmt.Printf("  %s  %-20s  %s\n", sev, l.Label, l.Message)
	}
}
