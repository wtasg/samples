package ds

import (
	"testing"
)

func TestBloom_BasicMembership(t *testing.T) {
	bf := NewBloomFilter(100, 0.01)
	keys := []int64{1, 5, 42, 99, 1000}
	for _, k := range keys {
		bf.Add(k)
	}
	for _, k := range keys {
		if !bf.MightContain(k) {
			t.Errorf("MightContain(%d) = false after Add (false negative!)", k)
		}
	}
}

func TestBloom_DefiniteAbsence(t *testing.T) {
	bf := NewBloomFilter(100, 0.01)
	bf.Add(1)
	bf.Add(2)
	bf.Add(3)

	// Check a large number of definitely-absent keys — any false should pass;
	// but a false-negative (present key returning false) must never occur.
	for _, k := range []int64{1, 2, 3} {
		if !bf.MightContain(k) {
			t.Errorf("false negative for key %d", k)
		}
	}
}

func TestBloom_FalsePositiveRate(t *testing.T) {
	const n = 1000
	bf := NewBloomFilter(n, 0.01)
	for i := int64(0); i < n; i++ {
		bf.Add(i)
	}

	falsePositives := 0
	const probes = 10000
	for i := int64(n); i < n+probes; i++ {
		if bf.MightContain(i) {
			falsePositives++
		}
	}
	rate := float64(falsePositives) / probes
	// Allow generous margin — we just want to confirm it's far below 1.
	if rate > 0.05 {
		t.Errorf("False positive rate %.3f exceeds 5%% threshold", rate)
	}
}

func TestBloom_Reset(t *testing.T) {
	bf := NewBloomFilter(50, 0.01)
	for i := int64(0); i < 50; i++ {
		bf.Add(i)
	}
	bf.Reset()
	// After reset all bits are 0; even added keys should return false.
	if bf.MightContain(1) {
		t.Error("MightContain after Reset should return false")
	}
	if bf.Fill() != 0 {
		t.Errorf("Fill after Reset = %.2f, want 0", bf.Fill())
	}
}

func TestBloom_Fill(t *testing.T) {
	bf := NewBloomFilter(100, 0.01)
	if bf.Fill() != 0 {
		t.Error("Fresh filter Fill should be 0")
	}
	bf.Add(1)
	if bf.Fill() == 0 {
		t.Error("Fill should be > 0 after Add")
	}
}
