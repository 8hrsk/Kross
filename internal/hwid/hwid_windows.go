//go:build windows

package hwid

import (
	"fmt"
	"os/exec"
	"strings"
)

// collectRaw retrieves the machine UUID via WMIC on Windows.
func collectRaw() (string, error) {
	out, err := exec.Command("wmic", "csproduct", "get", "UUID").Output()
	if err != nil {
		return "", fmt.Errorf("wmic command failed: %w", err)
	}

	lines := strings.Split(strings.ReplaceAll(string(out), "\r", ""), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip the header line and empty lines.
		if line == "" || strings.EqualFold(line, "UUID") {
			continue
		}
		return line, nil
	}

	return "", fmt.Errorf("UUID not found in wmic output")
}
