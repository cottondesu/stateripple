//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package adapters

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// gitPolicyKeys are the inherited Git settings that TestGitMissingBlobNeverFetches
// controls. Each case starts with all of them unset, so the result does not
// depend on the developer's or the CI runner's environment.
var gitPolicyKeys = []string{"GIT_NO_LAZY_FETCH", "GIT_TERMINAL_PROMPT", "GIT_OPTIONAL_LOCKS", "GIT_TRACE", "GIT_TRACE2", "GIT_TRACE_PACKET"}

func TestGitMissingBlobNeverFetches(t *testing.T) {
	for _, tc := range []struct {
		name string
		env  []string
	}{
		{"default", nil},
		{"no-lazy-fetch-1", []string{"GIT_NO_LAZY_FETCH=1"}},
		{"no-lazy-fetch-0", []string{"GIT_NO_LAZY_FETCH=0"}},
		{"no-lazy-fetch-empty", []string{"GIT_NO_LAZY_FETCH="}},
		{"terminal-prompt-0", []string{"GIT_TERMINAL_PROMPT=0"}},
		{"trace", []string{"GIT_TRACE=1", "GIT_TRACE2=1", "GIT_TRACE_PACKET=1"}},
		{"hostile", []string{"GIT_NO_LAZY_FETCH=0", "GIT_TERMINAL_PROMPT=1", "GIT_OPTIONAL_LOCKS=1", "GIT_TRACE=1"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given: the caller's Git policy is in place before the fixture is
			// built, so fixture construction is exercised under it as well. Every
			// Git child, fixture or product, is confined to the local file transport.
			for _, key := range gitPolicyKeys {
				unsetenv(t, key)
			}
			for _, item := range tc.env {
				key, value, _ := strings.Cut(item, "=")
				t.Setenv(key, value)
			}
			t.Setenv("GIT_ALLOW_PROTOCOL", "file")
			root, missing, canary := partialClone(t)
			beforeMissing := missingObjects(t, root)
			beforeStore := objectStore(t, root)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			// When
			_, meta, err := Analyze(ctx, root, "HEAD^")
			// Then: useful incomplete result naming every missing base source, no
			// upload-pack attempt, no object acquisition. The present base source
			// with an embed directive (c/c.go) must not hide the missing ones.
			if err != nil || !meta.Incomplete {
				t.Fatalf("missing base blob must give a useful incomplete result: err=%v meta=%+v", err, meta)
			}
			warnings := strings.Join(meta.Warnings, "\n")
			for _, path := range []string{"a/a.go", "b/b.go"} {
				if !strings.Contains(warnings, fmt.Sprintf("read base Go source %q", path)) {
					t.Fatalf("missing base source %s absent from diagnostics: %v", path, meta.Warnings)
				}
			}
			if strings.Contains(warnings, `read base Go source "c/c.go"`) {
				t.Fatalf("present base source reported as unreadable: %v", meta.Warnings)
			}
			if data := readCanary(t, canary); len(data) != 0 {
				t.Fatalf("implicit fetch attempted: %s", data)
			}
			if after := missingObjects(t, root); beforeMissing != after {
				t.Fatalf("missing objects changed: before=%s after=%s", beforeMissing, after)
			}
			if after := objectStore(t, root); beforeStore != after {
				t.Fatalf("object store changed: before=%s after=%s", beforeStore, after)
			}
			// Control: the canary is live. The same read with lazy fetching allowed
			// asks the promisor remote, so an unguarded product read would have
			// been recorded above. The shim transfers nothing.
			control := exec.Command("git", "cat-file", "blob", missing[0])
			control.Dir = root
			control.Env = fixtureGitEnvironment("GIT_NO_LAZY_FETCH=0")
			if out, err := control.CombinedOutput(); err == nil {
				t.Fatalf("fixture control: missing blob %s is readable: %s", missing[0], out)
			}
			if len(readCanary(t, canary)) == 0 {
				t.Fatal("fixture control: an allowed lazy fetch did not reach the canary; a zero count proves nothing")
			}
			if after := missingObjects(t, root); beforeMissing != after {
				t.Fatalf("fixture control: canary transferred objects: before=%s after=%s", beforeMissing, after)
			}
		})
	}
}

// partialClone returns a blob:none partial clone of a local origin whose HEAD
// rewrites a/a.go and b/b.go, the object IDs of their base (HEAD^) blobs, which
// are the only objects absent from the clone, and the log written by a canary
// installed as the promisor remote's upload-pack.
//
// Construction does not rely on Git's implicit promisor fetch: the clone
// transfers commits and trees without a checkout, the HEAD blobs are copied
// from origin as an explicit local pack, and the checkout then runs with lazy
// fetching forbidden. Every failure here is reported as "fixture setup", never
// as a product assertion.
func partialClone(t *testing.T) (root string, missing []string, canary string) {
	t.Helper()
	if err := requireGit(context.Background(), t.TempDir()); err != nil {
		t.Fatalf("fixture setup: the partial-clone fixture needs the Git that analyze requires: %v", err)
	}
	files := basicFiles()
	files["a/a.go"] = "package a\nimport _ \"embed\"\n//go:embed old.txt\nvar Data string\n"
	files["a/old.txt"] = "old embed\n"
	files["c/c.go"] = "package c\nimport _ \"embed\"\n//go:embed c.txt\nvar Data string\n"
	files["c/c.txt"] = "kept embed\n"
	origin := fixture(t, files)
	for _, path := range []string{"a/a.go", "b/b.go"} {
		missing = append(missing, strings.TrimSpace(string(fixtureGit(t, origin, nil, "rev-parse", "--verify", "HEAD:"+path))))
	}
	sort.Strings(missing)
	writeFile(t, origin, "a/a.go", "package a\nfunc A() {}\n")
	writeFile(t, origin, "b/b.go", files["b/b.go"]+"// changed\n")
	commit(t, origin)
	fixtureGit(t, origin, nil, "config", "uploadpack.allowFilter", "true")

	root = filepath.Join(t.TempDir(), "partial")
	fixtureGit(t, origin, nil, "clone", "-q", "--no-checkout", "--filter=blob:none", "file://"+origin, root)
	var head bytes.Buffer
	for _, record := range strings.Split(string(fixtureGit(t, root, nil, "ls-tree", "-r", "-z", "--full-tree", "HEAD")), "\x00") {
		if record == "" {
			continue
		}
		header, _, _ := strings.Cut(record, "\t")
		fields := strings.Fields(header)
		if len(fields) != 3 || fields[1] != "blob" {
			t.Fatalf("fixture setup: unexpected HEAD tree record %q", record)
		}
		head.WriteString(fields[2] + "\n")
	}
	pack := fixtureGit(t, origin, head.Bytes(), "pack-objects", "-q", "--stdout")
	fixtureGit(t, root, pack, "unpack-objects", "-q")
	fixtureGit(t, root, nil, "checkout", "-q", "HEAD", "--", ".")

	if status := fixtureGit(t, root, nil, "status", "--porcelain=v1", "--untracked-files=all"); len(status) != 0 {
		t.Fatalf("fixture setup: checkout is not clean: %s", status)
	}
	if promisor := strings.TrimSpace(string(fixtureGit(t, root, nil, "config", "--get", "remote.origin.promisor"))); promisor != "true" {
		t.Fatalf("fixture setup: origin is not a promisor remote: %q", promisor)
	}
	absent := []string{}
	for _, line := range strings.Split(missingObjects(t, root), "\n") {
		if oid, ok := strings.CutPrefix(line, "?"); ok {
			absent = append(absent, oid)
		}
	}
	sort.Strings(absent)
	if strings.Join(absent, " ") != strings.Join(missing, " ") {
		t.Fatalf("fixture setup: absent objects %v, want exactly the base blobs %v", absent, missing)
	}

	shim := t.TempDir()
	canary = filepath.Join(shim, "fetch.log")
	writeFile(t, shim, "upload-pack", "#!/bin/sh\nprintf 'fetch\\n' >> \""+canary+"\"\nexit 86\n")
	if err := os.Chmod(filepath.Join(shim, "upload-pack"), 0o700); err != nil {
		t.Fatalf("fixture setup: %v", err)
	}
	fixtureGit(t, root, nil, "config", "remote.origin.uploadpack", filepath.Join(shim, "upload-pack"))
	return root, missing, canary
}

// fixtureGitEnvironment is the caller's environment with the Git policy pinned
// for fixture work: local file transport only, no prompt, no trace, and the
// given settings (lazy fetching is forbidden unless a setting says otherwise).
func fixtureGitEnvironment(settings ...string) []string {
	env := []string{}
	for _, v := range os.Environ() {
		key, _, _ := strings.Cut(v, "=")
		switch {
		case key == "GIT_NO_LAZY_FETCH", key == "GIT_TERMINAL_PROMPT", key == "GIT_ALLOW_PROTOCOL", key == "GIT_CONFIG_NOSYSTEM", strings.HasPrefix(key, "GIT_TRACE"):
		default:
			env = append(env, v)
		}
	}
	env = append(env, "GIT_NO_LAZY_FETCH=1", "GIT_TERMINAL_PROMPT=0", "GIT_ALLOW_PROTOCOL=file", "GIT_CONFIG_NOSYSTEM=1")
	return append(env, settings...)
}

func fixtureGit(t *testing.T, dir string, stdin []byte, args ...string) []byte {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = fixtureGitEnvironment()
	cmd.Stdin = bytes.NewReader(stdin)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("fixture setup: git %v: %v: %s", args, err, stderr.String())
	}
	return out
}

func unsetenv(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "") // restores the caller's value when the test ends
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
}

func readCanary(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return data
}

func missingObjects(t *testing.T, root string) string {
	t.Helper()
	cmd := exec.Command("git", "--no-lazy-fetch", "rev-list", "--objects", "--missing=print", "--all")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// objectStore fingerprints every file below .git/objects.
func objectStore(t *testing.T, root string) string {
	t.Helper()
	dir := filepath.Join(root, ".git", "objects")
	var b strings.Builder
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		fmt.Fprintf(&b, "%s %x\n", relative, sha256.Sum256(data))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func TestGitChildrenRejectPromptAndOverrideInheritedEnvironment(t *testing.T) {
	// Given: a shim models a credential prompt unless the child forbids it and
	// lazy fetching, both by option and by environment.
	shim := t.TempDir()
	writeFile(t, shim, "git", "#!/bin/sh\nif [ \"$1\" != --no-lazy-fetch ] || [ \"$GIT_NO_LAZY_FETCH\" != 1 ] || [ \"$GIT_TERMINAL_PROMPT\" != 0 ]; then printf 'prompt would block\\n'; exit 85; fi\nprintf 'authentication unavailable\\n' >&2\nexit 86\n")
	if err := os.Chmod(filepath.Join(shim, "git"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", shim+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("GIT_NO_LAZY_FETCH", "0")
	t.Setenv("GIT_TERMINAL_PROMPT", "1")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// When
	out, err := command(ctx, shim, "git", "show", "HEAD:a.go")
	// Then
	if err == nil || len(out) != 0 || !strings.Contains(err.Error(), "authentication unavailable") {
		t.Fatalf("unsafe prompt: out=%s err=%v", out, err)
	}
	if os.Getenv("GIT_NO_LAZY_FETCH") != "0" || os.Getenv("GIT_TERMINAL_PROMPT") != "1" {
		t.Fatal("parent environment changed")
	}
}

func TestFormerEmbedsRejectsGitReadDiagnostic(t *testing.T) {
	root := fixture(t, basicFiles())
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	shim := t.TempDir()
	writeFile(t, shim, "git", "#!/bin/sh\nfor arg in \"$@\"; do if [ \"$arg\" = cat-file ]; then printf 'missing blob\\n' >&2; exit 1; fi; done\nexec \"$STATERIPPLE_REAL_GIT\" \"$@\"\n")
	if err := os.Chmod(filepath.Join(shim, "git"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", shim+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("STATERIPPLE_REAL_GIT", git)
	_, err = formerEmbeds(context.Background(), root, "HEAD")
	if err == nil {
		t.Fatal("Git read error accepted as no embed matches")
	}
}
