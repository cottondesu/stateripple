package adapters

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cottondesu/stateripple/internal/report"
	"github.com/cottondesu/stateripple/internal/snapshot"
)

type Metadata struct {
	Changes                  []report.FileChange
	UnknownChanges, Warnings []string
	BuildContext             *report.BuildContext
	Incomplete               bool
}

func (m *Metadata) warn(s string) { m.Incomplete = true; m.Warnings = append(m.Warnings, s) }

func Analyze(ctx context.Context, root, base string) (snapshot.Snapshot, Metadata, error) {
	m := Metadata{Changes: []report.FileChange{}, UnknownChanges: []string{}, Warnings: []string{}}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return snapshot.Snapshot{}, m, fmt.Errorf("root: %w", err)
	}
	root, err = filepath.EvalSymlinks(absolute)
	if err != nil {
		return snapshot.Snapshot{}, m, fmt.Errorf("root: %w", err)
	}
	if err := requireGit(ctx, root); err != nil {
		return snapshot.Snapshot{}, m, err
	}
	head, err := command(ctx, root, "git", "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return snapshot.Snapshot{}, m, fmt.Errorf("resolve HEAD: %w", err)
	}
	changes, err := gitChanges(ctx, root, base)
	if err != nil {
		return snapshot.Snapshot{}, m, err
	}
	m.Changes = changes
	inputs, err := headFiles(ctx, root, strings.TrimSpace(string(head)))
	if err != nil {
		return snapshot.Snapshot{}, m, err
	}
	if err := verifyHeadFiles(ctx, root, inputs, &m); err != nil {
		return snapshot.Snapshot{}, m, err
	}
	if work := os.Getenv("GOWORK"); work != "" && work != "off" {
		m.warn("explicit Go workspace environment is unsupported")
	}
	if _, err := os.Stat(filepath.Join(root, "go.work")); err == nil {
		m.warn("Go workspace is unsupported")
	} else if !os.IsNotExist(err) {
		return snapshot.Snapshot{}, m, fmt.Errorf("inspect workspace: %w", err)
	}
	tracked, err := command(ctx, root, "git", "ls-files", "-z")
	if err != nil {
		return snapshot.Snapshot{}, m, err
	}
	for _, p := range strings.Split(string(tracked), "\x00") {
		if p == "go.work" || p == "go.work.sum" {
			m.warn("Go workspace is unsupported")
		}
		if strings.HasSuffix(p, "/go.mod") {
			m.warn("nested Go modules are unsupported: " + p)
		}
	}
	s, packages, err := goGraph(ctx, root, &m)
	if err != nil {
		return s, m, err
	}
	trackedFiles := map[string]bool{}
	for _, path := range strings.Split(string(tracked), "\x00") {
		trackedFiles[path] = true
	}
	checkModuleInputs(root, trackedFiles, &m)
	for _, p := range packages {
		for _, file := range p.analysisFiles() {
			path, err := filepath.Rel(root, filepath.Join(p.Dir, file))
			if err != nil {
				return s, m, fmt.Errorf("resolve analyzed file: %w", err)
			}
			path = filepath.ToSlash(path)
			if !trackedFiles[path] {
				m.warn("analysis input is not committed: " + path)
			}
			info, err := os.Lstat(filepath.Join(root, path))
			if err != nil {
				m.warn("analysis input unavailable: " + path)
			} else if info.Mode()&os.ModeSymlink != 0 {
				m.warn("symbolic link analysis input unsupported: " + path)
			}
		}
	}
	patterns, err := formerEmbeds(ctx, root, base)
	if err != nil {
		m.warn("base embed analysis incomplete: " + err.Error())
	}
	var former map[string][]owner
	if err == nil {
		paths := []string{}
		for _, change := range m.Changes {
			paths = append(paths, change.Path)
		}
		former = formerEmbedOwners(root, packages, patterns, paths)
	}
	classify(root, &s, packages, &m, former)
	if err := verifyHeadFiles(ctx, root, inputs, &m); err != nil {
		return s, m, err
	}
	currentHead, err := command(ctx, root, "git", "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return s, m, err
	}
	if string(currentHead) != string(head) {
		return s, m, fmt.Errorf("HEAD changed during analysis")
	}
	normalized, err := snapshot.Normalize(s)
	if err != nil {
		return s, m, fmt.Errorf("normalize collected graph: %w", err)
	}
	if err := validateGoProfile(normalized); err != nil {
		return s, m, fmt.Errorf("collected graph violates Go profile: %w", err)
	}
	sort.Slice(m.Changes, func(i, j int) bool { return m.Changes[i].Path < m.Changes[j].Path })
	sort.Strings(m.UnknownChanges)
	m.UnknownChanges = unique(m.UnknownChanges)
	sort.Strings(m.Warnings)
	m.Warnings = unique(m.Warnings)
	return normalized, m, nil
}

func unique(items []string) []string {
	out := items[:0]
	for _, item := range items {
		if len(out) == 0 || out[len(out)-1] != item {
			out = append(out, item)
		}
	}
	return out
}

type owner struct {
	id             string
	risk           int
	classification string
}

func classify(root string, s *snapshot.Snapshot, packages []goPackage, m *Metadata, former map[string][]owner) {
	files := map[string][]owner{}
	dirs := map[string]goPackage{}
	for _, p := range packages {
		rel, err := filepath.Rel(root, p.Dir)
		if err != nil {
			m.warn("package directory cannot be resolved: " + p.ImportPath)
			continue
		}
		rel = filepath.ToSlash(rel)
		dirs[rel] = p
		groups := []struct {
			names  []string
			target owner
		}{
			{append(append([]string{}, p.GoFiles...), p.CgoFiles...), owner{"pkg:" + p.ImportPath, 3, "source"}},
			{p.nativeFiles(), owner{"pkg:" + p.ImportPath, 3, "native"}},
			{append(append([]string{}, p.TestGoFiles...), p.XTestGoFiles...), owner{"test:" + p.ImportPath, 2, "test"}},
			{p.EmbedFiles, owner{"pkg:" + p.ImportPath, 3, "embed"}},
			{append(append([]string{}, p.TestEmbedFiles...), p.XTestEmbedFiles...), owner{"test:" + p.ImportPath, 2, "test_embed"}},
		}
		for _, g := range groups {
			for _, name := range g.names {
				path := filepath.ToSlash(filepath.Join(rel, name))
				files[path] = append(files[path], g.target)
			}
		}
	}
	for i := range m.Changes {
		c := &m.Changes[i]
		if c.Classification == "gitlink" {
			c.Classification = "unknown"
			m.UnknownChanges = append(m.UnknownChanges, c.Path)
			m.Incomplete = true
			continue
		}
		targets := append(append([]owner{}, files[c.Path]...), former[c.Path]...)
		for _, target := range targets {
			if target.id == "" {
				m.warn("former embed owner unavailable: " + c.Path)
				m.UnknownChanges = append(m.UnknownChanges, c.Path)
			}
		}
		valid := targets[:0]
		for _, target := range targets {
			if target.id != "" {
				valid = append(valid, target)
			}
		}
		targets = valid
		switch {
		case c.Path == "go.mod" || c.Path == "go.sum":
			c.Classification = "module"
			for _, p := range packages {
				s.Seeds = append(s.Seeds, snapshot.Seed{ID: "pkg:" + p.ImportPath, Risk: 3, Reason: "module_changed:" + c.Path})
			}
			continue
		case len(targets) > 0:
		case strings.HasSuffix(c.Path, ".go") && c.Status == "D":
			dir := filepath.ToSlash(filepath.Dir(c.Path))
			p, ok := dirs[dir]
			if ok {
				if strings.HasSuffix(c.Path, "_test.go") {
					if hasTests(p) {
						targets = []owner{{"test:" + p.ImportPath, 2, "test"}}
					}
				} else {
					targets = []owner{{"pkg:" + p.ImportPath, 3, "source"}}
				}
			}
		case (strings.EqualFold(filepath.Ext(c.Path), ".md") || filepath.Base(c.Path) == "README") && len(former[c.Path]) == 0 && former != nil:
			c.Classification = "excluded_document"
			continue
		}
		if len(targets) == 0 {
			c.Classification = "unknown"
			m.UnknownChanges = append(m.UnknownChanges, c.Path)
			m.Incomplete = true
			continue
		}
		classes := []string{}
		for _, target := range targets {
			classes = append(classes, target.classification)
			s.Seeds = append(s.Seeds, snapshot.Seed{ID: target.id, Risk: target.risk, Reason: target.classification + "_changed:" + c.Path})
		}
		sort.Strings(classes)
		c.Classification = strings.Join(unique(classes), ",")
	}
}
