package ds

import (
	"testing"
)

func TestBPTree_InsertAndSearch(t *testing.T) {
	tree := NewBPTree()
	keys := []int64{5, 3, 7, 1, 4, 6, 8, 2}
	for i, k := range keys {
		tree.Insert(k, uint32(i+1))
	}

	for i, k := range keys {
		v, ok := tree.Search(k)
		if !ok {
			t.Errorf("Search(%d) not found", k)
		}
		if v != uint32(i+1) {
			t.Errorf("Search(%d) = %d, want %d", k, v, i+1)
		}
	}

	if tree.Size() != len(keys) {
		t.Errorf("Size() = %d, want %d", tree.Size(), len(keys))
	}
}

func TestBPTree_RangeScan(t *testing.T) {
	tree := NewBPTree()
	for i := int64(1); i <= 10; i++ {
		tree.Insert(i, uint32(i))
	}

	got := tree.RangeScan(3, 7)
	if len(got) != 5 {
		t.Fatalf("RangeScan(3,7) returned %d entries, want 5", len(got))
	}
	for i, e := range got {
		if e.Key != int64(i+3) {
			t.Errorf("entry[%d].Key = %d, want %d", i, e.Key, i+3)
		}
	}
}

func TestBPTree_Delete(t *testing.T) {
	tree := NewBPTree()
	for i := int64(1); i <= 5; i++ {
		tree.Insert(i, uint32(i))
	}

	if !tree.Delete(3) {
		t.Fatal("Delete(3) returned false")
	}
	if _, ok := tree.Search(3); ok {
		t.Error("Search(3) found after Delete")
	}
	if tree.Size() != 4 {
		t.Errorf("Size after delete = %d, want 4", tree.Size())
	}
}

func TestBPTree_AllEntries(t *testing.T) {
	tree := NewBPTree()
	for i := int64(5); i >= 1; i-- {
		tree.Insert(i, uint32(i))
	}
	tree.Delete(3)

	entries := tree.AllEntries()
	if len(entries) != 4 {
		t.Fatalf("AllEntries len = %d, want 4", len(entries))
	}
	// Must be sorted.
	for i := 1; i < len(entries); i++ {
		if entries[i].Key <= entries[i-1].Key {
			t.Errorf("AllEntries not sorted at index %d", i)
		}
	}
}

func TestBPTree_UpdateExisting(t *testing.T) {
	tree := NewBPTree()
	tree.Insert(1, 100)
	tree.Insert(1, 200) // update
	v, ok := tree.Search(1)
	if !ok || v != 200 {
		t.Errorf("expected updated value 200, got %d (ok=%v)", v, ok)
	}
	if tree.Size() != 1 {
		t.Errorf("Size after update = %d, want 1", tree.Size())
	}
}

func TestBPTree_ManySplits(t *testing.T) {
	tree := NewBPTree()
	const N = 200
	for i := int64(0); i < N; i++ {
		tree.Insert(i, uint32(i))
	}
	if tree.Size() != N {
		t.Fatalf("Size = %d after %d inserts", tree.Size(), N)
	}
	// Verify all entries are present and sorted.
	entries := tree.AllEntries()
	if len(entries) != N {
		t.Fatalf("AllEntries len = %d, want %d", len(entries), N)
	}
	for i, e := range entries {
		if e.Key != int64(i) {
			t.Errorf("entry[%d].Key = %d, want %d", i, e.Key, i)
		}
	}
}
