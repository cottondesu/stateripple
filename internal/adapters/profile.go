package adapters

import (
	"fmt"

	"github.com/cottondesu/stateripple/internal/snapshot"
)

// validateGoProfile enforces the Go dependency vocabulary that the core graph
// no longer restricts: package and test nodes, prod_import between packages,
// own_test and test_import from a package to a test, fixed attenuations, and
// no outgoing edges from test nodes. The Go adapter must never emit anything
// else, so a violation is an execution error rather than a warning.
func validateGoProfile(s snapshot.Snapshot) error {
	kinds := make(map[string]string, len(s.Nodes))
	for _, n := range s.Nodes {
		if n.Kind != "package" && n.Kind != "test" {
			return fmt.Errorf("invalid node kind %q", n.Kind)
		}
		kinds[n.ID] = n.Kind
	}
	for _, e := range s.Edges {
		from, to := kinds[e.From], kinds[e.To]
		if from != "package" {
			return fmt.Errorf("test nodes cannot have outgoing edges")
		}
		want := 1
		switch e.Kind {
		case "prod_import":
			if to != "package" {
				return fmt.Errorf("prod_import must target package")
			}
		case "own_test":
			want = 0
			fallthrough
		case "test_import":
			if to != "test" {
				return fmt.Errorf("test edge must target test")
			}
		default:
			return fmt.Errorf("invalid edge kind %q", e.Kind)
		}
		if e.Attenuation != want {
			return fmt.Errorf("invalid attenuation for %s", e.Kind)
		}
	}
	return nil
}
