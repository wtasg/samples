package ds

import (
	"sort"
	"testing"
)

func TestTrie_InsertAndSearch(t *testing.T) {
	tr := NewTrie()
	tr.Insert("apple", 1)
	tr.Insert("app", 2)
	tr.Insert("banana", 3)

	ids := tr.Exact("apple")
	if len(ids) != 1 || ids[0] != 1 {
		t.Errorf("Exact(apple) = %v, want [1]", ids)
	}
	ids = tr.Exact("app")
	if len(ids) != 1 || ids[0] != 2 {
		t.Errorf("Exact(app) = %v, want [2]", ids)
	}
	if tr.Exact("ap") != nil {
		t.Error("Exact(ap) should be nil (not a terminal)")
	}
}

func TestTrie_PrefixSearch(t *testing.T) {
	tr := NewTrie()
	words := []struct {
		w  string
		id uint32
	}{
		{"alice", 1}, {"alex", 2}, {"alfred", 3}, {"bob", 4}, {"alan", 5},
	}
	for _, w := range words {
		tr.Insert(w.w, w.id)
	}

	ids := tr.PrefixSearch("al")
	if len(ids) != 4 { // alice, alex, alfred, alan
		t.Errorf("PrefixSearch(al) returned %d ids, want 4; got %v", len(ids), ids)
	}

	ids = tr.PrefixSearch("bob")
	if len(ids) != 1 || ids[0] != 4 {
		t.Errorf("PrefixSearch(bob) = %v, want [4]", ids)
	}

	ids = tr.PrefixSearch("z")
	if len(ids) != 0 {
		t.Errorf("PrefixSearch(z) should be empty, got %v", ids)
	}

	// Full prefix = all entries.
	ids = tr.PrefixSearch("")
	if len(ids) != 5 {
		t.Errorf("PrefixSearch('') = %d ids, want 5", len(ids))
	}
}

func TestTrie_HasPrefix(t *testing.T) {
	tr := NewTrie()
	tr.Insert("hello", 1)
	tr.Insert("world", 2)

	if !tr.HasPrefix("hel") {
		t.Error("HasPrefix(hel) should be true")
	}
	if tr.HasPrefix("xyz") {
		t.Error("HasPrefix(xyz) should be false")
	}
}

func TestTrie_Delete(t *testing.T) {
	tr := NewTrie()
	tr.Insert("go", 10)
	tr.Insert("go", 20) // same word, two rows

	if !tr.Delete("go", 10) {
		t.Fatal("Delete(go,10) failed")
	}
	ids := tr.Exact("go")
	if len(ids) != 1 || ids[0] != 20 {
		t.Errorf("After Delete(10), Exact(go) = %v, want [20]", ids)
	}

	tr.Delete("go", 20)
	if tr.Has("go") {
		t.Error("go should not exist after deleting all rowIDs")
	}
	if tr.Size() != 0 {
		t.Errorf("Size = %d, want 0", tr.Size())
	}
}

func TestTrie_Words(t *testing.T) {
	tr := NewTrie()
	words := []string{"cat", "car", "card", "care", "dog"}
	for i, w := range words {
		tr.Insert(w, uint32(i))
	}
	got := tr.Words()
	sort.Strings(got)
	sort.Strings(words)
	if len(got) != len(words) {
		t.Fatalf("Words() len = %d, want %d", len(got), len(words))
	}
	for i := range words {
		if got[i] != words[i] {
			t.Errorf("Words()[%d] = %q, want %q", i, got[i], words[i])
		}
	}
}

func TestTrie_SetGetMeta(t *testing.T) {
	tr := NewTrie()
	tr.SetMeta("users", "table-schema")
	v, ok := tr.GetMeta("users")
	if !ok || v != "table-schema" {
		t.Errorf("GetMeta(users) = %v, ok=%v", v, ok)
	}
	_, ok = tr.GetMeta("orders")
	if ok {
		t.Error("GetMeta(orders) should not be found")
	}
}
