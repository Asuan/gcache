package gcache

import (
	"fmt"
	"testing"
)

func benchmarkHash(n int, b *testing.B) {
	b.StopTimer()
	v := ""
	for i := 0; i < n; i++ {
		v += fmt.Sprintf("-%d", i%10)
	}
	benchmarks := []struct {
		name string
		fn   HashCalculator
	}{
		{"calcHashFNV", calcHashFNV},
		{"calcHashCRC", calcHashCRC},
		{"dj33", djb33},
		{"calcSUM", calcSUM},
	}
	b.StartTimer()
	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				bm.fn(v)
			}
		})
	}
}

//Benchmarks of hash calculators
func BenchmarkHash2(b *testing.B)      { benchmarkHash(2, b) }
func BenchmarkHash10(b *testing.B)     { benchmarkHash(10, b) }
func BenchmarkHash100(b *testing.B)    { benchmarkHash(100, b) }
func BenchmarkHash1000(b *testing.B)   { benchmarkHash(1000, b) }
func BenchmarkHash10000(b *testing.B)  { benchmarkHash(10000, b) }
func BenchmarkHash100000(b *testing.B) { benchmarkHash(100000, b) }
