package adapters

import (
	"context"
	"crypto/sha1"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func BenchmarkHeadVerification(b *testing.B) {
	for _, n := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			root := b.TempDir()
			data := []byte(strings.Repeat("x", 1024))
			hash := sha1.New()
			if _, err := fmt.Fprintf(hash, "blob %d\x00", len(data)); err != nil {
				b.Fatal(err)
			}
			if _, err := hash.Write(data); err != nil {
				b.Fatal(err)
			}
			oid := fmt.Sprintf("%x", hash.Sum(nil))
			files := make([]headFile, n)
			for i := range files {
				path := fmt.Sprintf("file-%05d.go", i)
				if err := os.WriteFile(filepath.Join(root, path), data, 0644); err != nil {
					b.Fatal(err)
				}
				files[i] = headFile{path: path, mode: "100644", oid: oid}
			}
			b.ReportAllocs()
			b.SetBytes(int64(n * len(data) * 2))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				m := Metadata{}
				for pass := 0; pass < 2; pass++ {
					if err := verifyHeadFiles(context.Background(), root, files, &m); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}
