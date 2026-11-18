//go:build linux

package platform

import "strings"

// systemdEscapeQuoted escapes s for use inside a double-quoted systemd
// value (systemd.syntax(7)): backslash and double quote are escaped, and
// control characters use C-style escapes so a value can never break onto a
// new line and inject directives. '%' is doubled because systemd expands
// specifiers (systemd.unit(5)) in these settings.
func systemdEscapeQuoted(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '"':
			b.WriteString(`\"`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case '%':
			b.WriteString("%%")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// systemdQuoteExecWord renders one word of an ExecStart= command line so
// systemd passes it to the process as exactly one argv entry, unchanged:
// it is double-quoted, escaped as above, and '$' is doubled because
// ExecStart= expands $VAR / ${VAR} from the unit's environment
// (systemd.service(5), "Command lines").
func systemdQuoteExecWord(s string) string {
	return `"` + strings.ReplaceAll(systemdEscapeQuoted(s), "$", "$$") + `"`
}

// systemdQuoteEnvAssignment renders the value of an Environment= line as a
// single quoted "KEY=value" assignment. Environment= does not expand
// $VAR, so '$' is left alone.
func systemdQuoteEnvAssignment(key, value string) string {
	return `"` + systemdEscapeQuoted(key+"="+value) + `"`
}

// systemdPlainValue sanitizes a single-valued setting (Description=,
// WorkingDirectory=, User=, StandardOutput=append:...) that systemd does not
// unquote: specifiers are escaped and line breaks are replaced with spaces
// so the value cannot inject further directives.
func systemdPlainValue(s string) string {
	s = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(s)
	return strings.ReplaceAll(s, "%", "%%")
}
