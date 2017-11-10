package gcache

import (
	"fmt"
	"testing"
)

func BenchmarkHash(b *testing.B) {
	benchmarks := []struct {
		name string
		fn   HashCalculator
	}{
		{"FNV", calcHashFNV},
		{"CRC", calcHashCRC},
		{"dj33", djb33},
		{"SUM", calcSUM},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				bm.fn(fmt.Sprintf("testHash%d-%d", i, i))
			}
		})
	}
}
