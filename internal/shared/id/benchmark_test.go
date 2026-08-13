package id

import "testing"

func BenchmarkSnowflakeNextID(b *testing.B) {
	gen, err := NewSnowflake(1)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := gen.NextID(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUUIDNextID(b *testing.B) {
	gen := NewUUIDGenerator()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := gen.NextID(); err != nil {
			b.Fatal(err)
		}
	}
}
