package adapters

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func BenchmarkBaseSourceReading(b *testing.B) {
	for _, count := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			root := b.TempDir()
			for i := 0; i < count; i++ {
				data := fmt.Sprintf("package fixture\nvar Value%d = %d\n", i, i)
				if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("source-%05d.go", i)), []byte(data), 0o600); err != nil {
					b.Fatal(err)
				}
			}
			for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "-qm", "fixture"}} {
				cmd := exec.Command("git", args...)
				cmd.Dir = root
				if out, err := cmd.CombinedOutput(); err != nil {
					b.Fatalf("fixture: %v: %s", err, out)
				}
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := formerEmbeds(context.Background(), root, "HEAD"); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
