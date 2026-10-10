package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cottondesu/stateripple/internal/graph"
	"github.com/cottondesu/stateripple/internal/report"
	"github.com/cottondesu/stateripple/internal/snapshot"
)

type goModule struct {
	Path, Dir, Version string
	Main               bool
	Replace            *goModule
}
type packageError struct{ Err string }
type goPackage struct {
	ImportPath, Dir, ForTest, Name                                                                                                string
	GoFiles, CgoFiles, TestGoFiles, XTestGoFiles, Imports, TestImports, XTestImports, EmbedFiles, TestEmbedFiles, XTestEmbedFiles []string
	SFiles, CFiles, HFiles, CXXFiles, MFiles, FFiles, SwigFiles, SwigCXXFiles, SysoFiles                                          []string
	Incomplete                                                                                                                    bool
	Error                                                                                                                         *packageError
	DepsErrors                                                                                                                    []packageError
	Module                                                                                                                        *goModule
}

func goGraph(ctx context.Context, root string, meta *Metadata) (snapshot.Snapshot, []goPackage, error) {
	s := snapshot.Snapshot{SchemaVersion: 1, Ruleset: "risk-propagation-v1", MaxGenerations: 16, Nodes: []graph.Node{}, Edges: []graph.Edge{}, Seeds: []snapshot.Seed{}}
	envData, err := command(ctx, root, "go", "env", "-json", "GOVERSION", "GOOS", "GOARCH", "CGO_ENABLED", "GOMOD", "GOFLAGS", "GOEXPERIMENT")
	if err != nil {
		return s, nil, err
	}
	var env map[string]string
	if err = json.Unmarshal(envData, &env); err != nil {
		return s, nil, fmt.Errorf("decode Go environment: %w", err)
	}
	moduleFile, err := filepath.EvalSymlinks(env["GOMOD"])
	if err != nil {
		return s, nil, fmt.Errorf("resolve module file: %w", err)
	}
	expectedModuleFile, err := filepath.EvalSymlinks(filepath.Join(root, "go.mod"))
	if err != nil {
		return s, nil, fmt.Errorf("resolve root module file: %w", err)
	}
	if moduleFile != expectedModuleFile {
		return s, nil, fmt.Errorf("single module rooted at repository root required")
	}
	meta.BuildContext = &report.BuildContext{GoVersion: env["GOVERSION"], GOOS: env["GOOS"], GOARCH: env["GOARCH"], CGOEnabled: env["CGO_ENABLED"], BuildTags: []string{}, ToolTags: []string{}, ReleaseTags: []string{}}
	if os.Getenv("GOFLAGS") != "" || os.Getenv("GOEXPERIMENT") != "" {
		meta.warn("custom GOFLAGS or GOEXPERIMENT is unsupported")
	}
	contextData, err := command(ctx, root, "go", "list", "-mod=readonly", "-f", `{{join context.ToolTags ","}}|{{join context.ReleaseTags ","}}|{{join context.BuildTags ","}}`, "runtime")
	if err != nil {
		return s, nil, fmt.Errorf("Go build context: %w", err)
	}
	parts := strings.Split(strings.TrimSpace(string(contextData)), "|")
	if len(parts) != 3 {
		return s, nil, fmt.Errorf("invalid Go build context")
	}
	if parts[0] != "" {
		meta.BuildContext.ToolTags = strings.Split(parts[0], ",")
	}
	if parts[1] != "" {
		meta.BuildContext.ReleaseTags = strings.Split(parts[1], ",")
	}
	if parts[2] != "" {
		meta.BuildContext.BuildTags = strings.Split(parts[2], ",")
	}
	modData, err := command(ctx, root, "go", "list", "-m", "-json", "-mod=readonly")
	if err != nil {
		return s, nil, err
	}
	var mod goModule
	if err = json.Unmarshal(modData, &mod); err != nil {
		return s, nil, fmt.Errorf("decode module: %w", err)
	}
	meta.BuildContext.Module = mod.Path
	data, listErr := command(ctx, root, "go", "list", "-e", "-json", "-deps", "-test", "-mod=readonly", "./...")
	if listErr != nil {
		meta.warn(listErr.Error())
	}
	packages := []goPackage{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var p goPackage
		err := decoder.Decode(&p)
		if err == io.EOF {
			break
		}
		if err != nil {
			return s, nil, fmt.Errorf("decode go list: %w", err)
		}
		if p.Incomplete || p.Error != nil || len(p.DepsErrors) > 0 {
			meta.warn("Go dependency analysis incomplete: " + p.ImportPath)
		}
		if p.Module != nil && p.Module.Replace != nil && p.Module.Replace.Version == "" {
			meta.warn("local module replacements are unsupported: " + p.Module.Path)
		}
		if p.ForTest != "" || strings.Contains(p.ImportPath, " [") || (strings.HasSuffix(p.ImportPath, ".test") && p.Name == "main" && len(p.GoFiles) == 1 && filepath.IsAbs(p.GoFiles[0])) {
			continue
		}
		if p.Module != nil && p.Module.Main && p.Module.Path == mod.Path {
			resolvedDir, err := filepath.EvalSymlinks(p.Dir)
			if err != nil {
				return s, nil, fmt.Errorf("resolve package directory: %w", err)
			}
			p.Dir = resolvedDir
			files := append(append(append(append([]string{}, p.GoFiles...), p.CgoFiles...), p.TestGoFiles...), p.XTestGoFiles...)
			for _, file := range files {
				if _, err := parser.ParseFile(token.NewFileSet(), filepath.Join(p.Dir, file), nil, parser.AllErrors); err != nil {
					meta.warn("Go syntax analysis incomplete: " + p.ImportPath + "/" + file)
				}
			}
			packages = append(packages, p)
		}
	}
	if len(packages) == 0 && listErr != nil {
		return s, nil, fmt.Errorf("Go analysis failed without packages: %w", listErr)
	}
	known := map[string]bool{}
	for _, p := range packages {
		known[p.ImportPath] = true
		s.Nodes = append(s.Nodes, graph.Node{ID: "pkg:" + p.ImportPath, Kind: "package"})
		if hasTests(p) {
			s.Nodes = append(s.Nodes, graph.Node{ID: "test:" + p.ImportPath, Kind: "test"})
		}
	}
	for _, p := range packages {
		for _, dep := range p.Imports {
			if known[dep] {
				s.Edges = append(s.Edges, graph.Edge{From: "pkg:" + dep, To: "pkg:" + p.ImportPath, Kind: "prod_import", Attenuation: 1})
			}
		}
		if hasTests(p) {
			s.Edges = append(s.Edges, graph.Edge{From: "pkg:" + p.ImportPath, To: "test:" + p.ImportPath, Kind: "own_test"})
			for _, dep := range append(append([]string{}, p.TestImports...), p.XTestImports...) {
				if known[dep] && dep != p.ImportPath {
					s.Edges = append(s.Edges, graph.Edge{From: "pkg:" + dep, To: "test:" + p.ImportPath, Kind: "test_import", Attenuation: 1})
				}
			}
		}
	}
	return s, packages, nil
}

func hasTests(p goPackage) bool { return len(p.TestGoFiles)+len(p.XTestGoFiles) > 0 }
