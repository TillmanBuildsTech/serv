//go:build linux

package platform

import (
	"fmt"
	"strings"
	"testing"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

// trickyValues covers spaces, both quote kinds, systemd specifiers ('%'),
// environment expansion ('$'), backslashes, XML-special characters and an
// embedded newline (directive injection attempt).
var trickyValues = []string{
	"hello world",
	`say "hi"`,
	"it's",
	"100%",
	"%n",
	"$HOME",
	"${PATH}",
	`C:\path\to`,
	`trailing\`,
	"a&b<c>",
	"",
	"line1\nExecStartPre=/bin/evil",
}

func TestRenderUnitQuotesExecStartAndEnvironment(t *testing.T) {
	cfg := &api.ServiceConfig{
		Name:        "myapp",
		Executable:  "/opt/my app/bin/server",
		Arguments:   trickyValues,
		Environment: map[string]string{"B_VAR": "bar baz", "A_VAR": `q"uo\te 50% $x &<`},
	}
	unit := renderUnit(cfg)

	wantExec := `ExecStart="/opt/my app/bin/server" "hello world" "say \"hi\"" "it's" "100%%" "%%n" "$$HOME" "$${PATH}" "C:\\path\\to" "trailing\\" "a&b<c>" "" "line1\nExecStartPre=/bin/evil"`
	if !strings.Contains(unit, wantExec+"\n") {
		t.Errorf("ExecStart line wrong.\nwant: %s\ngot unit:\n%s", wantExec, unit)
	}
	wantEnv := "Environment=\"A_VAR=q\\\"uo\\\\te 50%% $x &<\"\nEnvironment=\"B_VAR=bar baz\"\n"
	if !strings.Contains(unit, wantEnv) {
		t.Errorf("Environment lines wrong (or not sorted).\nwant:\n%s\ngot unit:\n%s", wantEnv, unit)
	}
	for _, line := range strings.Split(unit, "\n") {
		if strings.HasPrefix(line, "ExecStartPre=") {
			t.Fatalf("newline in an argument injected a directive:\n%s", unit)
		}
	}
}

func TestExecStartRoundTrip(t *testing.T) {
	words := append([]string{"/opt/my app/bin/server"}, trickyValues...)
	quoted := make([]string, len(words))
	for i, w := range words {
		quoted[i] = systemdQuoteExecWord(w)
	}
	got, err := splitSystemdExecLine(strings.Join(quoted, " "), true)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(words) {
		t.Fatalf("got %d words %q, want %d %q", len(got), got, len(words), words)
	}
	for i := range words {
		if got[i] != words[i] {
			t.Errorf("word %d: got %q, want %q", i, got[i], words[i])
		}
	}
}

func TestEnvironmentRoundTrip(t *testing.T) {
	for _, v := range append(trickyValues, "a$$b") {
		// Environment= does not expand $VAR, so '$' must pass through as-is.
		got, err := splitSystemdExecLine(systemdQuoteEnvAssignment("K", v), false)
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0] != "K="+v {
			t.Errorf("value %q: got %q, want [%q]", v, got, "K="+v)
		}
	}
}

func TestSystemdPlainValue(t *testing.T) {
	if got := systemdPlainValue("50% off\nUser=root"); got != "50%% off User=root" {
		t.Errorf("systemdPlainValue = %q", got)
	}
}

// splitSystemdExecLine is a test-only model of how systemd splits and
// unescapes an ExecStart= line (systemd.syntax(7), systemd.service(5)):
// whitespace-separated words, double-quoted words with C-style escapes,
// "%%" -> "%" and (for ExecStart=) "$$" -> "$". It only supports the subset of syntax
// serv emits (every word double-quoted).
func splitSystemdExecLine(line string, dollarEscapes bool) ([]string, error) {
	var words []string
	i := 0
	for i < len(line) {
		for i < len(line) && line[i] == ' ' {
			i++
		}
		if i >= len(line) {
			break
		}
		if line[i] != '"' {
			return nil, fmt.Errorf("unquoted word at %d in %q", i, line)
		}
		i++
		var b strings.Builder
		closed := false
		for i < len(line) {
			c := line[i]
			switch {
			case c == '\\' && i+1 < len(line):
				switch line[i+1] {
				case 'n':
					b.WriteByte('\n')
				case 'r':
					b.WriteByte('\r')
				case 't':
					b.WriteByte('\t')
				default:
					b.WriteByte(line[i+1])
				}
				i += 2
			case c == '%' && i+1 < len(line) && line[i+1] == '%':
				b.WriteByte('%')
				i += 2
			case c == '%':
				return nil, fmt.Errorf("unescaped specifier at %d in %q", i, line)
			case dollarEscapes && c == '$' && i+1 < len(line) && line[i+1] == '$':
				b.WriteByte('$')
				i += 2
			case c == '"':
				closed = true
				i++
			default:
				b.WriteByte(c)
				i++
			}
			if closed {
				break
			}
		}
		if !closed {
			return nil, fmt.Errorf("unterminated quote in %q", line)
		}
		words = append(words, b.String())
	}
	return words, nil
}
