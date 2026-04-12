# Hardtag
[![License](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/catamat/hardtag/blob/master/LICENSE)
[![Build Status](https://travis-ci.org/catamat/hardtag.svg?branch=master)](https://travis-ci.org/catamat/hardtag)
[![Go Report Card](https://goreportcard.com/badge/github.com/catamat/hardtag)](https://goreportcard.com/report/github.com/catamat/hardtag)
[![Go Reference](https://pkg.go.dev/badge/github.com/catamat/hardtag.svg)](https://pkg.go.dev/github.com/catamat/hardtag)
[![Version](https://img.shields.io/github/tag/catamat/hardtag.svg?color=blue&label=version)](https://github.com/catamat/hardtag/releases)

Hardtag is a simple package that generates a deterministic SHA-256 hardware fingerprint of the machine it runs on. It works on Linux, Windows, and macOS with two identification strategies — **instance** and **hybrid** — that cover both privileged and unprivileged execution contexts.

## Installation
```golang
go get github.com/catamat/hardtag@latest
```

## Examples

### Instance level
```golang
package main

import (
    "fmt"
    "log"

    "github.com/catamat/hardtag"
)

func main() {
    // Access requirements depend on the platform and environment.
    res, err := hardtag.Generate(hardtag.InstanceLevel)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("level : %s\n", res.Level)
    fmt.Printf("hash  : %s\n", res.Fingerprint)
}
```

### Hybrid level
```golang
package main

import (
    "fmt"
    "log"

    "github.com/catamat/hardtag"
)

func main() {
	// Doesn't require elevated privileges
    res, err := hardtag.Generate(hardtag.HybridLevel)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("level : %s\n", res.Level)
    fmt.Printf("hash  : %s\n", res.Fingerprint)
}
```

## How it works

The library reads a small set of hardware identifiers from the operating system, concatenates them into a canonical string, and returns its SHA-256 hex digest. No data ever leaves the machine, and the same physical hardware always produces the same hash.

Two levels are available. Each one draws from a different set of identifiers, and the choice between them depends on the privileges you have and the guarantees you need.

## Levels

### Instance

The instance level reads identifiers that are burned into the motherboard firmware: the SMBIOS/UEFI product UUID, the baseboard serial number, and the product serial number on Linux, the SMBIOS/UEFI product UUID, the baseboard serial number, and the BIOS serial number on Windows, or the IOPlatformUUID and IOPlatformSerialNumber on macOS. These values are written by the manufacturer at the factory and stored in non-volatile memory on the board itself.

Because these identifiers live in firmware, access constraints depend on the platform and environment. On Linux, reading the DMI files typically requires root. On macOS, the values are read through `ioreg` and are often available without root. On Windows, access is mediated by WMI and PowerShell and may depend on the local policy. In return, the instance level offers several concrete advantages.

It survives peripheral hardware changes. Replacing a disk, adding a network card, or swapping a Wi-Fi module does not affect the fingerprint, because none of those components contribute to it. In licensing scenarios this means a user can upgrade storage or networking hardware without triggering a reactivation.

It is resistant to tampering. Changing a SMBIOS UUID requires flashing the BIOS firmware or writing directly to the EEPROM on the motherboard, which is a non-trivial operation that often requires physical access. By contrast, a disk serial can be overwritten with tools like hdparm, and a MAC address can be spoofed with a single command on any operating system.

It works in environments without local disks or physical network interfaces. Servers that boot over PXE, containers with remote-mounted storage, and embedded systems with a RAM-based root filesystem all lack the identifiers the hybrid level depends on, but they still have a motherboard with a UUID.

It provides stronger uniqueness guarantees. SMBIOS UUIDs are 128-bit values generated according to RFC 4122 and are designed to be globally unique. Disk serials and MAC addresses are nominally unique per manufacturer, but duplicate serials on budget SSDs and MAC collisions on low-cost network adapters are well-documented phenomena.

### Hybrid

The hybrid level reads disk serial numbers and physical MAC addresses. On Linux it uses lsblk (with a sysfs fallback), on Windows it queries Win32_DiskDrive and Win32_NetworkAdapter via PowerShell, and on macOS it parses system_profiler output and the net package's interface list. All values are normalized to a common canonical form — serials are lowercased and stripped of non-printable characters, MACs are converted to lowercase colon-separated octets — so that the same hardware produces the same hash regardless of the operating system installed on it.

The hybrid level does not require root or Administrator privileges, which makes it the practical choice for desktop applications distributed to end users who run the software under a regular account.

The trade-off is reduced stability. Replacing or adding a disk changes the set of serials, and replacing a network card changes the set of MACs, both of which alter the fingerprint. It is also easier to tamper with, since the identifiers it depends on can be modified from userspace.

### Choosing a level

For server, appliance, or enterprise installations where the process runs with elevated privileges and long-term stability matters, prefer the instance level. For desktop or end-user applications where root access is not guaranteed and cross-OS consistency on the same hardware is important, use the hybrid level.

Hardtag does not auto-select a level for you. The calling application should decide which level is appropriate for its execution model and permission model.

## Supported platforms

| Platform | Instance | Hybrid |
|----------|----------|--------|
| Linux | ✓ (typically requires root) — SMBIOS product UUID, baseboard serial, product serial via DMI files under `/sys/class/dmi/id/` | ✓ — disk serials via `lsblk` (fallback sysfs), physical MAC addresses via `net.Interfaces` |
| Windows | ✓ — SMBIOS product UUID, baseboard serial, BIOS serial via WMI classes `Win32_ComputerSystemProduct`, `Win32_BaseBoard`, `Win32_BIOS` | ✓ — disk serials via `Win32_DiskDrive`, physical MAC addresses via `Win32_NetworkAdapter` (PowerShell) |
| macOS | ✓ — IOPlatformUUID and IOPlatformSerialNumber via `ioreg` | ✓ — disk serials and physical MAC addresses via `system_profiler` and `net.Interfaces` |

### Cross-OS compatibility

The **hybrid** level is designed to be cross-OS stable: all serials and MAC addresses are normalized to a platform-independent canonical form (lowercase, trimmed, colon-separated octets for MACs) before hashing. When the same disk serials and physical MAC addresses are visible to the collectors on Linux, Windows, and macOS, the resulting fingerprint is the same.

The **instance** level is **not** cross-OS compatible. Each operating system exposes different firmware fields through different subsystems (DMI sysfs on Linux, WMI on Windows, IOKit on macOS), and macOS in particular reports only two values instead of three. The same motherboard will therefore produce a different instance fingerprint on each OS. This is by design: the instance level prioritizes stability and tamper resistance within a single OS installation, not portability across operating systems.

## API

`hardtag.Generate(level Level) (Result, error)` generates a fingerprint at the requested level.

`Result.Fingerprint` is the 64-character lowercase hex SHA-256 digest.

`Result.Level` echoes the requested level, returned as a Level constant that prints as "instance" or "hybrid".
