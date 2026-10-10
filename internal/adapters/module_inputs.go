package adapters

import (
	"os"
	"path/filepath"
)

func checkModuleInputs(root string, tracked map[string]bool, m *Metadata) {
	for _, name := range []string{"go.mod", "go.sum"} {
		info, err := os.Lstat(filepath.Join(root, name))
		if os.IsNotExist(err) && name == "go.sum" {
			continue
		}
		if err != nil {
			m.warn("module input unavailable: " + name)
			continue
		}
		if !tracked[name] {
			m.warn("module input is not committed: " + name)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			m.warn("symbolic link module input unsupported: " + name)
		}
	}
}
