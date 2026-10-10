package risk_test

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"

	"github.com/cottondesu/stateripple/internal/report"
	"github.com/cottondesu/stateripple/internal/rules/risk"
	"github.com/cottondesu/stateripple/internal/snapshot"
)

func TestDeterminismWhenInputOrderChanges(t *testing.T) {
	s := fixture([]string{"A", "B", "C", "D", "Z"}, [][2]string{{"A", "B"}, {"A", "C"}, {"B", "D"}, {"C", "D"}, {"D", "A"}}, []snapshot.Seed{{ID: "A", Risk: 3, Reason: "source"}, {ID: "A", Risk: 1, Reason: "other"}, {ID: "B", Risk: 2, Reason: "test"}})
	baseline, err := risk.Run(s)
	if err != nil {
		t.Fatal(err)
	}
	want, err := report.JSON(baseline)
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewSource(410))
	for i := 0; i < 200; i++ {
		rng.Shuffle(len(s.Nodes), func(a, b int) { s.Nodes[a], s.Nodes[b] = s.Nodes[b], s.Nodes[a] })
		rng.Shuffle(len(s.Edges), func(a, b int) { s.Edges[a], s.Edges[b] = s.Edges[b], s.Edges[a] })
		rng.Shuffle(len(s.Seeds), func(a, b int) { s.Seeds[a], s.Seeds[b] = s.Seeds[b], s.Seeds[a] })
		got, err := risk.Run(s)
		if err != nil {
			t.Fatal(err)
		}
		b, err := report.JSON(got)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, baseline) || !bytes.Equal(b, want) {
			t.Fatalf("non-deterministic iteration %d", i)
		}
	}
}

func TestDeterminismWhenRunRepeats(t *testing.T) {
	s := fixture([]string{"A", "B"}, [][2]string{{"A", "B"}}, []snapshot.Seed{{ID: "A", Risk: 3, Reason: "source"}})
	var want []byte
	for i := 0; i < 25; i++ {
		r, err := risk.Run(s)
		if err != nil {
			t.Fatal(err)
		}
		got, err := report.JSON(r)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			want = got
		} else if !bytes.Equal(want, got) {
			t.Fatal("repeated output differs")
		}
	}
}
