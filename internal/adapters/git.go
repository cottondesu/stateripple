package adapters

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cottondesu/stateripple/internal/report"
)

func gitChanges(ctx context.Context, root, base string) ([]report.FileChange, error) {
	top, err := command(ctx, root, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, err
	}
	canonicalTop, err := filepath.EvalSymlinks(strings.TrimSpace(string(top)))
	if err != nil {
		return nil, fmt.Errorf("resolve repository root: %w", err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve analysis root: %w", err)
	}
	if canonicalTop != canonicalRoot {
		return nil, fmt.Errorf("analysis must run at repository root")
	}
	status, err := command(ctx, root, "git", "status", "--porcelain=v1", "-z", "--untracked-files=all", "--ignore-submodules=none")
	if err != nil {
		return nil, err
	}
	if len(status) != 0 {
		return nil, fmt.Errorf("worktree contains tracked changes or untracked files")
	}
	if base == "" {
		base = "HEAD^"
	}
	rev, err := command(ctx, root, "git", "rev-parse", "--verify", "--end-of-options", base+"^{commit}")
	if err != nil {
		return nil, fmt.Errorf("resolve base: %w", err)
	}
	diff, err := command(ctx, root, "git", "diff", "--no-renames", "--raw", "--no-abbrev", "-z", "--no-ext-diff", "--no-textconv", "--ignore-submodules=none", strings.TrimSpace(string(rev)), "HEAD", "--")
	if err != nil {
		return nil, err
	}
	return parseRawDiff(diff)
}

func parseRawDiff(data []byte) ([]report.FileChange, error) {
	result := []report.FileChange{}
	if len(data) == 0 {
		return result, nil
	}
	tokens := strings.Split(string(data), "\x00")
	if tokens[len(tokens)-1] != "" || len(tokens)%2 != 1 {
		return nil, fmt.Errorf("malformed NUL Git raw diff")
	}
	for i := 0; i < len(tokens)-1; i += 2 {
		fields := strings.Fields(tokens[i])
		if len(fields) != 5 || !strings.HasPrefix(fields[0], ":") {
			return nil, fmt.Errorf("malformed Git raw diff header")
		}
		changes, err := parseDiff([]byte(fields[4] + "\x00" + tokens[i+1] + "\x00"))
		if err != nil {
			return nil, err
		}
		change := changes[0]
		if fields[0] == ":160000" || fields[1] == "160000" {
			change.Classification = "gitlink"
		}
		result = append(result, change)
	}
	return result, nil
}

func parseDiff(data []byte) ([]report.FileChange, error) {
	result := []report.FileChange{}
	if len(data) == 0 {
		return result, nil
	}
	tokens := strings.Split(string(data), "\x00")
	if tokens[len(tokens)-1] != "" || len(tokens)%2 != 1 {
		return nil, fmt.Errorf("malformed NUL Git diff")
	}
	for i := 0; i < len(tokens)-1; i += 2 {
		switch tokens[i] {
		case "A", "M", "D", "T":
		default:
			return nil, fmt.Errorf("unsupported Git change status %q", tokens[i])
		}
		if tokens[i+1] == "" {
			return nil, fmt.Errorf("empty Git change path")
		}
		result = append(result, report.FileChange{Path: tokens[i+1], Status: tokens[i]})
	}
	return result, nil
}
