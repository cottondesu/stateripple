package engine_test

import (
	"github.com/cottondesu/stateripple/internal/engine"
	"testing"
)

type boolRule struct{}

func (boolRule) Next(i int, previous []bool) bool { return previous[i] || i > 0 && previous[i-1] }

func TestGenericStateWhenBooleanWavePropagates(t *testing.T) {
	initial := []bool{true, false, false}
	r, err := engine.Run(initial, boolRule{}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Converged || r.GenerationCount != 3 || len(r.Waves) != 2 || len(r.Waves[0].Changes) != 1 || r.Waves[0].Changes[0].Index != 1 || initial[1] {
		t.Fatalf("synchronous result: %+v", r)
	}
}
