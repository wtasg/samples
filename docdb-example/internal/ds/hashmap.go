// hashmap.go — Open-addressing Hash Map with Robin Hood hashing for O(1)
// document ID lookups.
//
// A hash map provides the fastest possible key→value lookup for the primary
// access pattern in a document database: GET by _id.
//
// This implementation uses Robin Hood hashing, which reduces the variance of
// probe sequences by "stealing from the rich" (displacing entries with short
// probe distances in favour of entries with long probe distances).
//
// Robin Hood hashing is used in:
//   - Rust's standard HashMap (until 1.36)
//   - Swift's Dictionary
//   - Many game engines and embedded databases
//
// Properties:
//   - Open addressing with linear probing
//   - Robin Hood displacement for fairness
//   - Automatic resize at 75% load factor
//   - Tombstone-based lazy deletion
package ds

import "hash/fnv"

const (
	hmInitialCap = 16
	hmLoadFactor = 0.75
)

// hmEntry is a slot in the hash map.
type hmEntry struct {
	key     string
	val     []byte
	present bool
	deleted bool
}

// HashMap is an open-addressing hash map with Robin Hood hashing.
type HashMap struct {
	buckets []hmEntry
	cap     int
	size    int // active (non-deleted) entries
}

// NewHashMap returns an empty hash map with initial capacity.
func NewHashMap() *HashMap {
	return &HashMap{
		buckets: make([]hmEntry, hmInitialCap),
		cap:     hmInitialCap,
	}
}

// Size returns the number of active entries.
func (m *HashMap) Size() int { return m.size }

// hash returns the bucket index for a key.
func (m *HashMap) hash(key string) int {
	h := fnv.New64a()
	h.Write([]byte(key))
	return int(h.Sum64() % uint64(m.cap))
}

// probeDistance returns the distance of a bucket from its home position.
func (m *HashMap) probeDistance(bucketIdx int, entry *hmEntry) int {
	home := m.hash(entry.key)
	if bucketIdx >= home {
		return bucketIdx - home
	}
	return m.cap - home + bucketIdx
}

// Put adds or updates the mapping key → val.
func (m *HashMap) Put(key string, val []byte) {
	if float64(m.size+1)/float64(m.cap) > hmLoadFactor {
		m.resize(m.cap * 2)
	}

	entry := hmEntry{key: key, val: val, present: true}
	idx := m.hash(key)
	dist := 0

	for {
		bucket := &m.buckets[idx]

		if !bucket.present || bucket.deleted {
			// Empty or deleted slot — place here.
			m.buckets[idx] = entry
			m.size++
			return
		}

		if bucket.key == key {
			// Update existing.
			bucket.val = val
			return
		}

		// Robin Hood: if the existing entry has a shorter probe distance,
		// steal its spot.
		existingDist := m.probeDistance(idx, bucket)
		if dist > existingDist {
			// Swap: place our entry here, continue inserting the displaced one.
			m.buckets[idx], entry = entry, m.buckets[idx]
			dist = existingDist
		}

		dist++
		idx = (idx + 1) % m.cap
	}
}

// Get returns (value, true) if key exists, or (nil, false).
func (m *HashMap) Get(key string) ([]byte, bool) {
	idx := m.hash(key)
	dist := 0

	for {
		bucket := &m.buckets[idx]

		if !bucket.present {
			return nil, false
		}

		if !bucket.deleted && bucket.key == key {
			return bucket.val, true
		}

		// Robin Hood: if distance exceeds what this entry would have, key is absent.
		if bucket.present && !bucket.deleted {
			if dist > m.probeDistance(idx, bucket) {
				return nil, false
			}
		}

		dist++
		idx = (idx + 1) % m.cap

		if dist >= m.cap {
			return nil, false
		}
	}
}

// Delete removes the entry for key. Returns true if found and deleted.
func (m *HashMap) Delete(key string) bool {
	idx := m.hash(key)
	dist := 0

	for {
		bucket := &m.buckets[idx]

		if !bucket.present {
			return false
		}

		if !bucket.deleted && bucket.key == key {
			bucket.deleted = true
			m.size--
			return true
		}

		if bucket.present && !bucket.deleted {
			if dist > m.probeDistance(idx, bucket) {
				return false
			}
		}

		dist++
		idx = (idx + 1) % m.cap

		if dist >= m.cap {
			return false
		}
	}
}

// Has reports whether key exists in the map.
func (m *HashMap) Has(key string) bool {
	_, ok := m.Get(key)
	return ok
}

// Keys returns all active keys (unordered).
func (m *HashMap) Keys() []string {
	keys := make([]string, 0, m.size)
	for i := range m.buckets {
		if m.buckets[i].present && !m.buckets[i].deleted {
			keys = append(keys, m.buckets[i].key)
		}
	}
	return keys
}

// Entries returns all active key-value pairs.
type HMEntry struct {
	Key string
	Val []byte
}

// Entries returns all active entries (unordered).
func (m *HashMap) Entries() []HMEntry {
	entries := make([]HMEntry, 0, m.size)
	for i := range m.buckets {
		if m.buckets[i].present && !m.buckets[i].deleted {
			entries = append(entries, HMEntry{Key: m.buckets[i].key, Val: m.buckets[i].val})
		}
	}
	return entries
}

// resize grows the hash map and reinserts all active entries.
func (m *HashMap) resize(newCap int) {
	old := m.buckets
	m.buckets = make([]hmEntry, newCap)
	m.cap = newCap
	m.size = 0

	for i := range old {
		if old[i].present && !old[i].deleted {
			m.Put(old[i].key, old[i].val)
		}
	}
}

// LoadFactor returns the current load factor.
func (m *HashMap) LoadFactor() float64 {
	return float64(m.size) / float64(m.cap)
}
