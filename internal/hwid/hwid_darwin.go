//go:build darwin

package hwid

import (
	"fmt"
	"os/exec"
	"strings"
)

// collectRaw retrieves the IOPlatformUUID from macOS via ioreg.
func collectRaw() (string, error) {
	out, err := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice").Output()
	if err != nil {
		return "", fmt.Errorf("ioreg command failed: %w", err)
	}

	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "IOPlatformUUID") {
			// Line format: "IOPlatformUUID" = "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			uuid := strings.TrimSpace(parts[1])
			uuid = strings.Trim(uuid, `"`)
			if uuid == "" {
				return "", fmt.Errorf("empty IOPlatformUUID value")
			}
			return uuid, nil
		}
	}

	return "", fmt.Errorf("IOPlatformUUID not found in ioreg output")
}
