package adapters

import (
	"context"
	"strings"
	"testing"
)

func TestReplaceCompletenessDependsOnReachablePackages(t *testing.T) {
	for _, tt := range []struct {
		name                string
		local, exists, used bool
	}{
		{"active-local", true, true, true},
		{"missing-active-local", true, false, true},
		{"unused-local", true, true, false},
		{"missing-unused-local", true, false, false},
		{"unused-remote", false, false, false},
		{"missing-active-remote", false, false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GOMODCACHE", t.TempDir())
			files := basicFiles()
			files["go.mod"] = "module example.com/m\ngo 1.23\nrequire example.com/ext v1.0.0\n"
			if tt.local {
				dir := t.TempDir()
				if tt.exists {
					writeFile(t, dir, "go.mod", "module example.com/ext\ngo 1.23\n")
					writeFile(t, dir, "ext.go", "package ext\n")
				} else {
					dir += "/missing"
				}
				files["go.mod"] += "replace example.com/ext => " + dir + "\n"
			} else if !tt.used {
				files["go.mod"] += "replace example.com/ext => example.com/unused v1.2.3\n"
			}
			if tt.used {
				files["a/a.go"] = "package a\nimport _ \"example.com/ext\"\nfunc A(){}\n"
			}
			root := fixture(t, files)
			_, meta, err := Analyze(context.Background(), root, "HEAD")
			if err != nil || meta.Incomplete != tt.used {
				t.Fatalf("expected incomplete=%t: %v %+v", tt.used, err, meta)
			}
			if tt.local && tt.exists && tt.used && !strings.Contains(strings.Join(meta.Warnings, "\n"), "local module replacements") {
				t.Fatalf("missing local warning: %+v", meta)
			}
			if !tt.used && len(meta.Warnings) != 0 {
				t.Fatalf("unused directive warnings: %v", meta.Warnings)
			}
		})
	}
}
