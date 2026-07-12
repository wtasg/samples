// bloom.go — Bloom Filter used as a fast existence gate before disk reads.
//
// A Bloom Filter is a probabilistic set-membership structure:
//   - False negatives are IMPOSSIBLE: if MightContain returns false, the key
//     is definitely absent, so we can skip the B+ Tree lookup entirely.
//   - False positives are possible: MightContain may return true for a key
//     that is not actually present; we must then confirm via B+ Tree / pager.
//
// k independent hash functions set bits; the filter answers "maybe" or "no".
//
// Optimal parameters (given n expected items and desired false-positive rate p):
//   m = -n * ln(p) / ln(2)^2      (bit array size)
//   k = (m/n) * ln(2)             (number of hash functions)
package ds

import (
	"encoding/binary"
	"hash/fnv"
	"math"
)

// BloomFilter gates primary-key existence checks per table.
type BloomFilter struct {
	bits []uint64 // packed bit array
	m    uint     // total bit count
	k    uint     // hash function count
	n    uint     // expected capacity (stored for FalsePositiveRate)
}

// NewBloomFilter creates a filter optimised for n expected insertions and
// false-positive rate p (0 < p < 1, e.g. 0.01 for 1%).
func NewBloomFilter(n uint, p float64) *BloomFilter {
	m := optimalM(n, p)
	k := optimalK(m, n)
	if k == 0 {
		k = 1
	}
	words := (m + 63) / 64
	return &BloomFilter{bits: make([]uint64, words), m: m, k: k, n: n}
}

func optimalM(n uint, p float64) uint {
	ln2sq := math.Ln2 * math.Ln2
	return uint(math.Ceil(-float64(n) * math.Log(p) / ln2sq))
}

func optimalK(m, n uint) uint {
	return uint(math.Round(float64(m) / float64(n) * math.Ln2))
}

// hashPositions returns k bit positions for the given int64 key using
// double hashing: pos_i = (h1 + i*h2) mod m.
func (bf *BloomFilter) hashPositions(key int64) []uint {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(key))

	h1f := fnv.New64a()
	h1f.Write(b)
	h1 := h1f.Sum64()

	h2f := fnv.New64()
	h2f.Write(b)
	h2 := h2f.Sum64()
	if h2%2 == 0 { // ensure h2 is odd to guarantee full coverage
		h2++
	}

	pos := make([]uint, bf.k)
	for i := uint(0); i < bf.k; i++ {
		pos[i] = uint((h1 + uint64(i)*h2) % uint64(bf.m))
	}
	return pos
}

// Add records that key is a member of the set.
func (bf *BloomFilter) Add(key int64) {
	for _, p := range bf.hashPositions(key) {
		bf.bits[p/64] |= 1 << (p % 64)
	}
}

// MightContain returns false if key is DEFINITELY absent, true if POSSIBLY present.
func (bf *BloomFilter) MightContain(key int64) bool {
	for _, p := range bf.hashPositions(key) {
		if bf.bits[p/64]&(1<<(p%64)) == 0 {
			return false
		}
	}
	return true
}

// Reset clears all bits (used when compacting a table).
func (bf *BloomFilter) Reset() {
	for i := range bf.bits {
		bf.bits[i] = 0
	}
}

// FalsePositiveRate estimates the current false-positive probability based on
// the number of bits set. Formula: (1 - e^(-k*n/m))^k
func (bf *BloomFilter) FalsePositiveRate() float64 {
	// Count set bits.
	setBits := 0
	for _, w := range bf.bits {
		setBits += popcount(w)
	}
	ratio := float64(setBits) / float64(bf.m)
	return math.Pow(ratio, float64(bf.k))
}

// Fill returns the fraction of bits currently set (saturation).
func (bf *BloomFilter) Fill() float64 {
	setBits := 0
	for _, w := range bf.bits {
		setBits += popcount(w)
	}
	return float64(setBits) / float64(bf.m)
}

// Capacity returns the filter's expected capacity n.
func (bf *BloomFilter) Capacity() uint { return bf.n }

// popcount counts set bits in a 64-bit word (Brian Kernighan's method).
func popcount(x uint64) int {
	count := 0
	for x != 0 {
		x &= x - 1
		count++
	}
	return count
}
