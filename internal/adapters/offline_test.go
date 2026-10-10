package adapters

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoEnvironmentOverridesInheritedPolicyWithoutDuplicates(t *testing.T) {
	want := map[string]string{
		"GOPROXY": "off", "GONOPROXY": "none", "GOPRIVATE": "", "GOVCS": "*:off",
		"GOSUMDB": "off", "GONOSUMDB": "none", "GOINSECURE": "", "GOTOOLCHAIN": "local",
		"GOWORK": "off", "GOFLAGS": "", "GOENV": "off", "GOEXPERIMENT": "", "GO111MODULE": "on",
	}
	for key := range want {
		t.Setenv(key, "hostile")
	}
	t.Setenv("GOOS", "linux")
	t.Setenv("GOARCH", "amd64")
	t.Setenv("CGO_ENABLED", "0")
	seen := map[string]string{}
	for _, item := range offlineEnvironment() {
		key, value, _ := strings.Cut(item, "=")
		if _, exists := seen[key]; exists {
			t.Fatalf("duplicate environment key: %s", key)
		}
		seen[key] = value
	}
	for key, value := range want {
		if seen[key] != value {
			t.Fatalf("%s=%q, want %q", key, seen[key], value)
		}
		if os.Getenv(key) != "hostile" {
			t.Fatalf("parent %s changed", key)
		}
	}
	for key, value := range map[string]string{"GOOS": "linux", "GOARCH": "amd64", "CGO_ENABLED": "0"} {
		if seen[key] != value {
			t.Fatalf("build condition %s changed", key)
		}
	}
}

func TestGoCommandIgnoresSavedEnvironmentAndInjectedFlags(t *testing.T) {
	root := fixture(t, basicFiles())
	config := filepath.Join(t.TempDir(), "goenv")
	if err := os.WriteFile(config, []byte("GOPROXY=direct\nGONOPROXY=*\nGOVCS=*:all\nGOFLAGS=-mod=mod -tags=saved\nGOOS=invalid\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOENV", config)
	t.Setenv("GOFLAGS", "-mod=mod -modfile=/nonexistent/injected.mod -tags=injected")
	t.Setenv("GO111MODULE", "off")
	data, err := command(context.Background(), root, "go", "env", "-json", "GOPROXY", "GONOPROXY", "GOVCS", "GOFLAGS", "GOENV", "GO111MODULE")
	if err != nil {
		t.Fatal(err)
	}
	var env map[string]string
	if err := json.Unmarshal(data, &env); err != nil {
		t.Fatal(err)
	}
	for key, value := range map[string]string{"GOPROXY": "off", "GONOPROXY": "none", "GOVCS": "*:off", "GOFLAGS": "", "GOENV": "", "GO111MODULE": "on"} {
		if env[key] != value {
			t.Fatalf("effective %s=%q", key, env[key])
		}
	}
	_, meta, err := Analyze(context.Background(), root, "HEAD")
	if err != nil || !meta.Incomplete || len(meta.BuildContext.BuildTags) != 0 {
		t.Fatalf("flags applied or silently accepted: %v %+v", err, meta)
	}
}
