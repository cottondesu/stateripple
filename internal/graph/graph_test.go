package graph_test

import (
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/cottondesu/stateripple/internal/graph"
)

func TestNormalizeAcceptsFreeFormKinds(t *testing.T) {
	nodes := []graph.Node{{ID: "db", Kind: "database"}, {ID: "api", Kind: "service"}, {ID: "job", Kind: "ジョブ"}, {ID: "pkg", Kind: "package"}, {ID: "t", Kind: "test"}}
	edges := []graph.Edge{
		{From: "db", To: "api", Kind: "depends_on", Attenuation: 0},
		{From: "api", To: "job", Kind: "triggers", Attenuation: 1},
		{From: "job", To: "db", Kind: "blocks", Attenuation: 2},
		{From: "t", To: "pkg", Kind: "connects_to", Attenuation: 3},
		{From: "pkg", To: "t", Kind: "own_test", Attenuation: 3},
		{From: "db", To: "db", Kind: "self", Attenuation: 0},
	}
	ns, es, err := graph.Normalize(nodes, edges)
	if err != nil {
		t.Fatal(err)
	}
	if len(ns) != len(nodes) || len(es) != len(edges) {
		t.Fatalf("lost elements: %v %v", ns, es)
	}
	for i := 1; i < len(ns); i++ {
		if ns[i-1].ID >= ns[i].ID {
			t.Fatalf("nodes not sorted: %v", ns)
		}
	}
	for i := 1; i < len(es); i++ {
		if !graph.Less(es[i-1], es[i]) {
			t.Fatalf("edges not sorted: %v", es)
		}
	}
	if nodes[0].ID != "db" || edges[0].From != "db" {
		t.Fatal("normalization mutated caller")
	}
}

func TestNormalizeAttenuationRange(t *testing.T) {
	nodes := []graph.Node{{ID: "a", Kind: "k"}, {ID: "b", Kind: "k"}}
	for a := 0; a <= graph.MaxAttenuation; a++ {
		if _, es, err := graph.Normalize(nodes, []graph.Edge{{From: "a", To: "b", Kind: "e", Attenuation: a}}); err != nil || es[0].Attenuation != a {
			t.Fatalf("attenuation %d: %v %v", a, es, err)
		}
	}
	for _, a := range []int{-1, 4, 5, math.MaxInt32, math.MaxInt, math.MinInt} {
		if _, _, err := graph.Normalize(nodes, []graph.Edge{{From: "a", To: "b", Kind: "e", Attenuation: a}}); err == nil {
			t.Fatalf("attenuation %d accepted", a)
		}
	}
}

func TestNormalizeRejectsInvalidStructure(t *testing.T) {
	ok := []graph.Node{{ID: "a", Kind: "k"}, {ID: "b", Kind: "k"}}
	cases := []struct {
		name  string
		nodes []graph.Node
		edges []graph.Edge
	}{
		{"empty_id", []graph.Node{{ID: "", Kind: "k"}}, nil},
		{"invalid_utf8_id", []graph.Node{{ID: "\xff", Kind: "k"}}, nil},
		{"large_id", []graph.Node{{ID: strings.Repeat("a", graph.MaxString+1), Kind: "k"}}, nil},
		{"empty_kind", []graph.Node{{ID: "a", Kind: ""}}, nil},
		{"invalid_utf8_kind", []graph.Node{{ID: "a", Kind: "\xc3"}}, nil},
		{"large_kind", []graph.Node{{ID: "a", Kind: strings.Repeat("k", graph.MaxString+1)}}, nil},
		{"duplicate_node_same_kind", []graph.Node{{ID: "a", Kind: "k"}, {ID: "a", Kind: "k"}}, nil},
		{"duplicate_node_other_kind", []graph.Node{{ID: "a", Kind: "k"}, {ID: "a", Kind: "j"}}, nil},
		{"missing_from", ok, []graph.Edge{{From: "x", To: "a", Kind: "e", Attenuation: 1}}},
		{"missing_to", ok, []graph.Edge{{From: "a", To: "x", Kind: "e", Attenuation: 1}}},
		{"empty_edge_kind", ok, []graph.Edge{{From: "a", To: "b", Kind: "", Attenuation: 1}}},
		{"invalid_utf8_edge_kind", ok, []graph.Edge{{From: "a", To: "b", Kind: "\xff", Attenuation: 1}}},
		{"contradictory_attenuation", ok, []graph.Edge{{From: "a", To: "b", Kind: "e", Attenuation: 1}, {From: "a", To: "b", Kind: "e", Attenuation: 2}}},
		{"conflicting_kind", ok, []graph.Edge{{From: "a", To: "b", Kind: "e", Attenuation: 1}, {From: "a", To: "b", Kind: "f", Attenuation: 1}}},
		{"too_many_nodes", make([]graph.Node, graph.MaxNodes+1), nil},
		{"too_many_edges", ok, make([]graph.Edge, graph.MaxEdges+1)},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if _, _, err := graph.Normalize(tt.nodes, tt.edges); err == nil {
				t.Fatal("invalid graph accepted")
			}
		})
	}
}

func TestNormalizeMergesExactDuplicatesAndIsIdempotent(t *testing.T) {
	nodes := []graph.Node{{ID: "b", Kind: "k"}, {ID: "a", Kind: "k"}}
	e := graph.Edge{From: "a", To: "b", Kind: "e", Attenuation: 2}
	ns, es, err := graph.Normalize(nodes, []graph.Edge{e, e, e})
	if err != nil {
		t.Fatal(err)
	}
	if len(es) != 1 || es[0] != e || ns[0].ID != "a" {
		t.Fatalf("unexpected normalization: %v %v", ns, es)
	}
	ns2, es2, err := graph.Normalize(ns, es)
	if err != nil || !reflect.DeepEqual(ns, ns2) || !reflect.DeepEqual(es, es2) {
		t.Fatalf("not idempotent: %v", err)
	}
}

func TestNormalizeEmptyGraph(t *testing.T) {
	ns, es, err := graph.Normalize(nil, nil)
	if err != nil || len(ns) != 0 || len(es) != 0 {
		t.Fatalf("%v %v %v", ns, es, err)
	}
}
