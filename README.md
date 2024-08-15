# Hardtag
[![License](https://img.shields.io/github/license/mashape/apistatus.svg)](https://github.com/catamat/hardtag/blob/master/LICENSE)
[![Build Status](https://travis-ci.org/catamat/hardtag.svg?branch=master)](https://travis-ci.org/catamat/hardtag)
[![Go Report Card](https://goreportcard.com/badge/github.com/catamat/hardtag)](https://goreportcard.com/report/github.com/catamat/hardtag)
[![Go Reference](https://pkg.go.dev/badge/github.com/catamat/hardtag.svg)](https://pkg.go.dev/github.com/catamat/hardtag)
[![Version](https://img.shields.io/github/tag/catamat/hardtag.svg?color=blue&label=version)](https://github.com/catamat/hardtag/releases)

Hardtag is simple package to generate unique machine identifiers based on hardware.

## Installation:
```
go get github.com/catamat/hardtag@latest
```
## Example:
```golang
package main

import (
	"fmt"
	"github.com/catamat/hardtag"
)

func main() {
	tag, err := hardtag.GenerateFromMAC()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	hashedTag := hardtag.HashWithSHA256(tag)

	fmt.Printf("tag %s - length: %d\n", tag, len(tag))
	fmt.Printf("hashedTag %s - length: %d\n", hashedTag, len(hashedTag))
}
```