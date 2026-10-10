package report_test

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"os"
	"testing"

	"github.com/cottondesu/stateripple/internal/report"
)

func TestJSONMetadataAndTiedCausesAreOrderIndependent(t *testing.T) {
	data, err := os.ReadFile("../../testdata/example.report.json")
	if err != nil {
		t.Fatal(err)
	}
	var r report.Report
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatal(err)
	}
	r.Warnings = []string{"z-warning", "a-warning", "z-warning"}
	r.UnknownChanges = []string{"z-file", "a-file", "z-file"}
	r.Cells[0].Causes = append(r.Cells[0].Causes, report.Cause{Source: "other", Reason: "tie"})
	want, err := report.JSON(r)
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewSource(9008))
	for i := 0; i < 200; i++ {
		rng.Shuffle(len(r.Warnings), func(a, b int) { r.Warnings[a], r.Warnings[b] = r.Warnings[b], r.Warnings[a] })
		rng.Shuffle(len(r.UnknownChanges), func(a, b int) { r.UnknownChanges[a], r.UnknownChanges[b] = r.UnknownChanges[b], r.UnknownChanges[a] })
		rng.Shuffle(len(r.Cells), func(a, b int) { r.Cells[a], r.Cells[b] = r.Cells[b], r.Cells[a] })
		for _, cell := range r.Cells {
			rng.Shuffle(len(cell.Causes), func(a, b int) { cell.Causes[a], cell.Causes[b] = cell.Causes[b], cell.Causes[a] })
		}
		rng.Shuffle(len(r.TestTargets), func(a, b int) { r.TestTargets[a], r.TestTargets[b] = r.TestTargets[b], r.TestTargets[a] })
		rng.Shuffle(len(r.Waves), func(a, b int) { r.Waves[a], r.Waves[b] = r.Waves[b], r.Waves[a] })
		for _, wave := range r.Waves {
			rng.Shuffle(len(wave.Changes), func(a, b int) { wave.Changes[a], wave.Changes[b] = wave.Changes[b], wave.Changes[a] })
		}
		got, err := report.JSON(r)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("iteration=%d equal=%t error=%v", i, bytes.Equal(got, want), err)
		}
	}
}
