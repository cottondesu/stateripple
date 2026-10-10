package risk_test

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/cottondesu/stateripple/internal/graph"
	"github.com/cottondesu/stateripple/internal/rules/risk"
	"github.com/cottondesu/stateripple/internal/snapshot"
)

func checkProperties(t *testing.T, s snapshot.Snapshot) {
	t.Helper()
	r, err := risk.Run(s)
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != "complete" || r.GenerationCount > 3*len(s.Nodes)+1 {
		t.Fatalf("finite graph did not converge: %+v", r)
	}
	values := map[string]int{}
	for _, w := range r.Waves {
		for _, c := range w.Changes {
			if c.Before != values[c.ID] || c.After <= c.Before || c.After > 3 {
				t.Fatalf("not monotone/ranged: %+v", c)
			}
			values[c.ID] = c.After
		}
	}
	for _, c := range r.Cells {
		if c.Risk != values[c.ID] {
			t.Fatal("history/final mismatch")
		}
	}
	stable := s
	stable.Seeds = []snapshot.Seed{}
	for _, c := range r.Cells {
		stable.Seeds = append(stable.Seeds, snapshot.Seed{ID: c.ID, Risk: c.Risk, Reason: "fixed_point"})
	}
	final, err := risk.Run(stable)
	if err != nil {
		t.Fatal(err)
	}
	if final.GenerationCount != 1 {
		t.Fatal("fixed point not stable")
	}
}

func TestPropertiesWhenRandomFiniteGraphs(t *testing.T) {
	rng := rand.New(rand.NewSource(731))
	for iteration := 0; iteration < 100; iteration++ {
		n := 1 + rng.Intn(40)
		ids := make([]string, n)
		for i := range ids {
			ids[i] = fmt.Sprint(i)
		}
		s := fixture(ids, nil, nil)
		s.MaxGenerations = 3*n + 1
		for i := 0; i < n*3; i++ {
			s.Edges = append(s.Edges, graph.Edge{From: ids[rng.Intn(n)], To: ids[rng.Intn(n)], Kind: "prod_import", Attenuation: 1})
		}
		for i := 0; i < 5; i++ {
			s.Seeds = append(s.Seeds, snapshot.Seed{ID: ids[rng.Intn(n)], Risk: rng.Intn(4), Reason: "random"})
		}
		checkProperties(t, s)
	}
}

func FuzzPropagation(f *testing.F) {
	f.Add([]byte{3, 1, 2, 4, 2, 9, 3, 8})
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, b []byte) {
		n := 1 + len(b)%32
		ids := make([]string, n)
		for i := range ids {
			ids[i] = fmt.Sprint(i)
		}
		s := fixture(ids, nil, nil)
		s.MaxGenerations = 3*n + 1
		for i := 0; i+2 < len(b) && i < 300; i += 3 {
			s.Edges = append(s.Edges, graph.Edge{From: ids[int(b[i])%n], To: ids[int(b[i+1])%n], Kind: "prod_import", Attenuation: 1})
			s.Seeds = append(s.Seeds, snapshot.Seed{ID: ids[int(b[i])%n], Risk: int(b[i+2]) % 4, Reason: "fuzz"})
		}
		checkProperties(t, s)
	})
}
