package adapters

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAnalyzeFormerEmbedWhitespace(t *testing.T) {
	for _, space := range []string{" ", "\t"} {
		t.Run(space, func(t *testing.T) {
			files := basicFiles()
			files["a/asset.md"] = "before"
			files["a/embed.go"] = "package a\nimport _ \"embed\"\n//go:embed" + space + "asset.md\nvar text string\n"
			root := fixture(t, files)
			writeFile(t, root, "a/embed.go", "package a\n")
			writeFile(t, root, "a/asset.md", "after")
			commit(t, root)
			s, m, err := Analyze(context.Background(), root, "")
			if err != nil || m.Incomplete {
				t.Fatalf("%v %+v", err, m)
			}
			if !seeded(s, "pkg:example.com/m/a", 3) {
				t.Fatal("former embed not seeded")
			}
			found := false
			for _, change := range m.Changes {
				if change.Path == "a/asset.md" && change.Classification == "former_embed" {
					found = true
				}
			}
			if !found {
				t.Fatal("former embed incorrectly classified")
			}
		})
	}
}

func TestAnalyzeIgnoredModuleInputs(t *testing.T) {
	for _, name := range []string{"go.mod", "go.sum"} {
		t.Run(name, func(t *testing.T) {
			files := basicFiles()
			files[".gitignore"] = name + "\n"
			delete(files, name)
			root := fixture(t, files)
			content := "\n"
			if name == "go.mod" {
				content = "module example.com/m\ngo 1.23\n"
			}
			writeFile(t, root, name, content)
			_, m, err := Analyze(context.Background(), root, "HEAD")
			if err != nil {
				t.Fatal(err)
			}
			if !m.Incomplete {
				t.Fatal("uncommitted module input reported complete")
			}
		})
	}
}

func TestModuleInputsSymlink(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "module", "module example.com/m\ngo 1.23\n")
	if err := os.Symlink("module", filepath.Join(root, "go.mod")); err != nil {
		t.Fatal(err)
	}
	m := Metadata{}
	checkModuleInputs(root, map[string]bool{"go.mod": true}, &m)
	if !m.Incomplete {
		t.Fatal("symlink module input reported complete")
	}
}

func TestAnalyzeSymlinkModuleInput(t *testing.T) {
	files := basicFiles()
	module := files["go.mod"]
	delete(files, "go.mod")
	files["module.txt"] = module
	root := fixture(t, files)
	if err := os.Symlink("module.txt", filepath.Join(root, "go.mod")); err != nil {
		t.Fatal(err)
	}
	commit(t, root)
	_, m, err := Analyze(context.Background(), root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if !m.Incomplete {
		t.Fatal("symlink module reported complete")
	}
}
