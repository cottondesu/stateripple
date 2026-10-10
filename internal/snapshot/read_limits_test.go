package snapshot_test

import (
	"strings"
	"testing"

	"github.com/cottondesu/stateripple/internal/graph"
	"github.com/cottondesu/stateripple/internal/snapshot"
)

func TestReadCountsWhenArraysOversized(t *testing.T) {
	for _, tt := range []struct {
		field string
		limit int
	}{
		{"nodes", graph.MaxNodes}, {"edges", graph.MaxEdges}, {"seeds", snapshot.MaxSeeds},
	} {
		t.Run(tt.field, func(t *testing.T) {
			array := "[" + strings.Repeat("{},", tt.limit) + "{}]"
			input := strings.Replace(empty, `"`+tt.field+`":[]`, `"`+tt.field+`":`+array, 1)
			_, err := snapshot.Read(strings.NewReader(input))
			if err == nil || !strings.Contains(err.Error(), "element limit") {
				t.Fatalf("not bounded before object allocation: %v", err)
			}
		})
	}
}
