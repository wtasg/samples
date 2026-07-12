// Package ds implements the core data structures used by the RDBMS engine.
//
// bptree.go — B+ Tree used as the on-load primary-key index for each table.
//
// A B+ Tree of order m satisfies:
//   - Every internal node (except root) has between ⌈m/2⌉ and m children.
//   - Every leaf holds between ⌈(m-1)/2⌉ and m-1 key/value pairs.
//   - Internal nodes store only routing keys; all actual data lives in leaves.
//   - Leaf nodes are doubly-linked for efficient range scans.
//
// This implementation uses lazy deletion (tombstone flags) so that Delete is
// O(log n) without the complexity of merge/redistribution on underflow.
package ds

// BPOrder is the B+ tree order:
//   - max children per internal node = BPOrder
//   - max keys per leaf              = BPOrder - 1
const BPOrder = 4

// BPEntry is a key→value pair returned by scans.
type BPEntry struct {
	Key int64
	Val uint32 // row ID in the pager
}

// bpNode is a node in the B+ tree.
type bpNode struct {
	keys     []int64   // sorted routing/leaf keys
	vals     []uint32  // row IDs (leaf nodes only)
	children []*bpNode // child pointers (internal nodes only)
	next     *bpNode   // right leaf sibling (leaf nodes only)
	prev     *bpNode   // left  leaf sibling (leaf nodes only)
	isLeaf   bool
	deleted  []bool // tombstone per leaf entry
}

func newLeaf() *bpNode     { return &bpNode{isLeaf: true} }
func newInternal() *bpNode { return &bpNode{isLeaf: false} }

// BPTree maps int64 primary keys to uint32 row IDs.
type BPTree struct {
	root *bpNode
	size int // active (non-deleted) entry count
}

// NewBPTree returns an empty B+ tree.
func NewBPTree() *BPTree { return &BPTree{root: newLeaf()} }

// Size returns the number of active entries.
func (t *BPTree) Size() int { return t.size }

// findLeaf returns the leaf node that should contain key.
func (t *BPTree) findLeaf(key int64) *bpNode {
	n := t.root
	for !n.isLeaf {
		i := 0
		for i < len(n.keys) && key >= n.keys[i] {
			i++
		}
		n = n.children[i]
	}
	return n
}

// Search returns (rowID, true) if key exists and is not deleted.
func (t *BPTree) Search(key int64) (uint32, bool) {
	leaf := t.findLeaf(key)
	for i, k := range leaf.keys {
		if k == key {
			if leaf.deleted[i] {
				return 0, false
			}
			return leaf.vals[i], true
		}
	}
	return 0, false
}

// RangeScan returns all active entries with lo <= key <= hi, in sorted order.
func (t *BPTree) RangeScan(lo, hi int64) []BPEntry {
	var res []BPEntry
	leaf := t.findLeaf(lo)
	for leaf != nil {
		for i, k := range leaf.keys {
			if k < lo {
				continue
			}
			if k > hi {
				return res
			}
			if !leaf.deleted[i] {
				res = append(res, BPEntry{k, leaf.vals[i]})
			}
		}
		leaf = leaf.next
	}
	return res
}

// Insert adds or updates the mapping key → val.
func (t *BPTree) Insert(key int64, val uint32) {
	leaf := t.findLeaf(key)

	// Update existing (possibly tombstoned) entry.
	for i, k := range leaf.keys {
		if k == key {
			if leaf.deleted[i] {
				leaf.deleted[i] = false
				t.size++
			}
			leaf.vals[i] = val
			return
		}
	}

	insertLeafEntry(leaf, key, val)
	t.size++

	if len(leaf.keys) >= BPOrder {
		right, pushKey := splitLeaf(leaf)
		t.bubbleUp(leaf, pushKey, right)
	}
}

// insertLeafEntry inserts key+val into leaf in sorted order.
func insertLeafEntry(leaf *bpNode, key int64, val uint32) {
	i := 0
	for i < len(leaf.keys) && leaf.keys[i] < key {
		i++
	}
	leaf.keys = append(leaf.keys, 0)
	copy(leaf.keys[i+1:], leaf.keys[i:])
	leaf.keys[i] = key

	leaf.vals = append(leaf.vals, 0)
	copy(leaf.vals[i+1:], leaf.vals[i:])
	leaf.vals[i] = val

	leaf.deleted = append(leaf.deleted, false)
	copy(leaf.deleted[i+1:], leaf.deleted[i:])
	leaf.deleted[i] = false
}

// splitLeaf splits a full leaf, returning (new right leaf, push-up key).
func splitLeaf(leaf *bpNode) (*bpNode, int64) {
	mid := len(leaf.keys) / 2
	right := newLeaf()

	right.keys = append(right.keys, leaf.keys[mid:]...)
	right.vals = append(right.vals, leaf.vals[mid:]...)
	right.deleted = append(right.deleted, leaf.deleted[mid:]...)

	leaf.keys = leaf.keys[:mid]
	leaf.vals = leaf.vals[:mid]
	leaf.deleted = leaf.deleted[:mid]

	// Relink leaf doubly-linked list.
	right.next = leaf.next
	if leaf.next != nil {
		leaf.next.prev = right
	}
	right.prev = leaf
	leaf.next = right

	return right, right.keys[0]
}

// bubbleUp propagates a (key, right-child) split upward through the tree.
func (t *BPTree) bubbleUp(left *bpNode, key int64, right *bpNode) {
	if t.root == left {
		r := newInternal()
		r.keys = []int64{key}
		r.children = []*bpNode{left, right}
		t.root = r
		return
	}
	parent := t.findParent(t.root, left)
	insertInternalEntry(parent, key, right)
	if len(parent.keys) >= BPOrder {
		newRight, pushKey := splitInternal(parent)
		t.bubbleUp(parent, pushKey, newRight)
	}
}

// findParent returns the parent of target by DFS from node.
func (t *BPTree) findParent(node, target *bpNode) *bpNode {
	if node.isLeaf {
		return nil
	}
	for _, c := range node.children {
		if c == target {
			return node
		}
	}
	for _, c := range node.children {
		if p := t.findParent(c, target); p != nil {
			return p
		}
	}
	return nil
}

// insertInternalEntry inserts key + right child into an internal node.
func insertInternalEntry(node *bpNode, key int64, right *bpNode) {
	i := 0
	for i < len(node.keys) && node.keys[i] < key {
		i++
	}
	node.keys = append(node.keys, 0)
	copy(node.keys[i+1:], node.keys[i:])
	node.keys[i] = key

	node.children = append(node.children, nil)
	copy(node.children[i+2:], node.children[i+1:])
	node.children[i+1] = right
}

// splitInternal splits a full internal node; the mid key is pushed up.
func splitInternal(node *bpNode) (*bpNode, int64) {
	mid := len(node.keys) / 2
	pushKey := node.keys[mid]

	right := newInternal()
	right.keys = append(right.keys, node.keys[mid+1:]...)
	right.children = append(right.children, node.children[mid+1:]...)

	node.keys = node.keys[:mid]
	node.children = node.children[:mid+1]

	return right, pushKey
}

// Delete tombstones the entry for key. Returns true if found and deleted.
func (t *BPTree) Delete(key int64) bool {
	leaf := t.findLeaf(key)
	for i, k := range leaf.keys {
		if k == key && !leaf.deleted[i] {
			leaf.deleted[i] = true
			t.size--
			return true
		}
	}
	return false
}

// AllEntries returns all active entries in ascending key order by walking
// the leaf linked list from the leftmost leaf.
func (t *BPTree) AllEntries() []BPEntry {
	var res []BPEntry
	n := t.root
	for !n.isLeaf {
		n = n.children[0]
	}
	for n != nil {
		for i, k := range n.keys {
			if !n.deleted[i] {
				res = append(res, BPEntry{k, n.vals[i]})
			}
		}
		n = n.next
	}
	return res
}
