package platform

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"sort"
	"strings"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

// The plist renderer is platform-neutral (no launchctl calls) so its
// escaping can be tested on any OS; darwin.go uses it.

func label(name string) string { return fmt.Sprintf("com.serv.%s", name) }

// xmlText escapes s for use as XML character data inside a plist
// <string>/<key> element, so values containing '&', '<', '>' or quotes
// cannot produce a malformed plist or inject extra keys.
func xmlText(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// renderPlist builds a launchd plist from cfg.
func renderPlist(cfg *api.ServiceConfig) string {
	var b strings.Builder

	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString("<plist version=\"1.0\">\n<dict>\n")

	fmt.Fprintf(&b, "\t<key>Label</key>\n\t<string>%s</string>\n", xmlText(label(cfg.Name)))

	b.WriteString("\t<key>ProgramArguments</key>\n\t<array>\n")
	fmt.Fprintf(&b, "\t\t<string>%s</string>\n", xmlText(cfg.Executable))
	for _, arg := range cfg.Arguments {
		fmt.Fprintf(&b, "\t\t<string>%s</string>\n", xmlText(arg))
	}
	b.WriteString("\t</array>\n")

	if cfg.WorkingDirectory != "" {
		fmt.Fprintf(&b, "\t<key>WorkingDirectory</key>\n\t<string>%s</string>\n", xmlText(cfg.WorkingDirectory))
	}

	if len(cfg.Environment) > 0 {
		b.WriteString("\t<key>EnvironmentVariables</key>\n\t<dict>\n")
		keys := make([]string, 0, len(cfg.Environment))
		for k := range cfg.Environment {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "\t\t<key>%s</key>\n\t\t<string>%s</string>\n", xmlText(k), xmlText(cfg.Environment[k]))
		}
		b.WriteString("\t</dict>\n")
	}

	if cfg.Account.Type == api.AccountTypeUser && cfg.Account.Username != "" {
		fmt.Fprintf(&b, "\t<key>UserName</key>\n\t<string>%s</string>\n", xmlText(cfg.Account.Username))
	}

	keepAlive := cfg.Restart.Enabled == nil || *cfg.Restart.Enabled
	fmt.Fprintf(&b, "\t<key>KeepAlive</key>\n\t<%s/>\n", boolTag(keepAlive))

	if cfg.Restart.ThrottleCap.Unwrap() > 0 {
		fmt.Fprintf(&b, "\t<key>ThrottleInterval</key>\n\t<integer>%d</integer>\n", int(cfg.Restart.ThrottleCap.Unwrap().Seconds()))
	}

	if cfg.Stdout != "" {
		fmt.Fprintf(&b, "\t<key>StandardOutPath</key>\n\t<string>%s</string>\n", xmlText(cfg.Stdout))
	}
	if cfg.Stderr != "" {
		fmt.Fprintf(&b, "\t<key>StandardErrorPath</key>\n\t<string>%s</string>\n", xmlText(cfg.Stderr))
	}

	b.WriteString("</dict>\n</plist>\n")

	return b.String()
}

func boolTag(v bool) string {
	if v {
		return "true"
	}
	return "false"
}
