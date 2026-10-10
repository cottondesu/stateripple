package adapters

import (
	"archive/zip"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestAnalyzeCachedVersionedReplacement(t *testing.T) {
	const module = "example.com/replacement"
	const version = "v1.0.0"
	proxy, cache := t.TempDir(), t.TempDir()
	mod := "module " + module + "\ngo 1.23\n"
	dir := filepath.Join(module, "@v")
	writeFile(t, proxy, filepath.Join(dir, version+".mod"), mod)
	writeFile(t, proxy, filepath.Join(dir, version+".info"), `{"Version":"v1.0.0"}`)
	f, err := os.Create(filepath.Join(proxy, dir, version+".zip"))
	if err != nil {
		t.Fatal(err)
	}
	z := zip.NewWriter(f)
	for _, file := range []struct{ name, data string }{{"go.mod", mod}, {"ext.go", "package ext\nfunc F(){}\n"}} {
		entry, err := z.Create(module + "@" + version + "/" + file.name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write([]byte(file.data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	files := basicFiles()
	files["go.mod"] = "module example.com/m\ngo 1.23\nrequire example.com/original v1.0.0\nreplace example.com/original v1.0.0 => " + module + " " + version + "\n"
	files["a/a.go"] = "package a\nimport \"example.com/original\"\nfunc A(){ext.F()}\n"
	root := fixture(t, files)
	cmd := exec.Command("go", "mod", "download", module+"@"+version)
	cmd.Dir = root
	cmd.Env = append(offlineEnvironment(), "GOPROXY=file://"+filepath.ToSlash(proxy), "GOMODCACHE="+cache, "GOFLAGS=-modcacherw", "GOEXPERIMENT=")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("local prewarm: %v %s", err, out)
	}
	commit(t, root)
	t.Setenv("GOMODCACHE", cache)
	_, m, err := Analyze(context.Background(), root, "HEAD")
	if err != nil || m.Incomplete {
		t.Fatalf("remote replacement treated as local: %v %+v", err, m)
	}
}

func TestAnalyzeUnavailableReplacements(t *testing.T) {
	for _, replacement := range []string{"./missing", "example.com/replacement v1.0.0"} {
		t.Run(replacement, func(t *testing.T) {
			t.Setenv("GOMODCACHE", t.TempDir())
			files := basicFiles()
			files["go.mod"] = "module example.com/m\ngo 1.23\nrequire example.com/original v1.0.0\nreplace example.com/original => " + replacement + "\n"
			files["a/a.go"] = "package a\nimport _ \"example.com/original\"\nfunc A(){}\n"
			root := fixture(t, files)
			_, m, err := Analyze(context.Background(), root, "HEAD")
			if err == nil && !m.Incomplete {
				t.Fatal("unavailable replacement reported complete")
			}
		})
	}
}
