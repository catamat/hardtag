//go:build linux

package hardtag

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
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

// --- InstanceLevel (root) ----------------------------------------------------

func collectInstance() (string, error) {
	uuid := normalizeUUID(readDMI("product_uuid"))
	boardSerial := normalizeFirmwareValue(readDMI("board_serial"))
	productSerial := normalizeFirmwareValue(readDMI("product_serial"))

	if uuid == "" && boardSerial == "" && productSerial == "" {
		return "", unavailable("cannot read DMI instance fields (root required)")
	}

	return normalize([]string{
		"instance", uuid, boardSerial, productSerial,
	}), nil
}

// --- HybridLevel (no root) ---------------------------------------------------

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

// --- disk serials (raw, no model/bus prefix) ---------------------------------

// readDiskSerials extracts raw serial numbers using lsblk.
// Falls back to parsing /dev/disk/by-id/ if lsblk is unavailable.
func readDiskSerials() []string {
	if s := diskSerialsFromLsblk(); len(s) > 0 {
		return s
	}
	return diskSerialsFromByID()
}

// diskSerialsFromLsblk uses "lsblk -ndo SERIAL" to get raw serials.
func diskSerialsFromLsblk() []string {
	out, err := exec.Command("lsblk", "-ndo", "SERIAL").Output()
	if err != nil {
		return nil
	}

	seen := map[string]bool{}
	var serials []string
	for _, line := range strings.Split(string(out), "\n") {
		s := normalizeSerial(line)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		serials = append(serials, s)
	}
	sort.Strings(serials)
	return serials
}

// diskSerialsFromByID parses /dev/disk/by-id/ symlink names and extracts
// the serial portion (everything after the last underscore-separated model
// segment). This is the fallback when lsblk is not available.
func diskSerialsFromByID() []string {
	entries, err := os.ReadDir("/dev/disk/by-id")
	if err != nil {
		return nil
	}

	seen := map[string]bool{}
	var serials []string
	for _, e := range entries {
		name := e.Name()

		if strings.Contains(name, "-part") {
			continue
		}
		if !(strings.HasPrefix(name, "ata-") ||
			strings.HasPrefix(name, "nvme-") ||
			strings.HasPrefix(name, "scsi-") ||
			strings.HasPrefix(name, "virtio-")) {
			continue
		}

		// Attempt to read the serial from sysfs for this device.
		serial := serialFromSysfs(name)
		if serial == "" {
			serial = serialFromByIDName(name)
		}

		if serial == "" || seen[serial] {
			continue
		}
		seen[serial] = true
		serials = append(serials, serial)
	}
	sort.Strings(serials)
	return serials
}

// serialFromSysfs resolves a /dev/disk/by-id/ symlink to its block
// device and reads the kernel-exported serial from sysfs.
func serialFromSysfs(byIDName string) string {
	link := filepath.Join("/dev/disk/by-id", byIDName)
	target, err := os.Readlink(link)
	if err != nil {
		return ""
	}
	// target is e.g. "../../sda"
	dev := filepath.Base(target)

	// NVMe: /sys/block/nvme0n1/device/serial
	// SATA/SCSI: /sys/block/sda/device/serial (may not exist)
	for _, path := range []string{
		filepath.Join("/sys/block", dev, "device", "serial"),
	} {
		if data, err := os.ReadFile(path); err == nil {
			if s := normalizeSerial(string(data)); s != "" {
				return s
			}
		}
	}
	return ""
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

		// Skip virtual interfaces.
		link, err := os.Readlink(
			filepath.Join("/sys/class/net", iface.Name))
		if err == nil && strings.Contains(link, "/virtual/") {
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

// --- DMI helper (instance level only) ----------------------------------------

func readDMI(name string) string {
	data, err := os.ReadFile(filepath.Join("/sys/class/dmi/id", name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
