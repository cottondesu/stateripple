package snapshot_test

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/cottondesu/stateripple/internal/graph"
	"github.com/cottondesu/stateripple/internal/snapshot"
)

const empty = `{"schema_version":1,"ruleset":"risk-propagation-v1","nodes":[],"edges":[],"seeds":[]}`

func base() snapshot.Snapshot {
	return snapshot.Snapshot{SchemaVersion: 1, Ruleset: snapshot.Ruleset, MaxGenerations: 16, Nodes: []graph.Node{{ID: "A", Kind: "package"}, {ID: "B", Kind: "package"}, {ID: "T", Kind: "test"}}, Edges: []graph.Edge{{From: "A", To: "B", Kind: "prod_import", Attenuation: 1}}, Seeds: []snapshot.Seed{{ID: "A", Risk: 3, Reason: "source"}}}
}

func TestRejectWhenDomainInputInvalid(t *testing.T) {
	cases := []struct {
		name string
		edit func(*snapshot.Snapshot)
	}{
		{"schema", func(s *snapshot.Snapshot) { s.SchemaVersion = 2 }},
		{"rule", func(s *snapshot.Snapshot) { s.Ruleset = "other" }},
		{"zero_limit", func(s *snapshot.Snapshot) { s.MaxGenerations = 0 }},
		{"huge_limit", func(s *snapshot.Snapshot) { s.MaxGenerations = 1025 }},
		{"empty_id", func(s *snapshot.Snapshot) { s.Nodes[0].ID = "" }},
		{"large_id", func(s *snapshot.Snapshot) { s.Nodes[0].ID = strings.Repeat("a", 4097) }},
		{"duplicate_node", func(s *snapshot.Snapshot) { s.Nodes = append(s.Nodes, s.Nodes[0]) }},
		{"empty_node_kind", func(s *snapshot.Snapshot) { s.Nodes[0].Kind = "" }},
		{"large_node_kind", func(s *snapshot.Snapshot) { s.Nodes[0].Kind = strings.Repeat("k", 4097) }},
		{"invalid_utf8_node_kind", func(s *snapshot.Snapshot) { s.Nodes[0].Kind = "\xff" }},
		{"missing_endpoint", func(s *snapshot.Snapshot) { s.Edges[0].To = "missing" }},
		{"empty_edge_kind", func(s *snapshot.Snapshot) { s.Edges[0].Kind = "" }},
		{"large_edge_kind", func(s *snapshot.Snapshot) { s.Edges[0].Kind = strings.Repeat("k", 4097) }},
		{"invalid_utf8_edge_kind", func(s *snapshot.Snapshot) { s.Edges[0].Kind = "\xff" }},
		{"negative_attenuation", func(s *snapshot.Snapshot) { s.Edges[0].Attenuation = -1 }},
		{"attenuation_4", func(s *snapshot.Snapshot) { s.Edges[0].Attenuation = 4 }},
		{"attenuation_5", func(s *snapshot.Snapshot) { s.Edges[0].Attenuation = 5 }},
		{"attenuation_max_int", func(s *snapshot.Snapshot) { s.Edges[0].Attenuation = math.MaxInt }},
		{"attenuation_min_int", func(s *snapshot.Snapshot) { s.Edges[0].Attenuation = math.MinInt }},
		{"contradictory_edge", func(s *snapshot.Snapshot) {
			s.Edges = append(s.Edges, graph.Edge{From: "A", To: "B", Kind: "prod_import", Attenuation: 2})
		}},
		{"conflicting_kind", func(s *snapshot.Snapshot) {
			s.Edges = []graph.Edge{{From: "A", To: "T", Kind: "own_test"}, {From: "A", To: "T", Kind: "test_import", Attenuation: 1}}
		}},
		{"seed_missing", func(s *snapshot.Snapshot) { s.Seeds[0].ID = "Z" }},
		{"negative_risk", func(s *snapshot.Snapshot) { s.Seeds[0].Risk = -1 }},
		{"large_risk", func(s *snapshot.Snapshot) { s.Seeds[0].Risk = 4 }},
		{"empty_reason", func(s *snapshot.Snapshot) { s.Seeds[0].Reason = "" }},
		{"nodes_limit", func(s *snapshot.Snapshot) { s.Nodes = make([]graph.Node, graph.MaxNodes+1) }},
		{"edges_limit", func(s *snapshot.Snapshot) { s.Edges = make([]graph.Edge, graph.MaxEdges+1) }},
		{"seeds_limit", func(s *snapshot.Snapshot) { s.Seeds = make([]snapshot.Seed, snapshot.MaxSeeds+1) }},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s := base()
			tt.edit(&s)
			_, err := snapshot.Normalize(s)
			if err == nil {
				t.Fatal("invalid input accepted")
			}
		})
	}
}

// These inputs were rejected while the core validated the Go vocabulary. The
// generic core accepts them; the Go adapter still rejects them (see
// adapters.TestGoProfileRejectsFormerCoreCases).
func TestNormalizeAcceptsGenericVocabulary(t *testing.T) {
	cases := []struct {
		name string
		edit func(*snapshot.Snapshot)
	}{
		{"node_kind", func(s *snapshot.Snapshot) { s.Nodes[0].Kind = "bad" }},
		{"edge_kind", func(s *snapshot.Snapshot) { s.Edges[0].Kind = "bad" }},
		{"test_out", func(s *snapshot.Snapshot) { s.Edges[0].From = "T" }},
		{"prod_to_test", func(s *snapshot.Snapshot) { s.Edges[0].To = "T" }},
		{"test_to_prod", func(s *snapshot.Snapshot) { s.Edges[0].Kind = "test_import" }},
		{"attenuation_0", func(s *snapshot.Snapshot) { s.Edges[0].Attenuation = 0 }},
		{"attenuation_2", func(s *snapshot.Snapshot) { s.Edges[0].Attenuation = 2 }},
		{"attenuation_3", func(s *snapshot.Snapshot) { s.Edges[0].Attenuation = 3 }},
		{"unicode_kinds", func(s *snapshot.Snapshot) { s.Nodes[0].Kind, s.Edges[0].Kind = "倉庫", "ships to\u0007" }},
		{"self_loop", func(s *snapshot.Snapshot) { s.Edges[0].To = s.Edges[0].From }},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s := base()
			tt.edit(&s)
			n, err := snapshot.Normalize(s)
			if err != nil {
				t.Fatal(err)
			}
			again, err := snapshot.Normalize(n)
			if err != nil || !reflect.DeepEqual(n, again) {
				t.Fatalf("normalization not idempotent: %v", err)
			}
		})
	}
}

func TestReadAcceptsGenericSnapshot(t *testing.T) {
	input := `{"schema_version":1,"ruleset":"risk-propagation-v1","nodes":[{"id":"plant","kind":"facility"},{"id":"port","kind":"warehouse"}],"edges":[{"from":"plant","to":"port","kind":"ships_to","attenuation":2}],"seeds":[{"id":"plant","risk":3,"reason":"flood"}]}`
	s, err := snapshot.Read(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if s.Edges[0].Kind != "ships_to" || s.Edges[0].Attenuation != 2 || s.Nodes[0].Kind != "facility" {
		t.Fatalf("generic snapshot changed: %+v", s)
	}
	for _, bad := range []string{`"attenuation":4`, `"attenuation":5`, `"attenuation":-1`, `"attenuation":2.0`, `"attenuation":"2"`, `"attenuation":99999999999999999999`, `"attenuation":null`} {
		t.Run(bad, func(t *testing.T) {
			if _, err := snapshot.Read(strings.NewReader(strings.Replace(input, `"attenuation":2`, bad, 1))); err == nil {
				t.Fatal("invalid attenuation accepted")
			}
		})
	}
}

func TestReadWhenJSONInvalid(t *testing.T) {
	cases := []string{
		`{`, empty + empty, `null`, strings.Replace(empty, `"schema_version":1`, `"schema_version":2`, 1),
		strings.Replace(empty, `"risk-propagation-v1"`, `"unknown"`, 1),
		strings.Replace(empty, `"nodes":[]`, `"nodes":null`, 1),
		strings.Replace(empty, `"nodes":[]`, `"nodes":[{"id":"A"}]`, 1),
		strings.Replace(empty, `"edges":[]`, `"edges":[{"from":"A","to":"A","kind":"prod_import"}]`, 1),
		strings.Replace(empty, `"seeds":[]`, `"seeds":[{"id":"A","reason":"change"}]`, 1),
		strings.Replace(empty, `"schema_version":1`, `"schema_version":1,"schema_version":1`, 1),
		strings.Replace(empty, `"schema_version":1`, `"extra":1,"schema_version":1`, 1),
		strings.Replace(empty, `"schema_version":1`, `"max_generations":0,"schema_version":1`, 1),
		strings.Replace(empty, `"schema_version":1`, `"max_generations":1.2,"schema_version":1`, 1),
		strings.Repeat(" ", snapshot.MaxInputBytes+1),
	}
	for i, input := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			_, err := snapshot.Read(strings.NewReader(input))
			if err == nil {
				t.Fatal("invalid JSON accepted")
			}
		})
	}
}

// complete is accepted as written and refers only to existing nodes, so a case
// that removes or replaces exactly one field can fail for that reason alone.
const complete = `{"schema_version":1,"ruleset":"risk-propagation-v1","nodes":[{"id":"a","kind":"service"},{"id":"b","kind":"store"}],"edges":[{"from":"a","to":"b","kind":"calls","attenuation":2}],"seeds":[{"id":"a","risk":3,"reason":"outage"}]}`

func replaceOnce(t *testing.T, old, replacement string) string {
	t.Helper()
	if n := strings.Count(complete, old); n != 1 {
		t.Fatalf("fixture contains %q %d times, want 1", old, n)
	}
	return strings.Replace(complete, old, replacement, 1)
}

func TestReadRejectsOmittedRequiredField(t *testing.T) {
	if _, err := snapshot.Read(strings.NewReader(complete)); err != nil {
		t.Fatalf("complete snapshot rejected: %v", err)
	}
	cases := []struct {
		name, old, replacement, want string
	}{
		{"schema_version", `"schema_version":1,`, ``, `missing required field "schema_version"`},
		{"ruleset", `"ruleset":"risk-propagation-v1",`, ``, `missing required field "ruleset"`},
		{"nodes", `"nodes":[{"id":"a","kind":"service"},{"id":"b","kind":"store"}],`, ``, `missing required field "nodes"`},
		{"edges", `"edges":[{"from":"a","to":"b","kind":"calls","attenuation":2}],`, ``, `missing required field "edges"`},
		{"seeds", `,"seeds":[{"id":"a","risk":3,"reason":"outage"}]`, ``, `missing required field "seeds"`},
		{"node_id", `{"id":"b","kind":"store"}`, `{"kind":"store"}`, `nodes: missing required field "id"`},
		{"node_kind", `{"id":"b","kind":"store"}`, `{"id":"b"}`, `nodes: missing required field "kind"`},
		{"edge_from", `"from":"a",`, ``, `edges: missing required field "from"`},
		{"edge_to", `"to":"b",`, ``, `edges: missing required field "to"`},
		{"edge_kind", `"kind":"calls",`, ``, `edges: missing required field "kind"`},
		{"edge_attenuation", `,"attenuation":2`, ``, `edges: missing required field "attenuation"`},
		{"seed_id", `{"id":"a","risk":3,`, `{"risk":3,`, `seeds: missing required field "id"`},
		{"seed_risk", `"risk":3,`, ``, `seeds: missing required field "risk"`},
		{"seed_reason", `,"reason":"outage"`, ``, `seeds: missing required field "reason"`},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			_, err := snapshot.Read(strings.NewReader(replaceOnce(t, tt.old, tt.replacement)))
			if err == nil {
				t.Fatal("omitted field accepted")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("rejected for another reason: %v, want %s", err, tt.want)
			}
		})
	}
}

// Omission, null and an explicit zero are three different inputs: only the
// explicit zero is a value.
func TestReadDistinguishesZeroNullAndOmission(t *testing.T) {
	zero, err := snapshot.Read(strings.NewReader(replaceOnce(t, `"attenuation":2`, `"attenuation":0`)))
	if err != nil || zero.Edges[0].Attenuation != 0 {
		t.Fatalf("explicit attenuation 0: %+v, %v", zero.Edges, err)
	}
	zero, err = snapshot.Read(strings.NewReader(replaceOnce(t, `"risk":3`, `"risk":0`)))
	if err != nil || zero.Seeds[0].Risk != 0 {
		t.Fatalf("explicit risk 0: %+v, %v", zero.Seeds, err)
	}
	for _, field := range []string{`"attenuation":2`, `"risk":3`, `"kind":"calls"`, `"kind":"store"`} {
		name, _, _ := strings.Cut(field, ":")
		t.Run(name+":null", func(t *testing.T) {
			_, err := snapshot.Read(strings.NewReader(replaceOnce(t, field, name+":null")))
			if err == nil || !strings.Contains(err.Error(), "null is not accepted") {
				t.Fatalf("null: %v", err)
			}
		})
	}
}

func TestNormalizeWhenDuplicatesMatch(t *testing.T) {
	s := base()
	s.Edges = append(s.Edges, s.Edges[0])
	s.Seeds = append(s.Seeds, s.Seeds[0], snapshot.Seed{ID: "A", Risk: 1, Reason: "other"})
	n, err := snapshot.Normalize(s)
	if err != nil {
		t.Fatal(err)
	}
	if len(n.Edges) != 1 || len(n.Seeds) != 2 || len(s.Edges) != 2 {
		t.Fatalf("normalization: %+v", n)
	}
	n2, err := snapshot.Normalize(n)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(n, n2) {
		t.Fatal("normalization not idempotent")
	}
}

func TestHashWhenDefaultsNormalized(t *testing.T) {
	s, err := snapshot.Read(strings.NewReader(empty))
	if err != nil {
		t.Fatal(err)
	}
	got, err := snapshot.Hash(s)
	if err != nil {
		t.Fatal(err)
	}
	normalized := []byte("{\"schema_version\":1,\"ruleset\":\"risk-propagation-v1\",\"max_generations\":16,\"nodes\":[],\"edges\":[],\"seeds\":[]}\n")
	want := fmt.Sprintf("%x", sha256.Sum256(normalized))
	if got != want {
		t.Fatalf("hash %s != %s", got, want)
	}
	other, err := snapshot.Read(bytes.NewBufferString(`{"seeds":[],"edges":[],"nodes":[],"ruleset":"risk-propagation-v1","schema_version":1,"max_generations":16}`))
	if err != nil {
		t.Fatal(err)
	}
	h, err := snapshot.Hash(other)
	if err != nil {
		t.Fatal(err)
	}
	if h != got {
		t.Fatal("logical equivalence changed hash")
	}
}

func FuzzRead(f *testing.F) {
	f.Add([]byte(empty))
	f.Add([]byte(`{"nodes":[null]}`))
	f.Add([]byte(`[]`))
	f.Fuzz(func(t *testing.T, b []byte) {
		s, err := snapshot.Read(bytes.NewReader(b))
		if err != nil {
			return
		}
		n, err := snapshot.Normalize(s)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(s, n) {
			t.Fatal("normalization not idempotent")
		}
	})
}
