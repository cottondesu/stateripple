//go:build !aix && !darwin && !dragonfly && !freebsd && !illumos && !linux && !netbsd && !openbsd && !solaris

package fileio

import "os"

func openFile(path string) (*os.File, error) {
	return os.Open(path)
}
