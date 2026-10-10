package snapshot_test

import (
	"strings"
	"testing"

	"github.com/cottondesu/stateripple/internal/snapshot"
)

func TestReadWhenUnicodeMalformed(t *testing.T) {
	for _, id := range []string{"\xff", `\ud800`, `\udc00`, `\ud800\u0041`} {
		input := strings.Replace(empty, `"nodes":[]`, `"nodes":[{"id":"`+id+`","kind":"package"}]`, 1)
		_, err := snapshot.Read(strings.NewReader(input))
		if err == nil {
			t.Fatalf("invalid Unicode accepted: %q", id)
		}
	}
}

func TestReadWhenUnicodePairValid(t *testing.T) {
	input := strings.Replace(empty, `"nodes":[]`, `"nodes":[{"id":"\ud83d\ude00","kind":"package"}]`, 1)
	s, err := snapshot.Read(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if s.Nodes[0].ID != "😀" {
		t.Fatalf("decoded %q", s.Nodes[0].ID)
	}
}

func TestReadWhenFieldCaseAliases(t *testing.T) {
	for _, input := range []string{
		strings.Replace(empty, `"schema_version":1`, `"schema_version":1,"Max_Generations":1`, 1),
		strings.Replace(empty, `"nodes":[]`, `"nodes":[{"id":"A","ID":"B","kind":"package"}]`, 1),
		strings.Replace(empty, `"seeds":[]`, `"seeds":[{"id":"A","risk":2,"Risk":3,"reason":"change"}]`, 1),
	} {
		_, err := snapshot.Read(strings.NewReader(input))
		if err == nil {
			t.Fatal("case alias accepted")
		}
	}
}
