package risk_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/cottondesu/stateripple/internal/report"
	"github.com/cottondesu/stateripple/internal/rules/risk"
	"github.com/cottondesu/stateripple/internal/snapshot"
)

func TestChainGoldenWhenSnapshotLoaded(t *testing.T) {
	input, err := os.ReadFile("../../../testdata/example.json")
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("../../../testdata/example.report.json")
	if err != nil {
		t.Fatal(err)
	}
	s, err := snapshot.Read(bytes.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	r, err := risk.Run(s)
	if err != nil {
		t.Fatal(err)
	}
	got, err := report.JSON(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("golden report differs:\n%s", got)
	}
}
