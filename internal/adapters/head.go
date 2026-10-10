package adapters

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cottondesu/stateripple/internal/fileio"
)

type headFile struct {
	path, mode, oid string
}

func headFiles(ctx context.Context, root, commit string) ([]headFile, error) {
	data, err := command(ctx, root, "git", "ls-tree", "-r", "-z", "--full-tree", commit)
	if err != nil {
		return nil, fmt.Errorf("read HEAD tree: %w", err)
	}
	files := []headFile{}
	for _, record := range strings.Split(string(data), "\x00") {
		if record == "" {
			continue
		}
		header, path, ok := strings.Cut(record, "\t")
		fields := strings.Fields(header)
		if !ok || len(fields) != 3 || path == "" {
			return nil, fmt.Errorf("invalid HEAD tree record")
		}
		files = append(files, headFile{path, fields[0], fields[2]})
	}
	return files, nil
}

func verifyHeadFiles(ctx context.Context, root string, files []headFile, meta *Metadata) error {
	buffer := make([]byte, 32<<10)
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("verify HEAD inputs: %w", err)
		}
		if file.mode == "160000" {
			continue
		}
		path := filepath.Join(root, filepath.FromSlash(file.path))
		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("inspect HEAD input %q: %w", file.path, err)
		}
		if file.mode == "120000" && info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Stat(path)
			if err != nil || !target.Mode().IsRegular() {
				return fmt.Errorf("nonregular symbolic HEAD input %q", file.path)
			}
			meta.warn("symbolic link analysis input unsupported: " + file.path)
			continue
		}
		if !info.Mode().IsRegular() || (file.mode != "100644" && file.mode != "100755") {
			return fmt.Errorf("HEAD input type mismatch: %q", file.path)
		}
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return fmt.Errorf("resolve HEAD input %q: %w", file.path, err)
		}
		if resolved != path {
			meta.warn("symbolic link analysis path unsupported: " + file.path)
		}
		if runtime.GOOS != "windows" && (info.Mode().Perm()&0111 != 0) != (file.mode == "100755") {
			return fmt.Errorf("HEAD input executable mode mismatch: %q", file.path)
		}
		if err := verifyBlob(path, file, info, buffer); err != nil {
			return err
		}
	}
	return nil
}

func verifyBlob(path string, file headFile, expected os.FileInfo, buffer []byte) error {
	f, _, err := fileio.OpenRegular(path)
	if err != nil {
		return fmt.Errorf("open HEAD input %q: %w", file.path, err)
	}
	checkErr := func() error {
		info, err := f.Stat()
		if err != nil {
			return fmt.Errorf("stat input: %w", err)
		}
		if !info.Mode().IsRegular() || !os.SameFile(expected, info) {
			return fmt.Errorf("input identity changed")
		}
		var digest hash.Hash
		switch len(file.oid) {
		case 40:
			digest = sha1.New()
		case 64:
			digest = sha256.New()
		default:
			return fmt.Errorf("unsupported Git object ID")
		}
		if _, err := fmt.Fprintf(digest, "blob %d\x00", info.Size()); err != nil {
			return fmt.Errorf("hash header: %w", err)
		}
		if _, err := io.CopyBuffer(digest, io.LimitReader(f, info.Size()+1), buffer); err != nil {
			return fmt.Errorf("hash bytes: %w", err)
		}
		if hex.EncodeToString(digest.Sum(nil)) != file.oid {
			return fmt.Errorf("worktree bytes differ from HEAD (including filters or line-ending conversion)")
		}
		return nil
	}()
	if err := errors.Join(checkErr, f.Close()); err != nil {
		return fmt.Errorf("verify HEAD input %q: %w", file.path, err)
	}
	return nil
}
