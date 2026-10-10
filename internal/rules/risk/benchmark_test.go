package risk_test

import (
	"fmt"
	"testing"

	"github.com/cottondesu/stateripple/internal/rules/risk"
	"github.com/cottondesu/stateripple/internal/snapshot"
)

func BenchmarkPropagation(b *testing.B) {
	for _, n := range []int{100, 1000, 10000} {
		for _, multiple := range []bool{false, true} {
			b.Run(fmt.Sprintf("nodes=%d/multiple=%t", n, multiple), func(b *testing.B) {
				ids := make([]string, n)
				pairs := make([][2]string, 0, n-1)
				for i := range ids {
					ids[i] = fmt.Sprintf("pkg:%05d", i)
					if i > 0 {
						pairs = append(pairs, [2]string{ids[i-1], ids[i]})
					}
				}
				s := fixture(ids, pairs, []snapshot.Seed{{ID: ids[0], Risk: 3, Reason: "source"}})
				if multiple {
					for i := 10; i < n; i += 10 {
						s.Seeds = append(s.Seeds, snapshot.Seed{ID: ids[i], Risk: 3, Reason: "source"})
					}
				}
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					r, err := risk.Run(s)
					if err != nil {
						b.Fatal(err)
					}
					b.ReportMetric(float64(r.GenerationCount), "generations/op")
				}
				b.ReportMetric(float64(len(s.Edges)), "edges/op")
			})
		}
	}
}
