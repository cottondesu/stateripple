package adapters

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeGitlinkCannotBeHidden(t *testing.T) {
	for _, setting := range []string{"default", "environment", "local", "command"} {
		t.Run(setting, func(t *testing.T) {
			root := fixture(t, basicFiles())
			head := exec.Command("git", "rev-parse", "HEAD")
			head.Dir = root
			old, err := head.Output()
			if err != nil {
				t.Fatal(err)
			}
			runGit(t, root, "update-index", "--add", "--cacheinfo", "160000,"+strings.TrimSpace(string(old))+",dep.md")
			runGit(t, root, "commit", "-qm", "first gitlink")
			newOID := exec.Command("git", "rev-parse", "HEAD")
			newOID.Dir = root
			current, err := newOID.Output()
			if err != nil {
				t.Fatal(err)
			}
			runGit(t, root, "update-index", "--cacheinfo", "160000,"+strings.TrimSpace(string(current))+",dep.md")
			runGit(t, root, "commit", "-qm", "changed gitlink")
			if err := os.Mkdir(filepath.Join(root, "dep.md"), 0755); err != nil {
				t.Fatal(err)
			}
			switch setting {
			case "local":
				runGit(t, root, "config", "diff.ignoreSubmodules", "all")
			case "environment", "command":
				t.Setenv("GIT_CONFIG_COUNT", "1")
				t.Setenv("GIT_CONFIG_KEY_0", "diff.ignoreSubmodules")
				t.Setenv("GIT_CONFIG_VALUE_0", "all")
			}
			if setting == "command" {
				data, err := command(context.Background(), root, "git", "-c", "diff.ignoreSubmodules=all", "diff", "--raw", "--ignore-submodules=none", "HEAD~1", "HEAD", "--")
				if err != nil || len(data) == 0 {
					t.Fatalf("explicit override did not expose gitlink: %v", err)
				}
			}
			_, m, err := Analyze(context.Background(), root, "")
			if err != nil || !m.Incomplete || len(m.UnknownChanges) != 1 || m.UnknownChanges[0] != "dep.md" {
				t.Fatalf("gitlink hidden or excluded as document: %v %+v", err, m)
			}
		})
	}
}
