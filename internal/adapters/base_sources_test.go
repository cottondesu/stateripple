package adapters

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormerEmbedsReadsEveryBaseSource(t *testing.T) {
	for _, tc := range []struct {
		name    string
		missing []string
		match   bool
	}{
		{"present-match", nil, true},
		{"present-no-match", nil, false},
		{"missing-with-match", []string{"b.go"}, true},
		{"missing-without-match", []string{"b.go"}, false},
		{"multiple-missing", []string{"b.go", "c.go"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given: independent loose base objects, with selected inputs removed.
			files := map[string]string{"a.go": "package a\n", "b.go": "package b\n", "c.go": "package c\n"}
			if tc.match || len(tc.missing) > 0 {
				files["b.go"] += "//go:embed b.md\n"
			}
			if len(tc.missing) > 1 {
				files["c.go"] += "//go:embed c.md\n"
			}
			if tc.match {
				files["a.go"] += "//go:embed a.md\n"
			}
			root := fixture(t, files)
			for _, name := range tc.missing {
				cmd := exec.Command("git", "rev-parse", "HEAD:"+name)
				cmd.Dir = root
				out, err := cmd.Output()
				if err != nil {
					t.Fatal(err)
				}
				oid := strings.TrimSpace(string(out))
				if err := os.Remove(filepath.Join(root, ".git", "objects", oid[:2], oid[2:])); err != nil {
					t.Fatal(err)
				}
			}
			// When
			patterns, err := formerEmbeds(context.Background(), root, "HEAD")
			// Then: a match elsewhere cannot certify the missing inputs.
			if (err != nil) != (len(tc.missing) != 0) {
				t.Fatalf("patterns=%v err=%v", patterns, err)
			}
			if err != nil {
				for _, name := range tc.missing {
					if !strings.Contains(err.Error(), name) {
						t.Errorf("missing input %q absent from diagnostic: %v", name, err)
					}
				}
			}
		})
	}
}

func TestAnalyzeTraceDoesNotChangeCompleteness(t *testing.T) {
	for _, key := range []string{"GIT_TRACE", "GIT_TRACE2", "GIT_TRACE_PERFORMANCE", "GIT_TRACE2_EVENT"} {
		t.Run(key, func(t *testing.T) {
			// Given
			root := fixture(t, basicFiles())
			t.Setenv(key, "1")
			// When
			_, meta, err := Analyze(context.Background(), root, "HEAD")
			// Then
			if err != nil || meta.Incomplete {
				t.Fatalf("err=%v meta=%+v", err, meta)
			}
			if os.Getenv(key) != "1" {
				t.Fatal("parent trace setting changed")
			}
		})
	}
}

func TestBaseSourcesRejectsSymbolicGoSource(t *testing.T) {
	// Given: Go compiles a symlinked .go file, but its target bytes are not this entry's blob.
	tree := "100644 blob 1111111111111111111111111111111111111111\ta.go\x00" +
		"120000 blob 2222222222222222222222222222222222222222\tb/link.go\x00" +
		"120000 blob 3333333333333333333333333333333333333333\tdocs/link.md\x00"
	// When
	_, err := baseSources([]byte(tree))
	// Then
	if err == nil || !strings.Contains(err.Error(), "b/link.go") {
		t.Fatalf("symbolic base Go source accepted: %v", err)
	}
}
