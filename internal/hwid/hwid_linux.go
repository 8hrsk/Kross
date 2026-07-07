//go:build linux

package hwid

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// collectRaw attempts to read the DMI product UUID, falling back to
// the first non-loopback network interface MAC address.
func collectRaw() (string, error) {
	uuid, err := readProductUUID()
	if err == nil {
		return uuid, nil
	}

	mac, macErr := readFirstMACAddress()
	if macErr != nil {
		return "", fmt.Errorf("product_uuid failed (%v) and MAC fallback failed: %w", err, macErr)
	}
	return mac, nil
}

// readProductUUID reads the DMI product UUID from sysfs.
func readProductUUID() (string, error) {
	data, err := os.ReadFile("/sys/class/dmi/id/product_uuid")
	if err != nil {
		return "", err
	}
	uuid := strings.TrimSpace(string(data))
	if uuid == "" {
		return "", fmt.Errorf("empty product_uuid")
	}
	return uuid, nil
}

// readFirstMACAddress reads the MAC address of the first non-loopback
// network interface found in /sys/class/net/.
func readFirstMACAddress() (string, error) {
	entries, err := os.ReadDir("/sys/class/net")
	if err != nil {
		return "", fmt.Errorf("failed to read /sys/class/net: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if name == "lo" {
			continue
		}

		addrPath := filepath.Join("/sys/class/net", name, "address")
		data, err := os.ReadFile(addrPath)
		if err != nil {
			continue
		}

		mac := strings.TrimSpace(string(data))
		// Skip empty or all-zero MACs.
		if mac == "" || mac == "00:00:00:00:00:00" {
			continue
		}
		return mac, nil
	}

	return "", fmt.Errorf("no non-loopback network interface with valid MAC found")
}
