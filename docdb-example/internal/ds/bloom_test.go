package ds

import (
	"fmt"
	"testing"
)

func TestBloomFilter_AddAndContains(t *testing.T) {
	bf := NewBloomFilter(100, 0.01)

	keys := []string{"apple", "banana", "cherry", "date", "elderberry"}
	for _, k := range keys {
		bf.Add(k)
	}

	for _, k := range keys {
		if !bf.MightContain(k) {
			t.Errorf("MightContain(%q) = false, want true (zero false negatives)", k)
		}
	}
}

func TestBloomFilter_NoFalseNegatives(t *testing.T) {
	bf := NewBloomFilter(1000, 0.01)

	for i := 0; i < 500; i++ {
		bf.Add(fmt.Sprintf("key-%d", i))
	}

	for i := 0; i < 500; i++ {
		if !bf.MightContain(fmt.Sprintf("key-%d", i)) {
			t.Errorf("false negative for key-%d", i)
		}
	}
}

func TestBloomFilter_FalsePositiveRate(t *testing.T) {
	bf := NewBloomFilter(1000, 0.01)

	for i := 0; i < 1000; i++ {
		bf.Add(fmt.Sprintf("present-%d", i))
	}

	fps := 0
	const trials = 10000
	for i := 0; i < trials; i++ {
		if bf.MightContain(fmt.Sprintf("absent-%d", i)) {
			fps++
		}
	}

	rate := float64(fps) / trials
	// Allow up to 5% false positive rate (generous for a test).
	if rate > 0.05 {
		t.Errorf("false positive rate = %.4f, want < 0.05", rate)
	}
}

func TestBloomFilter_IntKeys(t *testing.T) {
	bf := NewBloomFilter(100, 0.01)

	for i := int64(1); i <= 50; i++ {
		bf.AddInt(i)
	}

	for i := int64(1); i <= 50; i++ {
		if !bf.MightContainInt(i) {
			t.Errorf("MightContainInt(%d) = false, want true", i)
		}
	}
}

func TestBloomFilter_Reset(t *testing.T) {
	bf := NewBloomFilter(100, 0.01)
	bf.Add("test")

	if !bf.MightContain("test") {
		t.Fatal("expected MightContain to return true before reset")
	}

	bf.Reset()

	if bf.MightContain("test") {
		t.Error("expected MightContain to return false after reset")
	}
}

func TestBloomFilter_FillAndCapacity(t *testing.T) {
	bf := NewBloomFilter(100, 0.01)

	if bf.Fill() != 0 {
		t.Errorf("Fill() = %f, want 0", bf.Fill())
	}
	if bf.Capacity() != 100 {
		t.Errorf("Capacity() = %d, want 100", bf.Capacity())
	}

	for i := 0; i < 50; i++ {
		bf.Add(fmt.Sprintf("key-%d", i))
	}

	if bf.Fill() <= 0 {
		t.Error("expected Fill() > 0 after insertions")
	}
}
