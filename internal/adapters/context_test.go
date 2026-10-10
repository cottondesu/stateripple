package adapters

import (
	"context"
	"os"
	"testing"
)

func TestAnalyzeCustomFlags(t *testing.T) {
	root := fixture(t, basicFiles())
	t.Setenv("GOFLAGS", "-tags=custom")
	_, m, err := Analyze(context.Background(), root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if !m.Incomplete {
		t.Fatal("custom flags were not reported")
	}
	if len(m.BuildContext.BuildTags) != 0 {
		t.Fatal("unsupported tags were applied")
	}
}

func TestAnalyzeExplicitWorkspaceEnvironment(t *testing.T) {
	root := fixture(t, basicFiles())
	t.Setenv("GOWORK", "/nonexistent/external/go.work")
	_, m, err := Analyze(context.Background(), root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if !m.Incomplete {
		t.Fatal("external workspace configuration reported complete")
	}
}

func TestAnalyzeSumChanged(t *testing.T) {
	root := fixture(t, basicFiles())
	writeFile(t, root, "go.sum", "\n")
	commit(t, root)
	s, m, err := Analyze(context.Background(), root, "")
	if err != nil || m.Incomplete {
		t.Fatalf("%v %+v", err, m)
	}
	for _, id := range []string{"pkg:example.com/m/a", "pkg:example.com/m/b"} {
		if !seeded(s, id, 3) {
			t.Fatal("missing " + id)
		}
	}
}

func TestAnalyzeRealDotTestPackage(t *testing.T) {
	files := basicFiles()
	files["real.test/file.go"] = "package real\n"
	root := fixture(t, files)
	s, m, err := Analyze(context.Background(), root, "HEAD")
	if err != nil || m.Incomplete {
		t.Fatalf("%v %+v", err, m)
	}
	found := false
	for _, n := range s.Nodes {
		if n.ID == "pkg:example.com/m/real.test" {
			found = true
		}
	}
	if !found {
		t.Fatal("real .test package omitted")
	}
}

func TestAnalyzeIgnoredSource(t *testing.T) {
	files := basicFiles()
	files[".gitignore"] = "a/generated.go\n"
	root := fixture(t, files)
	writeFile(t, root, "a/generated.go", "package a\n")
	_, m, err := Analyze(context.Background(), root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if !m.Incomplete {
		t.Fatal("uncommitted ignored source changed analyzed graph")
	}
}

func TestAnalyzeLocalReplacement(t *testing.T) {
	files := basicFiles()
	files["go.mod"] = "module example.com/m\ngo 1.23\nrequire example.com/ext v0.0.0\nreplace example.com/ext => ./external\n"
	files["external/go.mod"] = "module example.com/ext\ngo 1.23\n"
	files["external/ext.go"] = "package ext\n"
	files["a/a.go"] = "package a\nimport _ \"example.com/ext\"\nfunc A(){}\n"
	root := fixture(t, files)
	_, m, err := Analyze(context.Background(), root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if !m.Incomplete {
		t.Fatal("local replacement must be incomplete")
	}
}

func TestAnalyzeFormerEmbedDirective(t *testing.T) {
	files := basicFiles()
	files["a/asset.md"] = "before"
	files["a/embed.go"] = "package a\nimport _ \"embed\"\n//go:embed asset.md\nvar text string\n"
	root := fixture(t, files)
	writeFile(t, root, "a/embed.go", "package a\n")
	writeFile(t, root, "a/asset.md", "after")
	commit(t, root)
	s, m, err := Analyze(context.Background(), root, "")
	if err != nil || m.Incomplete {
		t.Fatalf("%v %+v", err, m)
	}
	if !seeded(s, "pkg:example.com/m/a", 3) {
		t.Fatal("former embed owner not seeded")
	}
	for _, change := range m.Changes {
		if change.Path == "a/asset.md" && change.Classification == "excluded_document" {
			t.Fatal("former embed excluded as documentation")
		}
	}
}

// Analyze clears GOFLAGS and GOEXPERIMENT for its own go commands, so any
// inherited value means the analysis may differ from the caller's build. It
// must be reported as incomplete whether the value is valid, invalid, or
// affects analysis, and it must never be applied.
func TestAnalyzeReportsInheritedGoPolicy(t *testing.T) {
	root := fixture(t, basicFiles())
	for _, tt := range []struct {
		key, value string
		incomplete bool
	}{
		{"GOFLAGS", "", false},
		{"GOFLAGS", "-mod=readonly", true},
		{"GOFLAGS", "-tags=custom", true},
		{"GOFLAGS", "-mod=vendor", true},
		{"GOFLAGS", "-definitely-not-a-go-flag", true},
		{"GOFLAGS", "not-a-flag", true},
		{"GOEXPERIMENT", "nosuchexperiment", true},
	} {
		t.Run(tt.key+"="+tt.value, func(t *testing.T) {
			t.Setenv(tt.key, tt.value)
			s, m, err := Analyze(context.Background(), root, "HEAD")
			if err != nil {
				t.Fatal(err)
			}
			if m.Incomplete != tt.incomplete || len(m.BuildContext.BuildTags) != 0 || len(s.Nodes) != 4 {
				t.Fatalf("incomplete=%v warnings=%v tags=%v nodes=%d", m.Incomplete, m.Warnings, m.BuildContext.BuildTags, len(s.Nodes))
			}
		})
	}
}

func TestInheritedGoPolicyClearedBeforeTests(t *testing.T) {
	for _, key := range inheritedGoPolicy {
		if value, ok := os.LookupEnv(key); ok {
			t.Fatalf("%s=%q leaked from the parent environment", key, value)
		}
	}
}
