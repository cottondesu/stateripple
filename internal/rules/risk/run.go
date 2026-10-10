package risk

import (
	"sort"

	"github.com/cottondesu/stateripple/internal/engine"
	"github.com/cottondesu/stateripple/internal/report"
	"github.com/cottondesu/stateripple/internal/snapshot"
)

func Run(input snapshot.Snapshot) (report.Report, error) {
	s, err := snapshot.Normalize(input)
	if err != nil {
		return report.Report{}, err
	}
	hash, err := snapshot.Hash(s)
	if err != nil {
		return report.Report{}, err
	}
	r := rule{seeds: make([][]snapshot.Seed, len(s.Nodes)), initial: make([]int, len(s.Nodes)), edges: make([][]incoming, len(s.Nodes))}
	indices := make(map[string]int, len(s.Nodes))
	for i, n := range s.Nodes {
		indices[n.ID] = i
		r.seeds[i] = []snapshot.Seed{}
	}
	for _, seed := range s.Seeds {
		i := indices[seed.ID]
		r.seeds[i] = append(r.seeds[i], seed)
		r.initial[i] = max(r.initial[i], seed.Risk)
	}
	for _, e := range s.Edges {
		i := indices[e.To]
		r.edges[i] = append(r.edges[i], incoming{indices[e.From], e})
	}
	result, err := engine.Run[int](r.initial, r, s.MaxGenerations)
	if err != nil {
		return report.Report{}, err
	}
	out := report.Report{SchemaVersion: 1, Ruleset: s.Ruleset, Status: "complete", InputSHA256: hash, GenerationCount: result.GenerationCount, StopReason: "converged", Cells: []report.Cell{}, TestTargets: []report.TestTarget{}, Waves: []report.Wave{}, UnknownChanges: []string{}, Warnings: []string{}, Changes: []report.FileChange{}}
	if !result.Converged {
		out.Status = "incomplete"
		out.StopReason = "generation_limit"
		out.Warnings = append(out.Warnings, "generation limit reached without confirming a fixed point")
	}
	first := make([]*int, len(s.Nodes))
	initialWave := report.Wave{Generation: 0, Changes: []report.Change{}}
	for i, value := range r.initial {
		if value > 0 {
			g := 0
			first[i] = &g
			causes := []report.Cause{}
			for _, seed := range r.seeds[i] {
				if seed.Risk == value {
					causes = append(causes, report.Cause{Source: seed.ID, Reason: seed.Reason})
				}
			}
			initialWave.Changes = append(initialWave.Changes, report.Change{ID: s.Nodes[i].ID, Before: 0, After: value, Causes: causes})
		}
	}
	if len(initialWave.Changes) > 0 {
		out.Waves = append(out.Waves, initialWave)
	}
	previous := append([]int{}, r.initial...)
	for _, wave := range result.Waves {
		w := report.Wave{Generation: wave.Generation, Changes: []report.Change{}}
		for _, c := range wave.Changes {
			if first[c.Index] == nil && c.After > 0 {
				g := wave.Generation
				first[c.Index] = &g
			}
			w.Changes = append(w.Changes, report.Change{ID: s.Nodes[c.Index].ID, Before: c.Before, After: c.After, Causes: r.causes(c.Index, c.After, previous)})
		}
		for _, c := range wave.Changes {
			previous[c.Index] = c.After
		}
		out.Waves = append(out.Waves, w)
	}
	for i, n := range s.Nodes {
		value := result.State[i]
		out.Cells = append(out.Cells, report.Cell{ID: n.ID, Kind: n.Kind, Risk: value, Seeds: r.seeds[i], FirstActiveGeneration: first[i], Causes: r.causes(i, value, result.State)})
		if n.Kind == "test" && value > 0 {
			out.TestTargets = append(out.TestTargets, report.TestTarget{ID: n.ID, Risk: value})
		}
	}
	sort.Slice(out.TestTargets, func(i, j int) bool {
		if out.TestTargets[i].Risk != out.TestTargets[j].Risk {
			return out.TestTargets[i].Risk > out.TestTargets[j].Risk
		}
		return out.TestTargets[i].ID < out.TestTargets[j].ID
	})
	return out, nil
}
