// lsmtree.go — Log-Structured Merge Tree used as the primary storage engine.
//
// An LSM Tree is a write-optimised data structure that buffers writes in an
// in-memory component (memtable) and periodically flushes sorted runs to disk
// (Sorted String Tables / SSTables). Reads check the memtable first, then
// each SST level from newest to oldest, using Bloom Filters to skip levels.
//
// LSM Trees are the canonical storage engine for NoSQL databases:
//   - Google Bigtable / LevelDB — the original LSM Tree implementation
//   - Facebook RocksDB — enhanced LevelDB powering MySQL (MyRocks), CockroachDB
//   - Apache Cassandra — uses LSM Trees for its storage engine
//   - MongoDB WiredTiger — supports LSM as an alternative to B-Tree
//   - ScyllaDB, InfluxDB, Apache HBase — all LSM-based
//
// Structure:
//
//	┌──────────────┐
//	│   Memtable   │  ← Skip List (sorted, in-memory, fast writes)
//	└──────┬───────┘
//	       │ flush (when memtable exceeds threshold)
//	       ▼
//	┌──────────────┐
//	│  L0 SSTables │  ← recently flushed, may overlap in key range
//	└──────┬───────┘
//	       │ compaction (merge-sort overlapping SSTables)
//	       ▼
//	┌──────────────┐
//	│  L1 SSTables │  ← non-overlapping, sorted runs
//	└──────────────┘
//
// This implementation stores SST data in-memory ([]SSTEntry slices) rather
// than on disk — suitable for a toy/demonstration database.
package ds

// SSTEntry is a key-value pair stored in an SSTable.
type SSTEntry struct {
	Key     string
	Val     []byte
	Deleted bool // tombstone marker
}

// SSTable is a sorted, immutable run of key-value entries with a Bloom Filter.
type SSTable struct {
	entries []SSTEntry
	bloom   *BloomFilter
	level   int
}

// newSSTable creates an SSTable from sorted entries and builds a Bloom Filter.
func newSSTable(entries []SSTEntry, level int) *SSTable {
	n := len(entries)
	if n == 0 {
		n = 1
	}
	bf := NewBloomFilter(uint(n), 0.01)
	for _, e := range entries {
		bf.Add(e.Key)
	}
	return &SSTable{entries: entries, bloom: bf, level: level}
}

// Search returns (value, deleted, found) for the given key using binary search.
// The Bloom Filter is checked first to avoid unnecessary binary searches.
func (sst *SSTable) Search(key string) ([]byte, bool, bool) {
	if !sst.bloom.MightContain(key) {
		return nil, false, false
	}

	lo, hi := 0, len(sst.entries)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		switch {
		case sst.entries[mid].Key < key:
			lo = mid + 1
		case sst.entries[mid].Key > key:
			hi = mid - 1
		default:
			return sst.entries[mid].Val, sst.entries[mid].Deleted, true
		}
	}
	return nil, false, false
}

// Range returns all entries with lo <= key <= hi from this SSTable.
func (sst *SSTable) Range(lo, hi string) []SSTEntry {
	// Find start position with binary search.
	start := 0
	low, high := 0, len(sst.entries)-1
	for low <= high {
		mid := (low + high) / 2
		if sst.entries[mid].Key < lo {
			low = mid + 1
		} else {
			start = mid
			high = mid - 1
		}
	}

	var res []SSTEntry
	for i := start; i < len(sst.entries) && sst.entries[i].Key <= hi; i++ {
		res = append(res, sst.entries[i])
	}
	return res
}

// Entries returns all entries in the SSTable.
func (sst *SSTable) Entries() []SSTEntry { return sst.entries }

// Size returns the number of entries in the SSTable.
func (sst *SSTable) Size() int { return len(sst.entries) }

// LSMTree is a Log-Structured Merge Tree with a Skip List memtable.
type LSMTree struct {
	memtable       *SkipList
	l0             []*SSTable // Level 0: recently flushed (may overlap)
	l1             []*SSTable // Level 1: compacted (non-overlapping)
	flushThreshold int        // max entries before memtable flush
}

// NewLSMTree returns a new LSM Tree with the given memtable flush threshold.
func NewLSMTree(flushThreshold int) *LSMTree {
	if flushThreshold <= 0 {
		flushThreshold = 100
	}
	return &LSMTree{
		memtable:       NewSkipList(),
		flushThreshold: flushThreshold,
	}
}

// Put writes a key-value pair to the memtable. If the memtable exceeds the
// flush threshold, it is flushed to an L0 SSTable.
func (lsm *LSMTree) Put(key string, val []byte) {
	lsm.memtable.Insert(key, val)
	if lsm.memtable.Size() >= lsm.flushThreshold {
		lsm.flush()
	}
}

// Get retrieves a value by key. Checks memtable first, then L0 (newest first),
// then L1 (newest first). Uses Bloom Filters on SSTables to skip unnecessary
// searches.
func (lsm *LSMTree) Get(key string) ([]byte, bool) {
	// ① Memtable: O(log n)
	val, deleted, found := lsm.memtable.SearchWithTombstone(key)
	if found {
		if deleted {
			return nil, false // key was deleted
		}
		return val, true
	}

	// ② L0 SSTables (newest first): Bloom Filter → binary search
	for i := len(lsm.l0) - 1; i >= 0; i-- {
		val, deleted, found = lsm.l0[i].Search(key)
		if found {
			if deleted {
				return nil, false
			}
			return val, true
		}
	}

	// ③ L1 SSTables (newest first)
	for i := len(lsm.l1) - 1; i >= 0; i-- {
		val, deleted, found = lsm.l1[i].Search(key)
		if found {
			if deleted {
				return nil, false
			}
			return val, true
		}
	}

	return nil, false
}

// Delete inserts a tombstone for the key.
func (lsm *LSMTree) Delete(key string) {
	lsm.memtable.InsertTombstone(key)
	if lsm.memtable.Size() >= lsm.flushThreshold {
		lsm.flush()
	}
}

// Scan returns all active entries with lo <= key <= hi across all levels.
// Newer entries take precedence over older ones.
func (lsm *LSMTree) Scan(lo, hi string) []SSTEntry {
	seen := make(map[string]SSTEntry)

	// Scan L1 first (oldest), then L0, then memtable (newest wins).
	for _, sst := range lsm.l1 {
		for _, e := range sst.Range(lo, hi) {
			seen[e.Key] = e
		}
	}
	for _, sst := range lsm.l0 {
		for _, e := range sst.Range(lo, hi) {
			seen[e.Key] = e
		}
	}
	for _, e := range lsm.memtable.Range(lo, hi) {
		seen[e.Key] = SSTEntry{Key: e.Key, Val: e.Val}
	}
	// Also include tombstoned memtable entries.
	for _, e := range lsm.memtable.AllEntriesFull() {
		if e.Key >= lo && e.Key <= hi {
			seen[e.Key] = SSTEntry{Key: e.Key, Val: e.Val, Deleted: e.Deleted}
		}
	}

	// Collect non-deleted entries in sorted order.
	var res []SSTEntry
	for _, e := range seen {
		if !e.Deleted {
			res = append(res, e)
		}
	}

	// Sort by key.
	sortSSTEntries(res)
	return res
}

// AllEntries returns all active entries across all levels, in sorted key order.
func (lsm *LSMTree) AllEntries() []SSTEntry {
	seen := make(map[string]SSTEntry)

	for _, sst := range lsm.l1 {
		for _, e := range sst.entries {
			seen[e.Key] = e
		}
	}
	for _, sst := range lsm.l0 {
		for _, e := range sst.entries {
			seen[e.Key] = e
		}
	}
	for _, e := range lsm.memtable.AllEntriesFull() {
		seen[e.Key] = SSTEntry{Key: e.Key, Val: e.Val, Deleted: e.Deleted}
	}

	var res []SSTEntry
	for _, e := range seen {
		if !e.Deleted {
			res = append(res, e)
		}
	}

	sortSSTEntries(res)
	return res
}

// flush writes the memtable contents to a new L0 SSTable and clears the
// memtable. Triggers compaction if L0 exceeds 4 SSTables.
func (lsm *LSMTree) flush() {
	entries := lsm.memtable.AllEntriesFull()
	if len(entries) == 0 {
		return
	}

	sstEntries := make([]SSTEntry, len(entries))
	for i, e := range entries {
		sstEntries[i] = SSTEntry{Key: e.Key, Val: e.Val, Deleted: e.Deleted}
	}

	sst := newSSTable(sstEntries, 0)
	lsm.l0 = append(lsm.l0, sst)
	lsm.memtable.Clear()

	// Compact L0→L1 when L0 has too many SSTables.
	if len(lsm.l0) >= 4 {
		lsm.compact()
	}
}

// Flush exposes the internal flush for testing.
func (lsm *LSMTree) Flush() { lsm.flush() }

// compact merges all L0 SSTables into a single L1 SSTable.
func (lsm *LSMTree) compact() {
	// Merge all L0 + L1 entries; newer entries win.
	seen := make(map[string]SSTEntry)

	for _, sst := range lsm.l1 {
		for _, e := range sst.entries {
			seen[e.Key] = e
		}
	}
	for _, sst := range lsm.l0 {
		for _, e := range sst.entries {
			seen[e.Key] = e
		}
	}

	// Build sorted entries for new L1 SSTable, dropping tombstones.
	var merged []SSTEntry
	for _, e := range seen {
		if !e.Deleted {
			merged = append(merged, e)
		}
	}
	sortSSTEntries(merged)

	if len(merged) > 0 {
		lsm.l1 = []*SSTable{newSSTable(merged, 1)}
	} else {
		lsm.l1 = nil
	}
	lsm.l0 = nil
}

// MemtableSize returns the number of entries in the memtable.
func (lsm *LSMTree) MemtableSize() int { return lsm.memtable.Size() }

// L0Count returns the number of L0 SSTables.
func (lsm *LSMTree) L0Count() int { return len(lsm.l0) }

// L1Count returns the number of L1 SSTables.
func (lsm *LSMTree) L1Count() int { return len(lsm.l1) }

// sortSSTEntries sorts entries by key using insertion sort
// (suitable for the small data volumes in a toy DB).
func sortSSTEntries(entries []SSTEntry) {
	for i := 1; i < len(entries); i++ {
		key := entries[i]
		j := i - 1
		for j >= 0 && entries[j].Key > key.Key {
			entries[j+1] = entries[j]
			j--
		}
		entries[j+1] = key
	}
}
