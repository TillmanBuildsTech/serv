package platform

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/TillmanBuildsTech/serv/pkg/api"
)

var plistTrickyValues = []string{
	"hello world",
	`say "hi"`,
	"it's",
	"100%",
	"$HOME",
	`C:\path\to`,
	"a&b<c>",
	"</string><key>Program</key><string>/bin/evil",
	"",
}

// plistNode is a minimal generic plist element tree used to read the
// rendered plist back with a real XML parser.
type plistNode struct {
	XMLName  xml.Name
	Text     string      `xml:",chardata"`
	Children []plistNode `xml:",any"`
}

func parsePlist(t *testing.T, doc string) map[string]plistNode {
	t.Helper()
	var root plistNode
	if err := xml.Unmarshal([]byte(doc), &root); err != nil {
		t.Fatalf("rendered plist is not well-formed XML: %v\n%s", err, doc)
	}
	if root.XMLName.Local != "plist" || len(root.Children) != 1 || root.Children[0].XMLName.Local != "dict" {
		t.Fatalf("unexpected plist structure:\n%s", doc)
	}
	return dictEntries(t, root.Children[0])
}

func dictEntries(t *testing.T, dict plistNode) map[string]plistNode {
	t.Helper()
	out := map[string]plistNode{}
	kids := dict.Children
	if len(kids)%2 != 0 {
		t.Fatalf("dict has an odd number of children (%d)", len(kids))
	}
	for i := 0; i < len(kids); i += 2 {
		if kids[i].XMLName.Local != "key" {
			t.Fatalf("expected <key>, got <%s>", kids[i].XMLName.Local)
		}
		if _, dup := out[kids[i].Text]; dup {
			t.Fatalf("duplicate key %q (injection?)", kids[i].Text)
		}
		out[kids[i].Text] = kids[i+1]
	}
	return out
}

func TestRenderPlistEscapesValues(t *testing.T) {
	cfg := &api.ServiceConfig{
		Name:             "myapp",
		Executable:       "/opt/my app & co/bin/server",
		Arguments:        plistTrickyValues,
		WorkingDirectory: "/srv/<data> & more",
		Environment:      map[string]string{"B": "x & y", "A&<": `q"uo\te 50% $x <tag>`},
		Account:          api.AccountConfig{Type: api.AccountTypeUser, Username: "svc&user"},
		Stdout:           "/var/log/a&b.log",
		Stderr:           "/var/log/<err>.log",
	}
	doc := renderPlist(cfg)
	top := parsePlist(t, doc)

	if _, ok := top["Program"]; ok {
		t.Fatalf("an argument injected a Program key:\n%s", doc)
	}

	args := top["ProgramArguments"].Children
	want := append([]string{cfg.Executable}, plistTrickyValues...)
	if len(args) != len(want) {
		t.Fatalf("ProgramArguments has %d entries, want %d:\n%s", len(args), len(want), doc)
	}
	for i := range want {
		if args[i].Text != want[i] {
			t.Errorf("ProgramArguments[%d] = %q, want %q", i, args[i].Text, want[i])
		}
	}

	env := dictEntries(t, top["EnvironmentVariables"])
	for k, v := range cfg.Environment {
		if env[k].Text != v {
			t.Errorf("EnvironmentVariables[%q] = %q, want %q", k, env[k].Text, v)
		}
	}

	for key, wantVal := range map[string]string{
		"WorkingDirectory":  cfg.WorkingDirectory,
		"UserName":          cfg.Account.Username,
		"StandardOutPath":   cfg.Stdout,
		"StandardErrorPath": cfg.Stderr,
		"Label":             "com.serv.myapp",
	} {
		if got := top[key].Text; got != wantVal {
			t.Errorf("%s = %q, want %q", key, got, wantVal)
		}
	}

	if !strings.Contains(doc, "<string>a&amp;b&lt;c&gt;</string>") {
		t.Errorf("expected XML-escaped argument in plist:\n%s", doc)
	}
}
