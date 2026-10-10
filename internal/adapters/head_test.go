package adapters

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestHeadVerificationAcrossReadBufferBoundaries(t *testing.T) {
	files := basicFiles()
	files["first.bin"] = strings.Repeat("abcdefg", 20000)
	files["second.bin"] = strings.Repeat("different", 17000)
	root, err := filepath.EvalSymlinks(fixture(t, files))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	inputs, err := headFiles(ctx, root, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	meta := Metadata{}
	if err := verifyHeadFiles(ctx, root, inputs, &meta); err != nil || meta.Incomplete {
		t.Fatalf("committed large blobs rejected: %v %+v", err, meta)
	}
	content := files["first.bin"]
	writeFile(t, root, "first.bin", content[:70000]+"X"+content[70001:])
	if err := verifyHeadFiles(ctx, root, inputs, &meta); err == nil {
		t.Fatal("same-size mutation beyond two buffer lengths missed")
	}
}
