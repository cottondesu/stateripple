//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package adapters

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeRejectsUnsupportedGitBeforeGoCollection(t *testing.T) {
	for _, tc := range []struct{ name, script string }{
		{"old", "printf 'git version 2.44.0\\n'\n"},
		{"option-missing", "printf 'unknown option: --no-lazy-fetch\\n' >&2\nexit 129\n"},
		{"unrecognized", "printf 'unrecognized version\\n'\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := fixture(t, basicFiles())
			shim := t.TempDir()
			log := filepath.Join(shim, "go-called")
			writeFile(t, shim, "git", "#!/bin/sh\n"+tc.script)
			writeFile(t, shim, "go", "#!/bin/sh\nprintf 'called' > '"+log+"'\nexit 85\n")
			for _, name := range []string{"git", "go"} {
				if err := os.Chmod(filepath.Join(shim, name), 0o700); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("PATH", shim+string(os.PathListSeparator)+os.Getenv("PATH"))
			_, _, err := Analyze(context.Background(), root, "HEAD")
			if err == nil || !strings.Contains(err.Error(), "Git 2.45.0") {
				t.Fatalf("unsupported Git diagnostic: %v", err)
			}
			if _, err := os.Stat(log); !os.IsNotExist(err) {
				t.Fatalf("Go invoked before Git capability accepted: %v", err)
			}
		})
	}
}
