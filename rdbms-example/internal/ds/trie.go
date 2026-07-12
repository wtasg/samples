// trie.go — Trie (prefix tree) used for:
//   1. Schema catalog: O(m) table and column name lookup.
//   2. TEXT column secondary index: enables LIKE 'prefix%' in O(m + k) where
//      k is the number of matching row IDs — far better than a full scan.
//
// Each terminal node stores a slice of row IDs that hold that exact value.
// Prefix search aggregates row IDs from all terminal nodes in the subtree.
package ds

const trieAlphabet = 128 // ASCII

// trieNode is a node in the Trie.
type trieNode struct {
	children [trieAlphabet]*trieNode
	rowIDs   []uint32 // row IDs whose column value equals the path to this node
	isEnd    bool
	// meta stores arbitrary metadata (used by the catalog for column defs, etc.)
	meta any
}

// Trie is a prefix tree over ASCII strings.
type Trie struct {
	root *trieNode
	size int // number of distinct words inserted
}

// NewTrie returns an empty Trie.
func NewTrie() *Trie {
	return &Trie{root: &trieNode{}}
}

// Size returns the number of distinct words in the trie.
func (t *Trie) Size() int { return t.size }

// Insert associates word with rowID.  Multiple inserts of the same word
// accumulate row IDs (useful for non-unique TEXT column values).
func (t *Trie) Insert(word string, rowID uint32) {
	n := t.root
	for _, ch := range []byte(word) {
		if ch >= trieAlphabet {
			continue
		}
		if n.children[ch] == nil {
			n.children[ch] = &trieNode{}
		}
		n = n.children[ch]
	}
	if !n.isEnd {
		n.isEnd = true
		t.size++
	}
	n.rowIDs = append(n.rowIDs, rowID)
}

// SetMeta stores arbitrary metadata at the terminal node for word.
// Used by the catalog to attach ColumnDef/TableSchema to names.
func (t *Trie) SetMeta(word string, meta any) {
	n := t.root
	for _, ch := range []byte(word) {
		if ch >= trieAlphabet {
			return
		}
		if n.children[ch] == nil {
			n.children[ch] = &trieNode{}
		}
		n = n.children[ch]
	}
	n.isEnd = true
	n.meta = meta
}

// GetMeta returns the metadata stored at word's terminal node.
func (t *Trie) GetMeta(word string) (any, bool) {
	n := t.navigate(word)
	if n == nil || !n.isEnd {
		return nil, false
	}
	return n.meta, true
}

// Exact returns all row IDs whose column value equals word exactly.
func (t *Trie) Exact(word string) []uint32 {
	n := t.navigate(word)
	if n == nil || !n.isEnd {
		return nil
	}
	return n.rowIDs
}

// Has reports whether word exists in the trie.
func (t *Trie) Has(word string) bool {
	n := t.navigate(word)
	return n != nil && n.isEnd
}

// HasPrefix reports whether any word in the trie starts with prefix.
func (t *Trie) HasPrefix(prefix string) bool {
	return t.navigate(prefix) != nil
}

// PrefixSearch returns all row IDs whose value starts with prefix.
// This is the key operation enabling fast LIKE 'prefix%' queries.
func (t *Trie) PrefixSearch(prefix string) []uint32 {
	n := t.navigate(prefix)
	if n == nil {
		return nil
	}
	var res []uint32
	t.collect(n, &res)
	return res
}

// Words returns all words stored in the trie (for debugging / SHOW).
func (t *Trie) Words() []string {
	var res []string
	t.wordCollect(t.root, []byte{}, &res)
	return res
}

// Delete removes rowID from word's entry. If the rowID list becomes empty,
// the terminal flag is cleared (the path nodes remain but become dead weight).
func (t *Trie) Delete(word string, rowID uint32) bool {
	n := t.navigate(word)
	if n == nil || !n.isEnd {
		return false
	}
	for i, id := range n.rowIDs {
		if id == rowID {
			n.rowIDs = append(n.rowIDs[:i], n.rowIDs[i+1:]...)
			if len(n.rowIDs) == 0 {
				n.isEnd = false
				t.size--
			}
			return true
		}
	}
	return false
}

// navigate walks to the node for path, returning nil if not found.
func (t *Trie) navigate(path string) *trieNode {
	n := t.root
	for _, ch := range []byte(path) {
		if ch >= trieAlphabet || n.children[ch] == nil {
			return nil
		}
		n = n.children[ch]
	}
	return n
}

// collect gathers all row IDs in the subtree rooted at n.
func (t *Trie) collect(n *trieNode, res *[]uint32) {
	if n == nil {
		return
	}
	if n.isEnd {
		*res = append(*res, n.rowIDs...)
	}
	for _, child := range n.children {
		if child != nil {
			t.collect(child, res)
		}
	}
}

// wordCollect gathers all complete words in the subtree.
func (t *Trie) wordCollect(n *trieNode, prefix []byte, res *[]string) {
	if n == nil {
		return
	}
	if n.isEnd {
		*res = append(*res, string(prefix))
	}
	for i, child := range n.children {
		if child != nil {
			t.wordCollect(child, append(prefix, byte(i)), res)
		}
	}
}
