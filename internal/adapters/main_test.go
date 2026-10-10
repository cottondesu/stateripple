package adapters

import (
	"os"
	"testing"
)

// inheritedGoPolicy lists the variables that Analyze itself inspects in its own
// process and reports as unsupported. A developer's or CI runner's values must
// not decide test outcomes, so they are cleared once before any test starts.
// Tests that exercise these variables set them explicitly with t.Setenv.
var inheritedGoPolicy = []string{"GOFLAGS", "GOEXPERIMENT", "GOWORK"}

func TestMain(m *testing.M) {
	for _, key := range inheritedGoPolicy {
		os.Unsetenv(key)
	}
	os.Exit(m.Run())
}
