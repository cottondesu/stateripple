package adapters

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/cottondesu/stateripple/internal/report"
	"github.com/cottondesu/stateripple/internal/snapshot"
)

func TestSuppressionSettingsCannotHideAssemblyMismatch(t *testing.T) {
	files := basicFiles()
	files["a/code.s"] = "// committed\n"
	root := fixture(t, files)
	runGit(t, root, "config", "diff.ignoreSubmodules", "all")
	runGit(t, root, "update-index", "--assume-unchanged", "a/code.s")
	writeFile(t, root, "a/code.s", "// hidden mutation\n")
	_, m, err := Analyze(context.Background(), root, "HEAD")
	if err == nil && !m.Incomplete {
		t.Fatal("suppressed native mismatch reported complete")
	}
}

func TestNativeInputsAllHaveProductionOwnership(t *testing.T) {
	root := t.TempDir()
	p := goPackage{ImportPath: "example.com/m/a", Dir: filepath.Join(root, "a"),
		SFiles: []string{"s"}, CFiles: []string{"c"}, HFiles: []string{"h"},
		CXXFiles: []string{"cxx"}, MFiles: []string{"m"}, FFiles: []string{"f"},
		SwigFiles: []string{"swig"}, SwigCXXFiles: []string{"swigcxx"}, SysoFiles: []string{"syso"}}
	for _, path := range p.nativeFiles() {
		m := Metadata{}
		m.Changes = append(m.Changes, report.FileChange{Path: "a/" + path, Status: "M"})
		s := snapshot.Snapshot{}
		classify(root, &s, []goPackage{p}, &m, nil)
		if m.Incomplete || !seeded(s, "pkg:example.com/m/a", 3) || m.Changes[0].Classification != "native" {
			t.Fatalf("field %s: %+v %+v", path, m, s.Seeds)
		}
	}
}
