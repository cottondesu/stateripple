package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type inputFileReference struct {
	path string
	info os.FileInfo
}

type reportTempFile interface {
	io.Writer
	Chmod(os.FileMode) error
	Close() error
	Name() string
	Sync() error
}

type createReportTemp func(string, string) (reportTempFile, error)
type removeReportTemp func(string) error
type publishReportTemp func(string, string) error

func writeReportFile(path string, data []byte, input *inputFileReference) error {
	return writeReportFileWith(path, data, input, func(dir, pattern string) (reportTempFile, error) {
		return os.CreateTemp(dir, pattern)
	}, os.Remove, os.Link)
}

func writeReportFileWith(path string, data []byte, input *inputFileReference, createTemp createReportTemp, removeTemp removeReportTemp, publishTemp publishReportTemp) (err error) {
	if path == "" || os.IsPathSeparator(path[len(path)-1]) {
		return fmt.Errorf("output must name a file without a trailing path separator: %s", path)
	}
	if err := inspectOutput(path, input); err != nil {
		return err
	}
	if input != nil && sameCleanPath(path, input.path) {
		return fmt.Errorf("output aliases input: %s", path)
	}
	parent, name := filepath.Split(path)
	if parent == "" {
		parent = "."
	}
	dir, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return fmt.Errorf("resolve output directory: %w", err)
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolve output directory: %w", err)
	}
	dirInfo, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("inspect output directory: %w", err)
	}
	path = filepath.Join(dir, name)
	if err := inspectOutput(path, input); err != nil {
		return err
	}
	temp, err := createTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary output: %w", err)
	}
	tempName := temp.Name()
	published := false
	defer func() {
		removeErr := removeTemp(tempName)
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			if published {
				err = errors.Join(err, fmt.Errorf("output published at %s; remove temporary output %s: %w", path, tempName, removeErr))
			} else {
				err = errors.Join(err, fmt.Errorf("remove temporary output %s: %w", tempName, removeErr))
			}
		}
	}()
	if err := writeAll(temp, data); err != nil {
		return fmt.Errorf("write temporary output: %w", errors.Join(err, temp.Close()))
	}
	if err := temp.Chmod(0o600); err != nil {
		return fmt.Errorf("set temporary output permissions: %w", errors.Join(err, temp.Close()))
	}
	if err := temp.Sync(); err != nil {
		return fmt.Errorf("sync temporary output: %w", errors.Join(err, temp.Close()))
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary output: %w", err)
	}
	afterDir, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("recheck output directory: %w", err)
	}
	if !os.SameFile(dirInfo, afterDir) {
		return fmt.Errorf("output directory changed before publication: %s", dir)
	}
	if err := publishTemp(tempName, path); err != nil {
		return fmt.Errorf("publish new output without replacement: %w", err)
	}
	published = true
	return nil
}

func inspectOutput(path string, input *inputFileReference) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect output: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("output must not be a symbolic link: %s", path)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("output must be a regular file: %s", path)
	}
	if input != nil && os.SameFile(input.info, info) {
		return fmt.Errorf("output aliases input: %s", path)
	}
	return fmt.Errorf("output already exists; refusing to overwrite: %s", path)
}

func sameCleanPath(a, b string) bool {
	aPath, aErr := filepath.Abs(a)
	bPath, bErr := filepath.Abs(b)
	return aErr == nil && bErr == nil && filepath.Clean(aPath) == filepath.Clean(bPath)
}

func writeAll(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if n > 0 {
			data = data[n:]
		}
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}
