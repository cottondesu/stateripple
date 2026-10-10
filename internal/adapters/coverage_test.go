package adapters

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzeIncompleteInputs(t *testing.T) {
	cases := []struct{ name, path, content string }{
		{"missing dependency", "a/a.go", "package a\nimport _ \"example.invalid/missing\"\n"},
		{"invalid syntax", "a/a.go", "package a\nfunc A(){ @ }\n"},
		{"build exclusion", "a/tagged.go", "//go:build custom\n\npackage a\n"},
		{"workspace", "go.work", "go 1.23\nuse .\n"},
		{"nested module", "nested/go.mod", "module example.com/nested\ngo 1.23\n"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			root := fixture(t, basicFiles())
			writeFile(t, root, tt.path, tt.content)
			commit(t, root)
			_, m, err := Analyze(context.Background(), root, "")
			if err != nil {
				t.Fatal(err)
			}
			if !m.Incomplete {
				t.Fatal("unsupported input reported complete")
			}
		})
	}
}

func TestAnalyzeEmbed(t *testing.T) {
	for _, kind := range []string{"production", "test", "deleted"} {
		t.Run(kind, func(t *testing.T) {
			files := basicFiles()
			files["a/asset.txt"] = "before"
			file := "a/embed.go"
			id := "pkg:example.com/m/a"
			risk := 3
			if kind == "test" {
				file = "a/embed_test.go"
				id = "test:example.com/m/a"
				risk = 2
			}
			files[file] = "package a\nimport _ \"embed\"\n//go:embed asset.txt\nvar asset string\n"
			root := fixture(t, files)
			if kind == "deleted" {
				if err := os.Remove(filepath.Join(root, "a/asset.txt")); err != nil {
					t.Fatal(err)
				}
			} else {
				writeFile(t, root, "a/asset.txt", "after")
			}
			commit(t, root)
			s, m, err := Analyze(context.Background(), root, "")
			if err != nil {
				t.Fatal(err)
			}
			if kind == "deleted" {
				if !m.Incomplete {
					t.Fatal("missing embed must be incomplete")
				}
			} else {
				if m.Incomplete || !seeded(s, id, risk) {
					t.Fatalf("embed attribution: %+v %+v", s.Seeds, m)
				}
			}
		})
	}
}

func TestAnalyzeMovedPackage(t *testing.T) {
	root := fixture(t, map[string]string{"go.mod": "module example.com/m\ngo 1.23\n", "old/source.go": "package old\n"})
	if err := os.Mkdir(filepath.Join(root, "new"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(root, "old/source.go"), filepath.Join(root, "new/source.go")); err != nil {
		t.Fatal(err)
	}
	commit(t, root)
	s, m, err := Analyze(context.Background(), root, "")
	if err != nil {
		t.Fatal(err)
	}
	if !m.Incomplete || len(m.UnknownChanges) != 1 || m.UnknownChanges[0] != "old/source.go" || !seeded(s, "pkg:example.com/m/new", 3) {
		t.Fatalf("move attribution: %+v %+v", s.Seeds, m)
	}
}

func TestGitNULPaths(t *testing.T) {
	root := fixture(t, map[string]string{"go.mod": "module example.com/m\ngo 1.23\n", "root.go": "package m\n"})
	for _, path := range []string{"space path.md", "tab\tpath.md", "line\npath.md", "日本語.md"} {
		writeFile(t, root, path, "text")
	}
	commit(t, root)
	changes, err := gitChanges(context.Background(), root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(changes) != 4 {
		t.Fatalf("%+v", changes)
	}
}

func TestParseDiffRejectsMalformed(t *testing.T) {
	for _, data := range []string{"M\x00x", "R100\x00x\x00y\x00", "M\x00\x00"} {
		if _, err := parseDiff([]byte(data)); err == nil {
			t.Fatalf("accepted %q", data)
		}
	}
}

func TestOfflineEnvironment(t *testing.T) {
	t.Setenv("GOPROXY", "https://invalid.example")
	env := offlineEnvironment()
	for _, key := range []string{"GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off"} {
		found := false
		for _, v := range env {
			if v == key {
				found = true
			}
		}
		if !found {
			t.Fatal("missing " + key)
		}
	}
	if os.Getenv("GOPROXY") != "https://invalid.example" {
		t.Fatal("parent environment changed")
	}
}
