package snapshot

import (
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/cottondesu/stateripple/internal/canonical"
	"github.com/cottondesu/stateripple/internal/graph"
)

const (
	SchemaVersion      = 1
	Ruleset            = "risk-propagation-v1"
	DefaultGenerations = 16
	MaxGenerations     = 1024
	MaxSeeds           = 100000
	MaxInputBytes      = 16 << 20
)

type Seed struct {
	ID     string `json:"id"`
	Risk   int    `json:"risk"`
	Reason string `json:"reason"`
}

type Snapshot struct {
	SchemaVersion  int          `json:"schema_version"`
	Ruleset        string       `json:"ruleset"`
	MaxGenerations int          `json:"max_generations"`
	Nodes          []graph.Node `json:"nodes"`
	Edges          []graph.Edge `json:"edges"`
	Seeds          []Seed       `json:"seeds"`
}

func Normalize(s Snapshot) (Snapshot, error) {
	if s.SchemaVersion != SchemaVersion {
		return Snapshot{}, fmt.Errorf("unsupported schema_version %d", s.SchemaVersion)
	}
	if s.Ruleset != Ruleset {
		return Snapshot{}, fmt.Errorf("unsupported ruleset %q", s.Ruleset)
	}
	if s.MaxGenerations < 1 || s.MaxGenerations > MaxGenerations {
		return Snapshot{}, fmt.Errorf("max_generations must be 1..%d", MaxGenerations)
	}
	if len(s.Seeds) > MaxSeeds {
		return Snapshot{}, fmt.Errorf("too many seeds")
	}
	nodes, edges, err := graph.Normalize(s.Nodes, s.Edges)
	if err != nil {
		return Snapshot{}, err
	}
	ids := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		ids[n.ID] = true
	}
	seeds := append([]Seed{}, s.Seeds...)
	for _, seed := range seeds {
		if !ids[seed.ID] {
			return Snapshot{}, fmt.Errorf("seed node not found: %q", seed.ID)
		}
		if seed.Risk < 0 || seed.Risk > 3 {
			return Snapshot{}, fmt.Errorf("invalid risk %d", seed.Risk)
		}
		if !graph.ValidString(seed.Reason) {
			return Snapshot{}, fmt.Errorf("invalid seed reason")
		}
	}
	sort.Slice(seeds, func(i, j int) bool {
		if seeds[i].ID != seeds[j].ID {
			return seeds[i].ID < seeds[j].ID
		}
		if seeds[i].Risk != seeds[j].Risk {
			return seeds[i].Risk > seeds[j].Risk
		}
		return seeds[i].Reason < seeds[j].Reason
	})
	unique := make([]Seed, 0, len(seeds))
	for _, seed := range seeds {
		if len(unique) == 0 || unique[len(unique)-1] != seed {
			unique = append(unique, seed)
		}
	}
	s.Nodes, s.Edges, s.Seeds = nodes, edges, unique
	return s, nil
}

func Hash(s Snapshot) (string, error) {
	n, err := Normalize(s)
	if err != nil {
		return "", err
	}
	b, err := canonical.JSON(n)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(b)), nil
}
