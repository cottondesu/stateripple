package adapters

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAnalyzeRejectsHiddenHeadMismatch(t *testing.T) {
	for _, flag := range []string{"--assume-unchanged", "--skip-worktree"} {
		t.Run(flag, func(t *testing.T) {
			root := fixture(t, basicFiles())
			runGit(t, root, "update-index", flag, "b/b.go")
			before, err := command(context.Background(), root, "git", "ls-files", "-v")
			if err != nil {
				t.Fatal(err)
			}
			writeFile(t, root, "b/b.go", "package b\nfunc B(){}\n")
			_, m, err := Analyze(context.Background(), root, "HEAD")
			if err == nil && !m.Incomplete {
				t.Fatal("non-HEAD graph reported complete")
			}
			after, flagErr := command(context.Background(), root, "git", "ls-files", "-v")
			if flagErr != nil || string(after) != string(before) {
				t.Fatal("analysis modified index flags")
			}
		})
	}
}

func TestAnalyzeRejectsHiddenExecutableModeMismatch(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows FileMode does not represent Git executable mode")
	}
	root := fixture(t, basicFiles())
	runGit(t, root, "update-index", "--assume-unchanged", "a/a.go")
	if err := os.Chmod(filepath.Join(root, "a/a.go"), 0755); err != nil {
		t.Fatal(err)
	}
	_, m, err := Analyze(context.Background(), root, "HEAD")
	if err == nil && !m.Incomplete {
		t.Fatal("non-HEAD mode reported complete")
	}
}

func TestAnalyzeRejectsTransformedWorktreeBytes(t *testing.T) {
	files := basicFiles()
	files[".gitattributes"] = "a/a.go text eol=crlf\n"
	files["a/a.go"] = "package a\r\nfunc A(){}\r\n"
	root := fixture(t, files)
	_, m, err := Analyze(context.Background(), root, "HEAD")
	if err == nil && !m.Incomplete {
		t.Fatal("transformed source reported as exact HEAD input")
	}
}
