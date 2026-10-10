package report

import (
	"fmt"
	"slices"
	"strings"

	"github.com/cottondesu/stateripple/internal/canonical"
	"github.com/cottondesu/stateripple/internal/snapshot"
)

func JSON(r Report) ([]byte, error) {
	r = ordered(r)
	b, err := canonical.JSON(r)
	if err != nil {
		return nil, fmt.Errorf("marshal report: %w", err)
	}
	return b, nil
}

func ordered(r Report) Report {
	r.Cells = append([]Cell{}, r.Cells...)
	for i := range r.Cells {
		c := &r.Cells[i]
		c.Seeds = append([]snapshot.Seed{}, c.Seeds...)
		slices.SortFunc(c.Seeds, func(a, b snapshot.Seed) int {
			if n := strings.Compare(a.ID, b.ID); n != 0 {
				return n
			}
			if a.Risk != b.Risk {
				return b.Risk - a.Risk
			}
			return strings.Compare(a.Reason, b.Reason)
		})
		c.Causes = orderCauses(c.Causes)
	}
	slices.SortFunc(r.Cells, func(a, b Cell) int { return strings.Compare(a.ID, b.ID) })
	r.TestTargets = append([]TestTarget{}, r.TestTargets...)
	slices.SortFunc(r.TestTargets, func(a, b TestTarget) int {
		if a.Risk != b.Risk {
			return b.Risk - a.Risk
		}
		return strings.Compare(a.ID, b.ID)
	})
	r.Waves = append([]Wave{}, r.Waves...)
	for i := range r.Waves {
		w := &r.Waves[i]
		w.Changes = append([]Change{}, w.Changes...)
		for j := range w.Changes {
			w.Changes[j].Causes = orderCauses(w.Changes[j].Causes)
		}
		slices.SortFunc(w.Changes, func(a, b Change) int { return strings.Compare(a.ID, b.ID) })
	}
	slices.SortFunc(r.Waves, func(a, b Wave) int { return a.Generation - b.Generation })
	r.UnknownChanges = orderStrings(r.UnknownChanges)
	r.Warnings = orderStrings(r.Warnings)
	r.Changes = append([]FileChange{}, r.Changes...)
	slices.SortFunc(r.Changes, func(a, b FileChange) int {
		if n := strings.Compare(a.Path, b.Path); n != 0 {
			return n
		}
		if n := strings.Compare(a.Status, b.Status); n != 0 {
			return n
		}
		return strings.Compare(a.Classification, b.Classification)
	})
	if r.BuildContext != nil {
		bc := *r.BuildContext
		bc.BuildTags = orderStrings(bc.BuildTags)
		bc.ToolTags = orderStrings(bc.ToolTags)
		bc.ReleaseTags = orderStrings(bc.ReleaseTags)
		r.BuildContext = &bc
	}
	return r
}

func orderStrings(values []string) []string {
	values = append([]string{}, values...)
	slices.Sort(values)
	return slices.Compact(values)
}

func orderCauses(values []Cause) []Cause {
	values = append([]Cause{}, values...)
	slices.SortFunc(values, func(a, b Cause) int {
		if n := strings.Compare(a.Source, b.Source); n != 0 {
			return n
		}
		if n := strings.Compare(a.Reason, b.Reason); n != 0 {
			return n
		}
		if a.Edge == nil {
			if b.Edge == nil {
				return 0
			}
			return -1
		}
		if b.Edge == nil {
			return 1
		}
		if n := strings.Compare(a.Edge.From, b.Edge.From); n != 0 {
			return n
		}
		if n := strings.Compare(a.Edge.To, b.Edge.To); n != 0 {
			return n
		}
		if n := strings.Compare(a.Edge.Kind, b.Edge.Kind); n != 0 {
			return n
		}
		return a.Edge.Attenuation - b.Edge.Attenuation
	})
	return values
}
