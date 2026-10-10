package risk

import (
	"github.com/cottondesu/stateripple/internal/graph"
	"github.com/cottondesu/stateripple/internal/report"
	"github.com/cottondesu/stateripple/internal/snapshot"
)

type incoming struct {
	index int
	edge  graph.Edge
}

type rule struct {
	seeds   [][]snapshot.Seed
	initial []int
	edges   [][]incoming
}

func (r rule) Next(i int, previous []int) int {
	next := max(previous[i], r.initial[i])
	for _, in := range r.edges[i] {
		next = max(next, previous[in.index]-in.edge.Attenuation)
	}
	return next
}

func (r rule) causes(i, value int, previous []int) []report.Cause {
	causes := []report.Cause{}
	if value == 0 {
		return causes
	}
	for _, s := range r.seeds[i] {
		if s.Risk == value {
			causes = append(causes, report.Cause{Source: s.ID, Reason: s.Reason})
		}
	}
	for _, in := range r.edges[i] {
		if previous[in.index]-in.edge.Attenuation == value {
			e := in.edge
			causes = append(causes, report.Cause{Source: e.From, Edge: &e, Reason: e.Kind})
		}
	}
	return causes
}
