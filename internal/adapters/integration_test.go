package adapters

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cottondesu/stateripple/internal/snapshot"
)

func fixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for path, content := range files {
		writeFile(t, root, path, content)
	}
	runGit(t, root, "init", "-q")
	runGit(t, root, "config", "user.email", "fixture@example.invalid")
	runGit(t, root, "config", "user.name", "Fixture")
	commit(t, root)
	return root
}
func writeFile(t *testing.T, root, path, content string) {
	t.Helper()
	full := filepath.Join(root, path)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
func runGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
}
func commit(t *testing.T, root string) {
	t.Helper()
	runGit(t, root, "add", "--all")
	runGit(t, root, "commit", "-qm", "fixture")
}
func basicFiles() map[string]string {
	return map[string]string{"go.mod": "module example.com/m\n\ngo 1.23\n", "a/a.go": "package a\nfunc A() {}\n", "a/a_test.go": "package a\nimport \"testing\"\nfunc TestA(t *testing.T) {}\n", "b/b.go": "package b\nimport \"example.com/m/a\"\nfunc B(){a.A()}\n", "b/b_test.go": "package b_test\nimport (\"testing\";\"example.com/m/a\")\nfunc TestB(t *testing.T){a.A()}\n"}
}
func seeded(s snapshot.Snapshot, id string, risk int) bool {
	for _, v := range s.Seeds {
		if v.ID == id && v.Risk == risk {
			return true
		}
	}
	return false
}

func TestAnalyzeChanges(t *testing.T) {
	tests := []struct {
		name, path, content, target string
		risk                        int
		remove, incomplete          bool
	}{
		{name: "source", path: "a/a.go", content: "package a\nfunc A(){println(1)}\n", target: "pkg:example.com/m/a", risk: 3},
		{name: "test only", path: "a/a_test.go", content: "package a\nimport \"testing\"\nfunc TestA(t *testing.T){t.Log(1)}\n", target: "test:example.com/m/a", risk: 2},
		{name: "add", path: "a/new.go", content: "package a\n", target: "pkg:example.com/m/a", risk: 3},
		{name: "delete", path: "a/extra.go", remove: true, target: "pkg:example.com/m/a", risk: 3},
		{name: "test deleted", path: "a/a_test.go", remove: true, incomplete: true},
		{name: "unknown", path: "Makefile", content: "all:\n", incomplete: true},
		{name: "module", path: "go.mod", content: "module example.com/m\n\ngo 1.23\n// changed\n", target: "pkg:example.com/m/b", risk: 3},
		{name: "Unicode", path: "a/日本 語.go", content: "package a\n", target: "pkg:example.com/m/a", risk: 3},
		{name: "space", path: "a/space file.go", content: "package a\n", target: "pkg:example.com/m/a", risk: 3},
		{name: "document", path: "README.md", content: "example\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := basicFiles()
			if tt.remove && tt.path == "a/extra.go" {
				files[tt.path] = "package a\n"
			}
			root := fixture(t, files)
			if tt.remove {
				if err := os.Remove(filepath.Join(root, tt.path)); err != nil {
					t.Fatal(err)
				}
			} else {
				writeFile(t, root, tt.path, tt.content)
			}
			commit(t, root)
			s, m, err := Analyze(context.Background(), root, "")
			if err != nil {
				t.Fatal(err)
			}
			if m.Incomplete != tt.incomplete {
				t.Fatalf("incomplete=%v warnings=%v unknown=%v", m.Incomplete, m.Warnings, m.UnknownChanges)
			}
			if tt.target != "" && !seeded(s, tt.target, tt.risk) {
				t.Fatalf("missing seed %+v", s.Seeds)
			}
			if tt.name == "test only" && seeded(s, "pkg:example.com/m/a", 3) {
				t.Fatal("test seed leaked into production")
			}
		})
	}
}

func TestAnalyzeWorktree(t *testing.T) {
	for _, kind := range []string{"tracked", "untracked", "ignored", "empty"} {
		t.Run(kind, func(t *testing.T) {
			root := fixture(t, basicFiles())
			switch kind {
			case "tracked":
				writeFile(t, root, "a/a.go", "package a\n")
			case "untracked":
				writeFile(t, root, "untracked", "x")
			case "ignored":
				writeFile(t, root, ".gitignore", "ignored\n")
				commit(t, root)
				writeFile(t, root, "ignored", "x")
			}
			_, _, err := Analyze(context.Background(), root, "HEAD")
			if (err != nil) != (kind == "tracked" || kind == "untracked") {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAnalyzeDependencies(t *testing.T) {
	root := fixture(t, basicFiles())
	s, m, err := Analyze(context.Background(), root, "HEAD")
	if err != nil || m.Incomplete {
		t.Fatalf("%v %+v", err, m)
	}
	want := map[string]bool{"pkg:example.com/m/a|pkg:example.com/m/b|prod_import": false, "pkg:example.com/m/a|test:example.com/m/b|test_import": false, "pkg:example.com/m/a|test:example.com/m/a|own_test": false}
	for _, e := range s.Edges {
		key := e.From + "|" + e.To + "|" + e.Kind
		if _, ok := want[key]; ok {
			want[key] = true
		}
	}
	for key, seen := range want {
		if !seen {
			t.Fatal("missing edge " + key)
		}
	}
	for _, n := range s.Nodes {
		if strings.Contains(n.ID, "testing") {
			t.Fatal("external node")
		}
	}
}
