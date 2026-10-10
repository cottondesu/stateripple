package adapters

import (
	"context"
	"testing"
)

func TestAnalyzeNativeSourceChanges(t *testing.T) {
	for _, file := range []string{"code.s", "object.syso", "config.h"} {
		t.Run(file, func(t *testing.T) {
			files := basicFiles()
			files["a/"+file] = "before\n"
			root := fixture(t, files)
			writeFile(t, root, "a/"+file, "after\n")
			commit(t, root)
			s, m, err := Analyze(context.Background(), root, "")
			if err != nil || m.Incomplete || !seeded(s, "pkg:example.com/m/a", 3) {
				t.Fatalf("native ownership missing: err=%v meta=%+v seeds=%+v", err, m, s.Seeds)
			}
		})
	}
}

func TestAnalyzeHiddenAssemblyChanges(t *testing.T) {
	for _, kind := range []string{"ignored", "--assume-unchanged", "--skip-worktree"} {
		t.Run(kind, func(t *testing.T) {
			files := basicFiles()
			files[".gitignore"] = "a/code.s\n"
			if kind != "ignored" {
				files[".gitignore"] = ""
				files["a/code.s"] = "// before\n"
			}
			root := fixture(t, files)
			if kind != "ignored" {
				runGit(t, root, "update-index", kind, "a/code.s")
			}
			writeFile(t, root, "a/code.s", "// after\n")
			_, m, err := Analyze(context.Background(), root, "HEAD")
			if err == nil && !m.Incomplete {
				t.Fatal("uncommitted native input reported complete")
			}
		})
	}
}

func TestAnalyzeCgoHeaderChanges(t *testing.T) {
	t.Setenv("CGO_ENABLED", "1")
	files := basicFiles()
	files["a/native.go"] = "package a\n/*\n#include \"config.h\"\n*/\nimport \"C\"\n"
	files["a/config.h"] = "#define VALUE 1\n"
	root := fixture(t, files)
	writeFile(t, root, "a/config.h", "#define VALUE 2\n")
	commit(t, root)
	s, m, err := Analyze(context.Background(), root, "")
	if err != nil || m.Incomplete || !seeded(s, "pkg:example.com/m/a", 3) {
		t.Fatalf("cgo header not seeded: %v %+v %+v", err, m, s.Seeds)
	}
}

func TestAnalyzeUnknownNativeInputIsIncomplete(t *testing.T) {
	root := fixture(t, basicFiles())
	writeFile(t, root, "a/custom.native", "changed\n")
	commit(t, root)
	_, m, err := Analyze(context.Background(), root, "")
	if err != nil || !m.Incomplete {
		t.Fatalf("unknown native impact reported complete: %v %+v", err, m)
	}
}
