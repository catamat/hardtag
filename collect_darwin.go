//go:build darwin

package hardtag

import (
	"fmt"
	"net"
	"os/exec"
	"sort"
	"strings"
)

func collect(level Level) (string, error) {
	switch level {
	case InstanceLevel:
		return collectInstance()
	case HybridLevel:
		return collectHybrid()
	default:
		return "", fmt.Errorf("unknown level %d", level)
	}
}

// --- InstanceLevel -----------------------------------------------------------

func collectInstance() (string, error) {
	values, err := ioregValues()
	if err != nil {
		return "", fmt.Errorf("ioreg: %w", err)
	}

	uuid := normalizeUUID(values["IOPlatformUUID"])
	serial := normalizeFirmwareValue(values["IOPlatformSerialNumber"])

	if uuid == "" && serial == "" {
		return "", unavailable("cannot read IOKit instance identifiers")
	}

	return normalize([]string{"instance", uuid, serial}), nil
}

// --- HybridLevel (cross-OS compatible) ---------------------------------------

func collectHybrid() (string, error) {
	serials := readDiskSerials()
	macs := readPhysicalMACs()

	if len(serials) == 0 && len(macs) == 0 {
		return "", unavailable("cannot read disk serials or physical MACs")
	}

	return normalize([]string{
		"hybrid",
		strings.Join(serials, ","),
		strings.Join(macs, ","),
	}), nil
}

// --- disk serials ------------------------------------------------------------

// readDiskSerials gets raw serial numbers from system_profiler.
func readDiskSerials() []string {
	out, err := exec.Command(
		"system_profiler",
		"SPNVMeDataType", "SPSerialATADataType",
	).Output()
	if err != nil {
		return nil
	}

	return parseSystemProfilerSerials(string(out))
}

// --- physical MACs -----------------------------------------------------------

func readPhysicalMACs() []string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	var macs []string
	seen := map[string]bool{}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		// On macOS, en0 is the built-in interface.
		// Accept en* interfaces (en0 = Wi-Fi or Ethernet, en1, etc.)
		if !strings.HasPrefix(iface.Name, "en") {
			continue
		}

		mac := normalizeMAC(iface.HardwareAddr.String())
		if mac == "" || mac == "00:00:00:00:00:00" || seen[mac] {
			continue
		}
		seen[mac] = true
		macs = append(macs, mac)
	}
	sort.Strings(macs)
	return macs
}

// --- ioreg helper (instance level only) --------------------------------------

func ioregValues() (map[string]string, error) {
	cmd := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	values := map[string]string{}
	for _, line := range strings.Split(string(out), "\n") {
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		key = strings.Trim(key, `"`)
		if key == "" {
			continue
		}
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"<> `)
		values[key] = val
	}
	return values, nil
}
