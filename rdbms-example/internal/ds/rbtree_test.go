package ds

import (
	"testing"
)

func TestRBTree_InsertAndSearch(t *testing.T) {
	tree := NewRBTree()
	tree.Insert(10, "ten")
	tree.Insert(5, "five")
	tree.Insert(15, "fifteen")
	tree.Insert(3, "three")
	tree.Insert(7, "seven")

	v, ok := tree.Search(7)
	if !ok || v != "seven" {
		t.Errorf("Search(7) = %v, ok=%v", v, ok)
	}
	_, ok = tree.Search(99)
	if ok {
		t.Error("Search(99) should not be found")
	}
	if tree.Size() != 5 {
		t.Errorf("Size = %d, want 5", tree.Size())
	}
}

func TestRBTree_InOrder(t *testing.T) {
	tree := NewRBTree()
	keys := []int64{5, 3, 7, 1, 4, 6, 8}
	for _, k := range keys {
		tree.Insert(k, k)
	}
	vals := tree.InOrder()
	if len(vals) != len(keys) {
		t.Fatalf("InOrder len = %d, want %d", len(vals), len(keys))
	}
	// Vals should be in ascending key order.
	prev := int64(-1 << 62)
	for _, v := range vals {
		cur := v.(int64)
		if cur <= prev {
			t.Errorf("InOrder not sorted: %d after %d", cur, prev)
		}
		prev = cur
	}
}

func TestRBTree_InOrderDesc(t *testing.T) {
	tree := NewRBTree()
	for i := int64(1); i <= 5; i++ {
		tree.Insert(i, i)
	}
	vals := tree.InOrderDesc()
	for i := 0; i < len(vals)-1; i++ {
		a, b := vals[i].(int64), vals[i+1].(int64)
		if a < b {
			t.Errorf("InOrderDesc not descending at %d: %d < %d", i, a, b)
		}
	}
}

func TestRBTree_Delete(t *testing.T) {
	tree := NewRBTree()
	for i := int64(1); i <= 7; i++ {
		tree.Insert(i, i)
	}
	if !tree.Delete(4) {
		t.Fatal("Delete(4) failed")
	}
	if _, ok := tree.Search(4); ok {
		t.Error("Search(4) found after Delete")
	}
	if tree.Size() != 6 {
		t.Errorf("Size after Delete = %d, want 6", tree.Size())
	}
	// Remaining entries still sorted.
	vals := tree.InOrder()
	for i := 1; i < len(vals); i++ {
		a, b := vals[i-1].(int64), vals[i].(int64)
		if a >= b {
			t.Errorf("InOrder not sorted after Delete at index %d", i)
		}
	}
}

func TestRBTree_DuplicateKeys(t *testing.T) {
	tree := NewRBTree()
	tree.Insert(5, "first")
	tree.Insert(5, "second")
	tree.Insert(5, "third")

	if tree.Size() != 1 {
		t.Errorf("Size = %d, want 1 (one distinct key)", tree.Size())
	}
	vals := tree.InOrder()
	if len(vals) != 3 {
		t.Errorf("InOrder len = %d, want 3 (accumulated values)", len(vals))
	}
}
