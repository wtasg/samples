// skiplist.go — Probabilistic Skip List used as the ordered memtable in the
// LSM Tree.
//
// A Skip List is a layered linked list that provides O(log n) expected time
// for insert, search, and delete operations through probabilistic balancing.
// Each node has a randomly determined height; higher levels act as "express
// lanes" that let searches skip over large sections of the list.
//
// Skip Lists are widely used as memtables in LSM-based storage engines:
//   - LevelDB / RocksDB use skip lists for their memtable layer.
//   - Redis uses skip lists for sorted sets (ZSET).
//   - Apache HBase uses ConcurrentSkipListMap for memstore.
//
// Compared to balanced BSTs (Red-Black, AVL), skip lists are simpler to
// implement, have similar expected complexity, and are more amenable to
// concurrent access (lock-free algorithms exist).
//
// Parameters:
//   - MaxLevel = 16  (supports ~65,536 entries efficiently)
//   - P        = 0.5 (probability of promoting to the next level)
package ds

import (
	"math/rand"
)

const (
	slMaxLevel = 16   // maximum height of any node
	slP        = 0.5  // probability of level promotion
)

// SLEntry is a key-value pair returned by scans.
type SLEntry struct {
	Key string
	Val []byte
}

// slNode is a node in the skip list.
type slNode struct {
	key     string
	val     []byte
	forward []*slNode // forward[i] is the next node at level i
	deleted bool      // tombstone marker for LSM deletes
}

// SkipList is a probabilistic ordered map from string keys to byte slices.
type SkipList struct {
	head  *slNode
	level int // current maximum level in use (0-indexed)
	size  int // number of active (non-deleted) entries
	rng   *rand.Rand
}

// NewSkipList returns an empty skip list.
func NewSkipList() *SkipList {
	head := &slNode{forward: make([]*slNode, slMaxLevel)}
	return &SkipList{
		head:  head,
		level: 0,
		rng:   rand.New(rand.NewSource(42)),
	}
}

// Size returns the number of active entries.
func (sl *SkipList) Size() int { return sl.size }

// randomLevel generates a random level for a new node using geometric
// distribution with parameter slP.
func (sl *SkipList) randomLevel() int {
	lvl := 0
	for lvl < slMaxLevel-1 && sl.rng.Float64() < slP {
		lvl++
	}
	return lvl
}

// Insert adds or updates the mapping key → val.
// If the key already exists, its value is updated and any tombstone is cleared.
func (sl *SkipList) Insert(key string, val []byte) {
	update := make([]*slNode, slMaxLevel)
	x := sl.head

	// Traverse from the highest level down.
	for i := sl.level; i >= 0; i-- {
		for x.forward[i] != nil && x.forward[i].key < key {
			x = x.forward[i]
		}
		update[i] = x
	}

	x = x.forward[0]

	if x != nil && x.key == key {
		// Update existing entry.
		if x.deleted {
			x.deleted = false
			sl.size++
		}
		x.val = val
		return
	}

	// Insert new node.
	newLevel := sl.randomLevel()
	if newLevel > sl.level {
		for i := sl.level + 1; i <= newLevel; i++ {
			update[i] = sl.head
		}
		sl.level = newLevel
	}

	newNode := &slNode{
		key:     key,
		val:     val,
		forward: make([]*slNode, newLevel+1),
	}

	for i := 0; i <= newLevel; i++ {
		newNode.forward[i] = update[i].forward[i]
		update[i].forward[i] = newNode
	}

	sl.size++
}

// Search returns (value, true) if key exists and is not deleted.
func (sl *SkipList) Search(key string) ([]byte, bool) {
	x := sl.head
	for i := sl.level; i >= 0; i-- {
		for x.forward[i] != nil && x.forward[i].key < key {
			x = x.forward[i]
		}
	}
	x = x.forward[0]

	if x != nil && x.key == key && !x.deleted {
		return x.val, true
	}
	return nil, false
}

// SearchWithTombstone returns (value, deleted, found).
// It reports tombstoned entries so the LSM Tree can propagate deletions.
func (sl *SkipList) SearchWithTombstone(key string) ([]byte, bool, bool) {
	x := sl.head
	for i := sl.level; i >= 0; i-- {
		for x.forward[i] != nil && x.forward[i].key < key {
			x = x.forward[i]
		}
	}
	x = x.forward[0]

	if x != nil && x.key == key {
		return x.val, x.deleted, true
	}
	return nil, false, false
}

// Delete tombstones the entry for key. Returns true if found and deleted.
func (sl *SkipList) Delete(key string) bool {
	x := sl.head
	for i := sl.level; i >= 0; i-- {
		for x.forward[i] != nil && x.forward[i].key < key {
			x = x.forward[i]
		}
	}
	x = x.forward[0]

	if x != nil && x.key == key && !x.deleted {
		x.deleted = true
		sl.size--
		return true
	}
	return false
}

// InsertTombstone inserts a delete marker for a key, used by LSM Tree to
// propagate deletes across SST levels.
func (sl *SkipList) InsertTombstone(key string) {
	sl.Insert(key, nil)
	// Mark as deleted without decrementing size (it was just inserted).
	x := sl.head
	for i := sl.level; i >= 0; i-- {
		for x.forward[i] != nil && x.forward[i].key < key {
			x = x.forward[i]
		}
	}
	x = x.forward[0]
	if x != nil && x.key == key {
		if !x.deleted {
			sl.size--
		}
		x.deleted = true
	}
}

// Range returns all active entries with lo <= key <= hi, in sorted order.
func (sl *SkipList) Range(lo, hi string) []SLEntry {
	var res []SLEntry
	x := sl.head
	for i := sl.level; i >= 0; i-- {
		for x.forward[i] != nil && x.forward[i].key < lo {
			x = x.forward[i]
		}
	}
	x = x.forward[0]

	for x != nil && x.key <= hi {
		if !x.deleted {
			res = append(res, SLEntry{Key: x.key, Val: x.val})
		}
		x = x.forward[0]
	}
	return res
}

// InOrder returns all active entries in ascending key order.
func (sl *SkipList) InOrder() []SLEntry {
	var res []SLEntry
	x := sl.head.forward[0]
	for x != nil {
		if !x.deleted {
			res = append(res, SLEntry{Key: x.key, Val: x.val})
		}
		x = x.forward[0]
	}
	return res
}

// AllEntries returns all entries including tombstones (for SST flush).
func (sl *SkipList) AllEntries() []SLEntry {
	var res []SLEntry
	x := sl.head.forward[0]
	for x != nil {
		e := SLEntry{Key: x.key, Val: x.val}
		if x.deleted {
			e.Val = nil // tombstone marker
		}
		res = append(res, e)
		x = x.forward[0]
	}
	return res
}

// AllEntriesWithTombstones returns all entries with tombstone flag exposed.
type SLEntryFull struct {
	Key     string
	Val     []byte
	Deleted bool
}

// AllEntriesFull returns all entries including tombstone flags.
func (sl *SkipList) AllEntriesFull() []SLEntryFull {
	var res []SLEntryFull
	x := sl.head.forward[0]
	for x != nil {
		res = append(res, SLEntryFull{Key: x.key, Val: x.val, Deleted: x.deleted})
		x = x.forward[0]
	}
	return res
}

// Clear resets the skip list to empty.
func (sl *SkipList) Clear() {
	sl.head = &slNode{forward: make([]*slNode, slMaxLevel)}
	sl.level = 0
	sl.size = 0
}
