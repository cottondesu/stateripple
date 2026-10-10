package graph

import (
	"fmt"
	"sort"
	"unicode/utf8"
)

const (
	MaxNodes  = 20000
	MaxEdges  = 200000
	MaxString = 4096
	// MaxAttenuation equals the maximum risk of RiskPropagationV1, so an edge
	// with attenuation 3 never contributes. Larger values are rejected rather
	// than clamped because they carry no additional meaning.
	MaxAttenuation = 3
)

// Node and edge kinds are free-form labels. Graph validation and propagation
// assign them no meaning, and adapters that need a fixed vocabulary validate it
// themselves. One exception lives outside this package: the risk rule projects
// nodes whose kind is exactly "test" into the report's test_targets.

type Node struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

type Edge struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Kind        string `json:"kind"`
	Attenuation int    `json:"attenuation"`
}

func Less(a, b Edge) bool {
	if a.From != b.From {
		return a.From < b.From
	}
	if a.To != b.To {
		return a.To < b.To
	}
	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	return a.Attenuation < b.Attenuation
}

func ValidString(s string) bool { return s != "" && len(s) <= MaxString && utf8.ValidString(s) }

func Normalize(nodes []Node, edges []Edge) ([]Node, []Edge, error) {
	if len(nodes) > MaxNodes || len(edges) > MaxEdges {
		return nil, nil, fmt.Errorf("graph size exceeds limits")
	}
	ns := append([]Node{}, nodes...)
	es := append([]Edge{}, edges...)
	sort.Slice(ns, func(i, j int) bool { return ns[i].ID < ns[j].ID })
	ids := make(map[string]bool, len(ns))
	for _, n := range ns {
		if !ValidString(n.ID) {
			return nil, nil, fmt.Errorf("invalid node ID")
		}
		if !ValidString(n.Kind) {
			return nil, nil, fmt.Errorf("invalid node kind %q", n.Kind)
		}
		if ids[n.ID] {
			return nil, nil, fmt.Errorf("duplicate node %q", n.ID)
		}
		ids[n.ID] = true
	}
	sort.Slice(es, func(i, j int) bool { return Less(es[i], es[j]) })
	out := make([]Edge, 0, len(es))
	for _, e := range es {
		if !ids[e.From] || !ids[e.To] {
			return nil, nil, fmt.Errorf("edge endpoint not found: %q -> %q", e.From, e.To)
		}
		if !ValidString(e.Kind) {
			return nil, nil, fmt.Errorf("invalid edge kind %q", e.Kind)
		}
		if e.Attenuation < 0 || e.Attenuation > MaxAttenuation {
			return nil, nil, fmt.Errorf("invalid attenuation %d: must be 0..%d", e.Attenuation, MaxAttenuation)
		}
		if len(out) > 0 {
			prev := out[len(out)-1]
			if prev.From == e.From && prev.To == e.To && prev.Kind == e.Kind {
				if prev.Attenuation != e.Attenuation {
					return nil, nil, fmt.Errorf("contradictory duplicate edge")
				}
				continue
			}
			if prev.From == e.From && prev.To == e.To {
				return nil, nil, fmt.Errorf("conflicting edge kinds for endpoints")
			}
		}
		out = append(out, e)
	}
	return ns, out, nil
}
