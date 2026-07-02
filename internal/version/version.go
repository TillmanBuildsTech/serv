// Package version holds the single source of truth for serv's version
// number, read from the VERSION file at build time via go:embed.
package version

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var raw string

// Version is the current serv version (e.g. "0.1.0").
var Version = strings.TrimSpace(raw)
