package adapters

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

const minimumGit = "2.45.0"

func requireGit(ctx context.Context, root string) error {
	data, err := command(ctx, root, "git", "--version")
	if err != nil {
		return fmt.Errorf("analyze requires Git %s or later with --no-lazy-fetch (no offline fallback): %w", minimumGit, err)
	}
	fields := strings.Fields(string(data))
	if len(fields) < 3 || fields[0] != "git" || fields[1] != "version" {
		return fmt.Errorf("analyze requires Git %s or later; unrecognized Git version", minimumGit)
	}
	parts := strings.Split(fields[2], ".")
	if len(parts) < 2 {
		return fmt.Errorf("analyze requires Git %s or later; unrecognized Git version", minimumGit)
	}
	major, majorErr := strconv.Atoi(parts[0])
	minor, minorErr := strconv.Atoi(parts[1])
	if majorErr != nil || minorErr != nil || major < 2 || (major == 2 && minor < 45) {
		return fmt.Errorf("analyze requires Git %s or later with --no-lazy-fetch; installed %s", minimumGit, fields[2])
	}
	return nil
}
