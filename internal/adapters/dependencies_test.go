package adapters

import (
	"archive/zip"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAnalyzeTestOnlyImport(t *testing.T) {
	files := basicFiles()
	files["b/b.go"] = "package b\nfunc B(){}\n"
	files["b/b_test.go"] = "package b\nimport (\"testing\"; \"example.com/m/a\")\nfunc TestB(t *testing.T){a.A()}\n"
	root := fixture(t, files)
	s, m, err := Analyze(context.Background(), root, "HEAD")
	if err != nil || m.Incomplete {
		t.Fatalf("%v %+v", err, m)
	}
	found := false
	for _, edge := range s.Edges {
		if edge.From == "pkg:example.com/m/a" && edge.To == "test:example.com/m/b" && edge.Kind == "test_import" {
			found = true
		}
		if edge.From == "pkg:example.com/m/a" && edge.To == "pkg:example.com/m/b" {
			t.Fatal("test-only import became production dependency")
		}
	}
	if !found {
		t.Fatal("test-only dependency missing")
	}
}

func TestAnalyzeCachedExternalImport(t *testing.T) {
	const module = "example.com/external"
	const version = "v1.0.0"
	mod := "module " + module + "\ngo 1.23\n"
	proxy := t.TempDir()
	cache := t.TempDir()
	versionDir := filepath.Join(module, "@v")
	writeFile(t, proxy, filepath.Join(versionDir, version+".mod"), mod)
	info, err := json.Marshal(struct{ Version string }{version})
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, proxy, filepath.Join(versionDir, version+".info"), string(info))
	zipFile, err := os.Create(filepath.Join(proxy, versionDir, version+".zip"))
	if err != nil {
		t.Fatal(err)
	}
	writer := zip.NewWriter(zipFile)
	for _, file := range []struct{ path, content string }{{"go.mod", mod}, {"external.go", "package external\nfunc External(){}\n"}} {
		entry, err := writer.Create(module + "@" + version + "/" + file.path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(file.content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zipFile.Close(); err != nil {
		t.Fatal(err)
	}
	files := basicFiles()
	files["go.mod"] = "module example.com/m\ngo 1.23\nrequire " + module + " " + version + "\n"
	files["a/a.go"] = "package a\nimport \"" + module + "\"\nfunc A(){external.External()}\n"
	root := fixture(t, files)
	cmd := exec.Command("go", "mod", "download", module+"@"+version)
	cmd.Dir = root
	cmd.Env = append(offlineEnvironment(), "GOPROXY=file://"+filepath.ToSlash(proxy), "GOMODCACHE="+cache, "GOFLAGS=-modcacherw", "GOEXPERIMENT=")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("local proxy prewarm: %v: %s", err, output)
	}
	commit(t, root)
	t.Setenv("GOMODCACHE", cache)
	s, m, err := Analyze(context.Background(), root, "HEAD")
	if err != nil || m.Incomplete {
		t.Fatalf("%v %+v", err, m)
	}
	if len(s.Nodes) != 4 {
		t.Fatalf("unexpected module graph: %+v", s.Nodes)
	}
	for _, node := range s.Nodes {
		if node.ID == "pkg:"+module {
			t.Fatal("external dependency became graph node")
		}
	}
}
