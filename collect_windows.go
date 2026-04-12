//go:build windows

package hardtag

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

const fieldSep = "\x1f"

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
	raw, err := runPS(`
	$p = Get-CimInstance Win32_ComputerSystemProduct
	$b = Get-CimInstance Win32_BaseBoard
$i = Get-CimInstance Win32_BIOS
@($p.UUID, $b.SerialNumber, $i.SerialNumber) -join [char]0x1F
`)
	if err != nil {
		return "", fmt.Errorf("powershell: %w", err)
	}

	f := padFields(splitFields(raw), 3)
	uuid := normalizeUUID(f[0])
	boardSerial := normalizeFirmwareValue(f[1])
	biosSerial := normalizeFirmwareValue(f[2])

	if uuid == "" && boardSerial == "" && biosSerial == "" {
		return "", unavailable("cannot read WMI instance identifiers")
	}

	return normalize([]string{
		"instance", uuid, boardSerial, biosSerial,
	}), nil
}

// --- HybridLevel (cross-OS compatible) ---------------------------------------

func collectHybrid() (string, error) {
	raw, err := runPS(`
$dsk = (Get-CimInstance Win32_DiskDrive |
        Sort-Object DeviceID |
        Where-Object { $_.SerialNumber } |
        ForEach-Object { $_.SerialNumber.Trim() }) -join ','
$mac = (Get-CimInstance Win32_NetworkAdapter |
        Where-Object { $_.PhysicalAdapter -and $_.MACAddress } |
        Sort-Object DeviceID |
        ForEach-Object { $_.MACAddress }) -join ','
@($dsk, $mac) -join [char]0x1F
`)
	if err != nil {
		return "", fmt.Errorf("powershell: %w", err)
	}

	f := padFields(splitFields(raw), 2)
	rawSerials, rawMACs := f[0], f[1]

	serials := normalizeSortedList(rawSerials, normalizeSerial)
	macs := normalizeSortedList(rawMACs, normalizeMAC)

	if serials == "" && macs == "" {
		return "", unavailable("cannot read disk serials or MACs")
	}

	return normalize([]string{"hybrid", serials, macs}), nil
}

// --- helpers -----------------------------------------------------------------

func runPS(script string) (string, error) {
	shell, err := findCommand("pwsh", "powershell")
	if err != nil {
		return "", err
	}

	cmd := exec.Command(shell,
		"-NoProfile", "-NoLogo", "-NonInteractive",
		"-Command", script)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func splitFields(raw string) []string {
	parts := strings.Split(raw, fieldSep)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func padFields(f []string, n int) []string {
	for len(f) < n {
		f = append(f, "")
	}
	return f
}

// normalizeSortedList splits a comma-separated string, applies fn to each
// element, removes blanks, sorts, and re-joins with commas.
func normalizeSortedList(csv string, fn func(string) string) string {
	if csv == "" {
		return ""
	}
	parts := strings.Split(csv, ",")
	var out []string
	seen := map[string]bool{}
	for _, p := range parts {
		s := fn(p)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}
