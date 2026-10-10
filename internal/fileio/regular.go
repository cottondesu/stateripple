package fileio

import (
	"errors"
	"fmt"
	"os"
)

// OpenRegular opens path only when its pre-open and opened identities match regular files.
func OpenRegular(path string) (*os.File, os.FileInfo, error) {
	before, err := os.Stat(path)
	if err != nil {
		return nil, nil, err
	}
	if !before.Mode().IsRegular() {
		return nil, nil, fmt.Errorf("input must be a regular file: %s", path)
	}
	f, err := openFile(path)
	if err != nil {
		return nil, nil, err
	}
	info, statErr := f.Stat()
	if statErr != nil || !info.Mode().IsRegular() || !os.SameFile(before, info) {
		closeErr := f.Close()
		if statErr != nil {
			return nil, nil, errors.Join(statErr, closeErr)
		}
		if !info.Mode().IsRegular() {
			return nil, nil, errors.Join(fmt.Errorf("input must be a regular file: %s", path), closeErr)
		}
		return nil, nil, errors.Join(fmt.Errorf("input changed while opening: %s", path), closeErr)
	}
	return f, info, nil
}
