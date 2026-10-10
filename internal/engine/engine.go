package engine

import "fmt"

// Rule reads only the immutable previous generation. It must not modify it.
type Rule[S comparable] interface {
	Next(index int, previous []S) S
}

type Change[S comparable] struct {
	Index  int
	Before S
	After  S
}

type Wave[S comparable] struct {
	Generation int
	Changes    []Change[S]
}

type Result[S comparable] struct {
	State           []S
	Waves           []Wave[S]
	GenerationCount int
	Converged       bool
}

func Run[S comparable](initial []S, rule Rule[S], limit int) (Result[S], error) {
	if limit < 1 || limit > 1024 {
		return Result[S]{}, fmt.Errorf("update limit must be 1..1024")
	}
	current := append([]S{}, initial...)
	result := Result[S]{Waves: []Wave[S]{}}
	for generation := 1; generation <= limit; generation++ {
		next := make([]S, len(current))
		changes := []Change[S]{}
		for i, value := range current {
			next[i] = rule.Next(i, current)
			if next[i] != value {
				changes = append(changes, Change[S]{i, value, next[i]})
			}
		}
		result.GenerationCount = generation
		current = next
		if len(changes) == 0 {
			result.Converged = true
			break
		}
		result.Waves = append(result.Waves, Wave[S]{generation, changes})
	}
	result.State = current
	return result, nil
}
