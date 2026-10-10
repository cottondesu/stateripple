package adapters

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const commandLimit = 64 << 20

type limitedBuffer struct {
	bytes.Buffer
	exceeded bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > commandLimit {
		b.exceeded = true
		return 0, fmt.Errorf("command output exceeds %d bytes", commandLimit)
	}
	return b.Buffer.Write(p)
}

func command(ctx context.Context, root string, name string, args ...string) ([]byte, error) {
	if name == "git" {
		args = append([]string{"--no-lazy-fetch"}, args...)
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = root
	if name == "go" {
		cmd.Env = offlineEnvironment()
	} else if name == "git" {
		cmd.Env = gitOfflineEnvironment()
	}
	var out, stderr limitedBuffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if out.exceeded || stderr.exceeded {
		return nil, fmt.Errorf("%s output limit exceeded", name)
	}
	if err != nil {
		return out.Bytes(), fmt.Errorf("%s: %w", name, &commandFailure{err, strings.ReplaceAll(strings.TrimSpace(stderr.String()), root, "<root>")})
	}
	return out.Bytes(), nil
}

type commandFailure struct {
	err        error
	diagnostic string
}

func (e *commandFailure) Error() string { return fmt.Sprintf("%v: %s", e.err, e.diagnostic) }
func (e *commandFailure) Unwrap() error { return e.err }

func gitOfflineEnvironment() []string {
	env := []string{}
	for _, v := range os.Environ() {
		key, _, _ := strings.Cut(v, "=")
		if key != "GIT_NO_LAZY_FETCH" && key != "GIT_TERMINAL_PROMPT" && key != "GIT_OPTIONAL_LOCKS" && !strings.HasPrefix(key, "GIT_TRACE") {
			env = append(env, v)
		}
	}
	return append(env, "GIT_NO_LAZY_FETCH=1", "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0")
}

func offlineEnvironment() []string {
	env := []string{}
	for _, v := range gitOfflineEnvironment() {
		key, _, _ := strings.Cut(v, "=")
		switch key {
		case "GOWORK", "GOTOOLCHAIN", "GOPROXY", "GONOPROXY", "GOPRIVATE", "GOVCS", "GOSUMDB", "GONOSUMDB", "GOINSECURE", "GOENV", "GOFLAGS", "GOEXPERIMENT", "GO111MODULE":
		default:
			env = append(env, v)
		}
	}
	return append(env, "GOWORK=off", "GOTOOLCHAIN=local", "GOPROXY=off", "GONOPROXY=none", "GOPRIVATE=", "GOVCS=*:off", "GOSUMDB=off", "GONOSUMDB=none", "GOINSECURE=", "GOENV=off", "GOFLAGS=", "GOEXPERIMENT=", "GO111MODULE=on")
}
