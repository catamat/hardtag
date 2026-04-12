package hardtag

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

type Level int

const (
	InstanceLevel Level = iota
	HybridLevel
)

func (l Level) String() string {
	switch l {
	case InstanceLevel:
		return "instance"
	case HybridLevel:
		return "hybrid"
	default:
		return "unknown"
	}
}

type Result struct {
	Fingerprint string
	Level       Level
}

var errUnavailable = errors.New("hardware identifiers unavailable")

func Generate(level Level) (Result, error) {
	raw, err := collect(level)
	if err != nil {
		return Result{}, fmt.Errorf("fingerprint [%s]: %w", level, err)
	}
	sum := sha256.Sum256([]byte(raw))
	return Result{
		Fingerprint: hex.EncodeToString(sum[:]),
		Level:       level,
	}, nil
}

func normalize(parts []string) string {
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, strings.ToLower(p))
		}
	}
	return strings.Join(out, "|")
}

func unavailable(reason string) error {
	return fmt.Errorf("%w: %s", errUnavailable, reason)
}

// normalizeMAC converts any MAC format to lowercase colon-separated.
// "AA:BB:CC:DD:EE:FF" → "aa:bb:cc:dd:ee:ff"
// "AA-BB-CC-DD-EE-FF" → "aa:bb:cc:dd:ee:ff"
// "AABBCCDDEEFF"       → "aa:bb:cc:dd:ee:ff"
func normalizeMAC(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	raw = strings.ReplaceAll(raw, "-", "")
	raw = strings.ReplaceAll(raw, ":", "")
	raw = strings.ReplaceAll(raw, ".", "")
	if len(raw) != 12 {
		return ""
	}
	return fmt.Sprintf("%s:%s:%s:%s:%s:%s",
		raw[0:2], raw[2:4], raw[4:6],
		raw[6:8], raw[8:10], raw[10:12])
}

// normalizeSerial strips whitespace and non-printable chars,
// then lowercases. This is the cross-platform canonical form.
var nonPrint = regexp.MustCompile(`[^\x20-\x7E]`)

func normalizeSerial(raw string) string {
	s := strings.TrimSpace(raw)
	s = nonPrint.ReplaceAllString(s, "")
	s = strings.ToLower(s)
	return s
}

var invalidFirmwareValues = map[string]struct{}{
	"default string":         {},
	"n/a":                    {},
	"na":                     {},
	"none":                   {},
	"not applicable":         {},
	"not specified":          {},
	"system product name":    {},
	"system serial number":   {},
	"to be filled by o.e.m.": {},
	"to be filled by oem":    {},
	"unknown":                {},
}

func normalizeFirmwareValue(raw string) string {
	s := normalizeSerial(raw)
	if _, bad := invalidFirmwareValues[s]; bad {
		return ""
	}
	return s
}

func normalizeUUID(raw string) string {
	s := normalizeFirmwareValue(raw)
	if s == "" {
		return ""
	}

	compact := strings.NewReplacer("-", "", "{", "", "}", "").Replace(s)
	if len(compact) == 32 {
		if strings.Trim(compact, "0") == "" {
			return ""
		}
		if strings.Trim(compact, "f") == "" {
			return ""
		}
	}

	return s
}

func parseSystemProfilerSerials(raw string) []string {
	seen := map[string]bool{}
	var serials []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Serial Number:") {
			continue
		}
		if _, val, ok := strings.Cut(line, ":"); ok {
			s := normalizeSerial(val)
			if s == "" || seen[s] {
				continue
			}
			seen[s] = true
			serials = append(serials, s)
		}
	}
	sort.Strings(serials)
	return serials
}

func serialFromByIDName(name string) string {
	kind, rest, ok := strings.Cut(name, "-")
	if !ok || rest == "" {
		return ""
	}

	parts := strings.Split(rest, "_")
	if kind == "nvme" && len(parts) > 1 && isDigits(parts[len(parts)-1]) {
		parts = parts[:len(parts)-1]
	}
	if len(parts) == 0 {
		return ""
	}

	return normalizeSerial(parts[len(parts)-1])
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func findCommand(names ...string) (string, error) {
	for _, name := range names {
		path, err := exec.LookPath(name)
		if err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("command not found: %s", strings.Join(names, ", "))
}
