//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package adapters

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeOfflineRejectsInheritedPrivateVCS(t *testing.T) {
	// Given: a missing private module with syntactically valid sums (without
	// sums Go can stop before the vulnerable VCS lookup).
	files := basicFiles()
	files["go.mod"] = "module example.com/m\ngo 1.23\nrequire example.invalid/private.git v0.0.0\n"
	files["go.sum"] = "example.invalid/private.git v0.0.0 h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\nexample.invalid/private.git v0.0.0/go.mod h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\n"
	files["a/a.go"] = "package a\nimport _ \"example.invalid/private.git\"\nfunc A(){}\n"
	root := fixture(t, files)
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	shim := t.TempDir()
	log := filepath.Join(shim, "vcs.log")
	script := "#!/bin/sh\nfor arg in \"$@\"; do\ncase \"$arg\" in ls-remote|fetch|clone|pull|push) printf '%s\\n' \"$*\" >> \"$STATERIPPLE_VCS_LOG\"; exit 86;; esac\ndone\nexec \"$STATERIPPLE_REAL_GIT\" \"$@\"\n"
	writeFile(t, shim, "git", script)
	if err := os.Chmod(filepath.Join(shim, "git"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", shim+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("STATERIPPLE_REAL_GIT", git)
	t.Setenv("STATERIPPLE_VCS_LOG", log)
	t.Setenv("GOMODCACHE", t.TempDir())
	t.Setenv("GONOPROXY", "*")
	t.Setenv("GOPRIVATE", "*")
	t.Setenv("GOVCS", "*:all")
	// When: only analysis uses the shim; fixture Git operations preceded it.
	_, meta, analyzeErr := Analyze(context.Background(), root, "HEAD")
	// Then: unresolved dependencies cannot be complete and VCS is never called.
	if analyzeErr == nil && !meta.Incomplete {
		t.Fatal("missing dependency reported complete")
	}
	data, readErr := os.ReadFile(log)
	if readErr != nil && !os.IsNotExist(readErr) {
		t.Fatal(readErr)
	}
	if len(strings.TrimSpace(string(data))) != 0 {
		t.Fatalf("offline VCS calls: %s", data)
	}
}
