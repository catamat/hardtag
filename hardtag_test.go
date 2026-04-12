package hardtag

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeMAC(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "AA:BB:CC:DD:EE:FF", want: "aa:bb:cc:dd:ee:ff"},
		{in: "AA-BB-CC-DD-EE-FF", want: "aa:bb:cc:dd:ee:ff"},
		{in: "AABB.CCDD.EEFF", want: "aa:bb:cc:dd:ee:ff"},
		{in: "bad", want: ""},
	}

	for _, tt := range tests {
		if got := normalizeMAC(tt.in); got != tt.want {
			t.Fatalf("normalizeMAC(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizeFirmwareValue(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: " To Be Filled By O.E.M. ", want: ""},
		{in: "Default String", want: ""},
		{in: "ABC123\x00", want: "abc123"},
	}

	for _, tt := range tests {
		if got := normalizeFirmwareValue(tt.in); got != tt.want {
			t.Fatalf("normalizeFirmwareValue(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestNormalizeUUID(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "00000000-0000-0000-0000-000000000000", want: ""},
		{in: "FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF", want: ""},
		{in: "{A1B2C3D4-0000-1111-2222-333344445555}", want: "{a1b2c3d4-0000-1111-2222-333344445555}"},
	}

	for _, tt := range tests {
		if got := normalizeUUID(tt.in); got != tt.want {
			t.Fatalf("normalizeUUID(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseSystemProfilerSerials(t *testing.T) {
	raw := `
NVMExpress:

    Apple SSD Controller:
        APPLE SSD AP0512R:
          Serial Number: 0BA01812810C6834
          Model: APPLE SSD AP0512R

Serial-ATA:
    Some Drive:
      Serial Number: disk-002
      Serial Number: disk-002
`

	got := parseSystemProfilerSerials(raw)
	want := []string{"0ba01812810c6834", "disk-002"}

	if len(got) != len(want) {
		t.Fatalf("parseSystemProfilerSerials() len = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseSystemProfilerSerials()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestSerialFromByIDName(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "ata-Samsung_SSD_860_EVO_500GB_S3Z9NB0K123456",
			want: "s3z9nb0k123456",
		},
		{
			name: "nvme-Samsung_SSD_970_EVO_Plus_2TB_S4J4NJ0N123456_1",
			want: "s4j4nj0n123456",
		},
		{
			name: "scsi-3600508b1001c7b6d6f12",
			want: "3600508b1001c7b6d6f12",
		},
	}

	for _, tt := range tests {
		if got := serialFromByIDName(tt.name); got != tt.want {
			t.Fatalf("serialFromByIDName(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestFindCommand(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)

	pwsh := filepath.Join(dir, "pwsh")
	if err := os.WriteFile(pwsh, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write pwsh: %v", err)
	}

	powershell := filepath.Join(dir, "powershell")
	if err := os.WriteFile(powershell, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write powershell: %v", err)
	}

	got, err := findCommand("pwsh", "powershell")
	if err != nil {
		t.Fatalf("findCommand() error = %v", err)
	}
	if got != pwsh {
		t.Fatalf("findCommand() = %q, want %q", got, pwsh)
	}

	if err := os.Remove(pwsh); err != nil {
		t.Fatalf("remove pwsh: %v", err)
	}

	got, err = findCommand("pwsh", "powershell")
	if err != nil {
		t.Fatalf("findCommand() fallback error = %v", err)
	}
	if got != powershell {
		t.Fatalf("findCommand() fallback = %q, want %q", got, powershell)
	}
}
