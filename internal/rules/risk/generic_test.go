package risk_test

import (
	"bytes"
	"container/heap"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/cottondesu/stateripple/internal/graph"
	"github.com/cottondesu/stateripple/internal/report"
	"github.com/cottondesu/stateripple/internal/rules/risk"
	"github.com/cottondesu/stateripple/internal/snapshot"
)

// referenceResult is computed directly from the RiskPropagationV1 formula
//
//	R_next(v) = max(R(v), Seed(v), max over u->v of max(0, R(u) - attenuation(u,v)))
//
// using maps keyed by node ID. It shares no code with the engine.
type referenceResult struct {
	final       map[string]int
	generations int
	converged   bool
	waves       []map[string][2]int
}

func reference(s snapshot.Snapshot) referenceResult {
	seed := map[string]int{}
	for _, sd := range s.Seeds {
		seed[sd.ID] = max(seed[sd.ID], sd.Risk)
	}
	current := map[string]int{}
	for _, n := range s.Nodes {
		current[n.ID] = seed[n.ID]
	}
	result := referenceResult{}
	for g := 1; g <= s.MaxGenerations; g++ {
		next := map[string]int{}
		for id, v := range current {
			next[id] = max(v, seed[id])
		}
		for _, e := range s.Edges {
			next[e.To] = max(next[e.To], max(0, current[e.From]-e.Attenuation))
		}
		changed := map[string][2]int{}
		for id, v := range current {
			if next[id] != v {
				changed[id] = [2]int{v, next[id]}
			}
		}
		result.generations = g
		current = next
		if len(changed) == 0 {
			result.converged = true
			break
		}
		result.waves = append(result.waves, changed)
	}
	result.final = current
	return result
}

type item struct {
	id   string
	dist int
}
type queue []item

func (q queue) Len() int           { return len(q) }
func (q queue) Less(i, j int) bool { return q[i].dist < q[j].dist }
func (q queue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *queue) Push(x any)        { *q = append(*q, x.(item)) }
func (q *queue) Pop() any {
	old := *q
	x := old[len(old)-1]
	*q = old[:len(old)-1]
	return x
}

// pathFixedPoint computes the least fixed point a second, independent way:
// R*(v) = max(0, max over seeded u of Seed(u) - d(u,v)), where d is the minimum
// total attenuation of a path from u to v (Dijkstra; attenuations are >= 0).
func pathFixedPoint(s snapshot.Snapshot) map[string]int {
	out := map[string][]graph.Edge{}
	for _, e := range s.Edges {
		out[e.From] = append(out[e.From], e)
	}
	seed := map[string]int{}
	for _, sd := range s.Seeds {
		seed[sd.ID] = max(seed[sd.ID], sd.Risk)
	}
	final := map[string]int{}
	for _, n := range s.Nodes {
		final[n.ID] = 0
	}
	for source, risk := range seed {
		if risk == 0 {
			continue
		}
		dist := map[string]int{source: 0}
		q := &queue{{source, 0}}
		for q.Len() > 0 {
			cur := heap.Pop(q).(item)
			if cur.dist > dist[cur.id] {
				continue
			}
			final[cur.id] = max(final[cur.id], risk-cur.dist)
			for _, e := range out[cur.id] {
				d := cur.dist + e.Attenuation
				if old, ok := dist[e.To]; (!ok || d < old) && d < risk {
					dist[e.To] = d
					heap.Push(q, item{e.To, d})
				}
			}
		}
	}
	return final
}

func generic(limit int, nodes []graph.Node, edges []graph.Edge, seeds []snapshot.Seed) snapshot.Snapshot {
	return snapshot.Snapshot{SchemaVersion: 1, Ruleset: snapshot.Ruleset, MaxGenerations: limit, Nodes: nodes, Edges: edges, Seeds: seeds}
}

func nodesOf(kind string, ids ...string) []graph.Node {
	nodes := []graph.Node{}
	for _, id := range ids {
		nodes = append(nodes, graph.Node{ID: id, Kind: kind})
	}
	return nodes
}

// compareWithReference checks the report against both reference computations
// and checks that every recorded cause satisfies the rule.
func compareWithReference(t *testing.T, s snapshot.Snapshot, r report.Report) {
	t.Helper()
	ref := reference(s)
	if !reflect.DeepEqual(risks(r), ref.final) {
		t.Fatalf("final state %v, reference %v", risks(r), ref.final)
	}
	if r.GenerationCount != ref.generations {
		t.Fatalf("generation_count %d, reference %d", r.GenerationCount, ref.generations)
	}
	wantStatus, wantStop := "incomplete", "generation_limit"
	if ref.converged {
		wantStatus, wantStop = "complete", "converged"
		if paths := pathFixedPoint(s); !reflect.DeepEqual(paths, ref.final) {
			t.Fatalf("fixed point %v, path model %v", ref.final, paths)
		}
	}
	if r.Status != wantStatus || r.StopReason != wantStop {
		t.Fatalf("status %s/%s, want %s/%s", r.Status, r.StopReason, wantStatus, wantStop)
	}
	seedMax := map[string]int{}
	for _, sd := range s.Seeds {
		seedMax[sd.ID] = max(seedMax[sd.ID], sd.Risk)
	}
	waves := r.Waves
	if len(waves) > 0 && waves[0].Generation == 0 {
		for _, c := range waves[0].Changes {
			if c.Before != 0 || c.After != seedMax[c.ID] || len(c.Causes) == 0 {
				t.Fatalf("wave 0 change %+v", c)
			}
			for _, cause := range c.Causes {
				if cause.Edge != nil {
					t.Fatalf("wave 0 cause has edge: %+v", cause)
				}
			}
		}
		waves = waves[1:]
	}
	if len(waves) != len(ref.waves) {
		t.Fatalf("%d waves, reference %d", len(waves), len(ref.waves))
	}
	previous := map[string]int{}
	for _, n := range s.Nodes {
		previous[n.ID] = seedMax[n.ID]
	}
	edgesTo := map[string][]graph.Edge{}
	for _, e := range s.Edges {
		edgesTo[e.To] = append(edgesTo[e.To], e)
	}
	for i, w := range waves {
		if w.Generation != i+1 {
			t.Fatalf("wave generation %d at %d", w.Generation, i)
		}
		got := map[string][2]int{}
		for _, c := range w.Changes {
			got[c.ID] = [2]int{c.Before, c.After}
			if len(c.Causes) == 0 {
				t.Fatalf("change without cause: %+v", c)
			}
			for _, cause := range c.Causes {
				if cause.Edge == nil || cause.Edge.To != c.ID || previous[cause.Edge.From]-cause.Edge.Attenuation != c.After || cause.Reason != cause.Edge.Kind {
					t.Fatalf("generation %d cause does not support %+v: %+v", w.Generation, c, cause)
				}
			}
		}
		if !reflect.DeepEqual(got, ref.waves[i]) {
			t.Fatalf("generation %d changes %v, reference %v", w.Generation, got, ref.waves[i])
		}
		for id, change := range got {
			previous[id] = change[1]
		}
	}
	for _, c := range r.Cells {
		// Final causes describe support in the last computed state. Only a
		// fixed point guarantees that every positive cell has support.
		if ref.converged && (c.Risk > 0) != (len(c.Causes) > 0) {
			t.Fatalf("final causes for %+v", c)
		}
		if c.Risk == 0 && len(c.Causes) > 0 {
			t.Fatalf("zero cell with causes: %+v", c)
		}
		for _, cause := range c.Causes {
			if cause.Edge == nil && seedMax[c.ID] != c.Risk {
				t.Fatalf("seed cause below final risk: %+v", c)
			}
			if cause.Edge != nil && ref.final[cause.Edge.From]-cause.Edge.Attenuation != c.Risk {
				t.Fatalf("edge cause does not support final risk: %+v", c)
			}
		}
	}
	kinds := map[string]string{}
	for _, n := range s.Nodes {
		kinds[n.ID] = n.Kind
	}
	targets := []report.TestTarget{}
	for id, v := range ref.final {
		if kinds[id] == "test" && v > 0 {
			targets = append(targets, report.TestTarget{ID: id, Risk: v})
		}
	}
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].Risk != targets[j].Risk {
			return targets[i].Risk > targets[j].Risk
		}
		return targets[i].ID < targets[j].ID
	})
	if !reflect.DeepEqual(r.TestTargets, targets) {
		t.Fatalf("test targets %v, reference %v", r.TestTargets, targets)
	}
}

func TestGenericGraphScenarios(t *testing.T) {
	seed := func(id string, risk int) snapshot.Seed { return snapshot.Seed{ID: id, Risk: risk, Reason: "seeded"} }
	edge := func(from, to, kind string, a int) graph.Edge {
		return graph.Edge{From: from, To: to, Kind: kind, Attenuation: a}
	}
	cases := []struct {
		name        string
		s           snapshot.Snapshot
		want        map[string]int
		generations int
	}{
		{"empty", generic(16, nil, nil, nil), map[string]int{}, 1},
		{"single", generic(16, nodesOf("server", "web"), nil, []snapshot.Seed{seed("web", 2)}), map[string]int{"web": 2}, 1},
		{"linear_chain", generic(16, nodesOf("service", "api", "db", "cache", "disk"),
			[]graph.Edge{edge("api", "db", "depends_on", 1), edge("db", "cache", "depends_on", 1), edge("cache", "disk", "depends_on", 1)},
			[]snapshot.Seed{seed("api", 3)}), map[string]int{"api": 3, "db": 2, "cache": 1, "disk": 0}, 3},
		{"branching_mixed_attenuation", generic(16, nodesOf("component", "root", "a0", "a1", "a2", "a3"),
			[]graph.Edge{edge("root", "a0", "connects_to", 0), edge("root", "a1", "connects_to", 1), edge("root", "a2", "connects_to", 2), edge("root", "a3", "connects_to", 3)},
			[]snapshot.Seed{seed("root", 3)}), map[string]int{"root": 3, "a0": 3, "a1": 2, "a2": 1, "a3": 0}, 2},
		{"zero_attenuation_cycle", generic(16, nodesOf("job", "a", "b", "c"),
			[]graph.Edge{edge("a", "b", "triggers", 0), edge("b", "c", "triggers", 0), edge("c", "a", "triggers", 0)},
			[]snapshot.Seed{seed("a", 2)}), map[string]int{"a": 2, "b": 2, "c": 2}, 3},
		{"self_loop", generic(16, nodesOf("task", "a", "b"),
			[]graph.Edge{edge("a", "a", "retries", 0), edge("b", "b", "retries", 0)},
			[]snapshot.Seed{seed("a", 1)}), map[string]int{"a": 1, "b": 0}, 1},
		{"disconnected_components", generic(16, nodesOf("server", "a", "b", "x", "y", "z"),
			[]graph.Edge{edge("a", "b", "replicates_to", 1), edge("x", "y", "replicates_to", 2)},
			[]snapshot.Seed{seed("a", 3), seed("x", 3)}), map[string]int{"a": 3, "b": 2, "x": 3, "y": 1, "z": 0}, 2},
		{"multiple_seeds", generic(16, nodesOf("task", "a", "b", "c"),
			[]graph.Edge{edge("a", "c", "blocks", 2), edge("b", "c", "blocks", 0)},
			[]snapshot.Seed{seed("a", 3), seed("b", 2)}), map[string]int{"a": 3, "b": 2, "c": 2}, 2},
		{"duplicate_seeds", generic(16, nodesOf("task", "a"), nil,
			[]snapshot.Seed{seed("a", 1), seed("a", 3), seed("a", 3), {ID: "a", Risk: 2, Reason: "other"}}), map[string]int{"a": 3}, 1},
		{"risk_zero_seed", generic(16, nodesOf("task", "a", "b"), []graph.Edge{edge("a", "b", "blocks", 0)},
			[]snapshot.Seed{seed("a", 0)}), map[string]int{"a": 0, "b": 0}, 1},
	}
	for a := 0; a <= 3; a++ {
		gens := 2
		if a == 3 {
			gens = 1
		}
		cases = append(cases, struct {
			name        string
			s           snapshot.Snapshot
			want        map[string]int
			generations int
		}{fmt.Sprintf("attenuation_%d", a), generic(16, nodesOf("node", "src", "dst"), []graph.Edge{edge("src", "dst", "link", a)}, []snapshot.Seed{seed("src", 3)}), map[string]int{"src": 3, "dst": 3 - a}, gens})
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			r, err := risk.Run(tt.s)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(risks(r), tt.want) || r.GenerationCount != tt.generations || r.Status != "complete" || r.StopReason != "converged" {
				t.Fatalf("got %v generations %d status %s/%s", risks(r), r.GenerationCount, r.Status, r.StopReason)
			}
			if len(r.TestTargets) != 0 {
				t.Fatalf("generic graph produced test targets: %v", r.TestTargets)
			}
			compareWithReference(t, tt.s, r)
		})
	}
}

func TestGenericRejectsInvalidAttenuation(t *testing.T) {
	for _, a := range []int{-1, 4, 5, 1000} {
		s := generic(16, nodesOf("node", "a", "b"), []graph.Edge{{From: "a", To: "b", Kind: "link", Attenuation: a}}, nil)
		if _, err := risk.Run(s); err == nil {
			t.Fatalf("attenuation %d accepted", a)
		}
	}
}

func TestEqualRiskSourcesListEveryCause(t *testing.T) {
	s := generic(16, nodesOf("service", "a", "b", "c", "d"), []graph.Edge{
		{From: "a", To: "d", Kind: "depends_on", Attenuation: 1},
		{From: "b", To: "d", Kind: "calls", Attenuation: 0},
		{From: "c", To: "d", Kind: "depends_on", Attenuation: 2},
	}, []snapshot.Seed{{ID: "a", Risk: 3, Reason: "outage"}, {ID: "b", Risk: 2, Reason: "degraded"}, {ID: "c", Risk: 3, Reason: "outage"}})
	r, err := risk.Run(s)
	if err != nil {
		t.Fatal(err)
	}
	d := r.Cells[3]
	if d.ID != "d" || d.Risk != 2 || len(d.Causes) != 2 || d.Causes[0].Source != "a" || d.Causes[1].Source != "b" || d.Causes[1].Reason != "calls" {
		t.Fatalf("causes: %+v", d)
	}
	compareWithReference(t, s, r)
}

// A zero-attenuation cycle can support its own final value, but the waves
// always trace back to a generation 0 seed.
func TestZeroAttenuationCycleCauses(t *testing.T) {
	s := generic(16, nodesOf("job", "a", "b"), []graph.Edge{{From: "a", To: "b", Kind: "triggers", Attenuation: 0}, {From: "b", To: "a", Kind: "triggers", Attenuation: 0}}, []snapshot.Seed{{ID: "a", Risk: 3, Reason: "failed"}})
	r, err := risk.Run(s)
	if err != nil {
		t.Fatal(err)
	}
	a := r.Cells[0]
	if len(a.Causes) != 2 || a.Causes[0].Source != "a" || a.Causes[0].Edge != nil || a.Causes[1].Source != "b" || a.Causes[1].Edge == nil {
		t.Fatalf("final support of a: %+v", a.Causes)
	}
	if len(r.Waves) != 2 || r.Waves[0].Changes[0].ID != "a" || r.Waves[1].Changes[0].ID != "b" || r.Waves[1].Changes[0].Causes[0].Source != "a" {
		t.Fatalf("waves: %+v", r.Waves)
	}
	compareWithReference(t, s, r)
}

func zeroChain(n, limit int) snapshot.Snapshot {
	ids := make([]string, n)
	for i := range ids {
		ids[i] = fmt.Sprintf("n%05d", i)
	}
	edges := []graph.Edge{}
	for i := 0; i+1 < n; i++ {
		edges = append(edges, graph.Edge{From: ids[i], To: ids[i+1], Kind: "feeds", Attenuation: 0})
	}
	return generic(limit, nodesOf("stage", ids...), edges, []snapshot.Seed{{ID: ids[0], Risk: 3, Reason: "start"}})
}

// A chain of n nodes needs n-1 changing updates plus one confirming update.
func TestZeroAttenuationChainCompletesOrStopsAtLimit(t *testing.T) {
	for _, tt := range []struct {
		n, limit, generations int
		status, stop          string
		last                  int
	}{
		{1, 1, 1, "complete", "converged", 3},
		{2, 1, 1, "incomplete", "generation_limit", 3},
		{2, 2, 2, "complete", "converged", 3},
		{50, 1024, 50, "complete", "converged", 3},
		{1024, 1024, 1024, "complete", "converged", 3},
		// The last node reaches 3 in update 1024, but no update confirms a
		// fixed point, so the result stays incomplete.
		{1025, 1024, 1024, "incomplete", "generation_limit", 3},
		{1026, 1024, 1024, "incomplete", "generation_limit", 0},
		{3000, 1024, 1024, "incomplete", "generation_limit", 0},
	} {
		t.Run(fmt.Sprintf("n=%d,limit=%d", tt.n, tt.limit), func(t *testing.T) {
			s := zeroChain(tt.n, tt.limit)
			r, err := risk.Run(s)
			if err != nil {
				t.Fatal(err)
			}
			last := r.Cells[len(r.Cells)-1]
			if r.GenerationCount != tt.generations || r.Status != tt.status || r.StopReason != tt.stop || last.Risk != tt.last {
				t.Fatalf("generations=%d status=%s stop=%s last=%d", r.GenerationCount, r.Status, r.StopReason, last.Risk)
			}
			if tt.status == "incomplete" && (len(r.Warnings) != 1 || !strings.Contains(r.Warnings[0], "generation limit")) {
				t.Fatalf("warnings: %v", r.Warnings)
			}
			compareWithReference(t, s, r)
		})
	}
}

// When the generation limit stops an unconverged run, final causes are
// evaluated against the last state, which is not a fixed point. A cell can then
// hold a positive risk whose recorded support has since been superseded; its
// wave still records the cause. The result is incomplete (exit 2), never
// complete.
func TestIncompleteFinalCausesMayBeEmpty(t *testing.T) {
	s := generic(2, nodesOf("node", "P", "Q", "R", "U", "V"), []graph.Edge{
		{From: "P", To: "U", Kind: "e", Attenuation: 0},
		{From: "Q", To: "R", Kind: "e", Attenuation: 0},
		{From: "R", To: "U", Kind: "e", Attenuation: 0},
		{From: "U", To: "V", Kind: "e", Attenuation: 0},
	}, []snapshot.Seed{{ID: "P", Risk: 1, Reason: "p"}, {ID: "Q", Risk: 3, Reason: "q"}})
	r, err := risk.Run(s)
	if err != nil {
		t.Fatal(err)
	}
	v := r.Cells[4]
	if r.Status != "incomplete" || v.ID != "V" || v.Risk != 1 || len(v.Causes) != 0 {
		t.Fatalf("status=%s cell=%+v", r.Status, v)
	}
	last := r.Waves[len(r.Waves)-1]
	if last.Generation != 2 || len(last.Changes) != 2 || last.Changes[1].ID != "V" || last.Changes[1].Causes[0].Source != "U" {
		t.Fatalf("wave: %+v", last)
	}
	compareWithReference(t, s, r)
}

func TestNonSoftwareSupplyChain(t *testing.T) {
	s := generic(16, []graph.Node{
		{ID: "mine", Kind: "supplier"}, {ID: "smelter", Kind: "facility"}, {ID: "port", Kind: "warehouse"},
		{ID: "factory", Kind: "facility"}, {ID: "store", Kind: "retailer"}, {ID: "backup", Kind: "supplier"},
	}, []graph.Edge{
		{From: "mine", To: "smelter", Kind: "ships_to", Attenuation: 0},
		{From: "smelter", To: "port", Kind: "ships_to", Attenuation: 1},
		{From: "port", To: "factory", Kind: "ships_to", Attenuation: 0},
		{From: "backup", To: "factory", Kind: "ships_to", Attenuation: 2},
		{From: "factory", To: "store", Kind: "delivers_to", Attenuation: 1},
	}, []snapshot.Seed{{ID: "mine", Risk: 3, Reason: "flood"}, {ID: "backup", Risk: 3, Reason: "strike"}})
	r, err := risk.Run(s)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"mine": 3, "smelter": 3, "port": 2, "factory": 2, "store": 1, "backup": 3}
	if !reflect.DeepEqual(risks(r), want) || r.GenerationCount != 5 || r.Status != "complete" {
		t.Fatalf("got %v generations %d", risks(r), r.GenerationCount)
	}
	compareWithReference(t, s, r)
}

func randomGeneric(rng *rand.Rand, n, edges, limit int) snapshot.Snapshot {
	kinds := []string{"service", "db", "test", "package", "倉庫", "job"}
	edgeKinds := []string{"depends_on", "ships_to", "own_test", "triggers"}
	nodes := make([]graph.Node, n)
	for i := range nodes {
		nodes[i] = graph.Node{ID: fmt.Sprintf("v%d", i), Kind: kinds[rng.Intn(len(kinds))]}
	}
	used := map[[2]int]graph.Edge{}
	es := []graph.Edge{}
	for i := 0; i < edges && n > 0; i++ {
		from, to := rng.Intn(n), rng.Intn(n)
		if e, ok := used[[2]int{from, to}]; ok {
			es = append(es, e)
			continue
		}
		e := graph.Edge{From: nodes[from].ID, To: nodes[to].ID, Kind: edgeKinds[rng.Intn(len(edgeKinds))], Attenuation: rng.Intn(4)}
		used[[2]int{from, to}] = e
		es = append(es, e)
	}
	seeds := []snapshot.Seed{}
	for i := 0; i < 1+rng.Intn(4) && n > 0; i++ {
		seeds = append(seeds, snapshot.Seed{ID: nodes[rng.Intn(n)].ID, Risk: rng.Intn(4), Reason: fmt.Sprint("r", rng.Intn(2))})
	}
	return generic(limit, nodes, es, seeds)
}

func TestGenericMatchesReferenceWhenRandomGraphs(t *testing.T) {
	rng := rand.New(rand.NewSource(20261010))
	for i := 0; i < 1000; i++ {
		n := rng.Intn(16)
		s := randomGeneric(rng, n, rng.Intn(3*n+1), 1+rng.Intn(20))
		r, err := risk.Run(s)
		if err != nil {
			t.Fatal(err)
		}
		compareWithReference(t, s, r)
		if r.Status == "complete" && r.GenerationCount > max(1, len(s.Nodes)) {
			t.Fatalf("generation bound max(1,|V|) exceeded: %d > %d", r.GenerationCount, len(s.Nodes))
		}
	}
}

func TestGenericLargeSparseGraph(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	s := randomGeneric(rng, graph.MaxNodes, 2*graph.MaxNodes, snapshot.MaxGenerations)
	for i := 0; i < 200; i++ {
		s.Seeds = append(s.Seeds, snapshot.Seed{ID: s.Nodes[rng.Intn(len(s.Nodes))].ID, Risk: 3, Reason: "incident"})
	}
	r, err := risk.Run(s)
	if err != nil {
		t.Fatal(err)
	}
	compareWithReference(t, s, r)
}

func TestGenericDeterminismWhenInputOrderChanges(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	s := randomGeneric(rng, 40, 120, 64)
	s.Seeds = append(s.Seeds, s.Seeds...)
	s.Edges = append(s.Edges, s.Edges[:10]...)
	baseline, err := risk.Run(s)
	if err != nil {
		t.Fatal(err)
	}
	want, err := report.JSON(baseline)
	if err != nil {
		t.Fatal(err)
	}
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
		if !bytes.Equal(b, want) || got.InputSHA256 != baseline.InputSHA256 {
			t.Fatalf("non-deterministic iteration %d", i)
		}
	}
	for i := 0; i < 25; i++ {
		got, err := risk.Run(s)
		if err != nil {
			t.Fatal(err)
		}
		if b, _ := report.JSON(got); !bytes.Equal(b, want) {
			t.Fatalf("repeated run %d differs", i)
		}
	}
}

func FuzzGenericPropagation(f *testing.F) {
	f.Add([]byte{3, 1, 2, 0, 2, 9, 3, 8, 1, 7})
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, b []byte) {
		n := 1 + len(b)%24
		ids := make([]string, n)
		for i := range ids {
			ids[i] = fmt.Sprint(i)
		}
		s := generic(1+len(b)%40, nodesOf("cell", ids...), nil, nil)
		used := map[[2]string]bool{}
		for i := 0; i+3 < len(b) && i < 400; i += 4 {
			from, to := ids[int(b[i])%n], ids[int(b[i+1])%n]
			if !used[[2]string{from, to}] {
				used[[2]string{from, to}] = true
				s.Edges = append(s.Edges, graph.Edge{From: from, To: to, Kind: fmt.Sprint("k", b[i+3]%3), Attenuation: int(b[i+2]) % 4})
			}
			s.Seeds = append(s.Seeds, snapshot.Seed{ID: from, Risk: int(b[i+3]) % 4, Reason: "fuzz"})
		}
		r, err := risk.Run(s)
		if err != nil {
			t.Fatal(err)
		}
		compareWithReference(t, s, r)
	})
}
