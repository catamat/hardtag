package hardtag

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"sort"
	"strings"
)

// getMACAddresses returns a list of MAC addresses of the machine.
func getMACAddresses() ([]string, error) {
	var macAddresses []string
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		// If the second least significant bit of the first octet of the MAC address is 1,
		// then it's a locally administered address so we omit it.
		// If HardwareAddr is nil, it's a loopbook, so omit.
		if (iface.HardwareAddr != nil) && (iface.HardwareAddr[0]&2 != 2) {
			mac := iface.HardwareAddr.String()
			if mac != "" {
				macAddresses = append(macAddresses, mac)
			}
		}
	}

	return macAddresses, nil
}

// GenerateFromMAC generates a unique tag based on the MAC addresses of the machine.
func GenerateFromMAC() (string, error) {
	macAddresses, err := getMACAddresses()
	if err != nil {
		return "", err
	}

	if len(macAddresses) == 0 {
		return "", fmt.Errorf("no MAC addresses found")
	}

	// Normalize all MAC addresses to uppercase
	for i, mac := range macAddresses {
		macAddresses[i] = strings.ToUpper(mac)
	}

	// Sort the MAC addresses
	sort.Strings(macAddresses)

	// Join the MAC addresses with a pipe character
	tag := strings.Join(macAddresses, "|")

	return tag, nil
}

// HashWithSHA256 hashes the tag using SHA-256.
func HashWithSHA256(tag string) string {
	hash := sha256.Sum256([]byte(tag))
	return hex.EncodeToString(hash[:])
}
