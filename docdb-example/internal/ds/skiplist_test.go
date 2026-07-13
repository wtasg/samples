package ds

import (
	"testing"
)

func TestSkipList_InsertAndSearch(t *testing.T) {
	sl := NewSkipList()
	sl.Insert("banana", []byte("yellow"))
	sl.Insert("apple", []byte("red"))
	sl.Insert("cherry", []byte("dark red"))

	tests := []struct {
		key  string
		want string
	}{
		{"apple", "red"},
		{"banana", "yellow"},
		{"cherry", "dark red"},
	}

	for _, tt := range tests {
		v, ok := sl.Search(tt.key)
		if !ok {
			t.Errorf("Search(%q) not found", tt.key)
			continue
		}
		if string(v) != tt.want {
			t.Errorf("Search(%q) = %q, want %q", tt.key, v, tt.want)
		}
	}

	if sl.Size() != 3 {
		t.Errorf("Size() = %d, want 3", sl.Size())
	}
}

func TestSkipList_UpdateExisting(t *testing.T) {
	sl := NewSkipList()
	sl.Insert("key", []byte("v1"))
	sl.Insert("key", []byte("v2"))

	v, ok := sl.Search("key")
	if !ok || string(v) != "v2" {
		t.Errorf("expected updated value 'v2', got %q (ok=%v)", v, ok)
	}
	if sl.Size() != 1 {
		t.Errorf("Size() = %d, want 1", sl.Size())
	}
}

func TestSkipList_Delete(t *testing.T) {
	sl := NewSkipList()
	sl.Insert("a", []byte("1"))
	sl.Insert("b", []byte("2"))
	sl.Insert("c", []byte("3"))

	if !sl.Delete("b") {
		t.Fatal("Delete('b') returned false")
	}
	if _, ok := sl.Search("b"); ok {
		t.Error("Search('b') found after Delete")
	}
	if sl.Size() != 2 {
		t.Errorf("Size() = %d, want 2", sl.Size())
	}

	// Delete non-existent key.
	if sl.Delete("z") {
		t.Error("Delete('z') returned true for non-existent key")
	}
}

func TestSkipList_Range(t *testing.T) {
	sl := NewSkipList()
	for _, k := range []string{"a", "b", "c", "d", "e", "f"} {
		sl.Insert(k, []byte(k))
	}

	got := sl.Range("b", "e")
	if len(got) != 4 {
		t.Fatalf("Range('b','e') returned %d entries, want 4", len(got))
	}
	expected := []string{"b", "c", "d", "e"}
	for i, e := range got {
		if e.Key != expected[i] {
			t.Errorf("entry[%d].Key = %q, want %q", i, e.Key, expected[i])
		}
	}
}

func TestSkipList_InOrder(t *testing.T) {
	sl := NewSkipList()
	// Insert in reverse order to verify sorting.
	for _, k := range []string{"z", "m", "a", "g", "t"} {
		sl.Insert(k, []byte(k))
	}

	entries := sl.InOrder()
	if len(entries) != 5 {
		t.Fatalf("InOrder len = %d, want 5", len(entries))
	}
	for i := 1; i < len(entries); i++ {
		if entries[i].Key <= entries[i-1].Key {
			t.Errorf("InOrder not sorted at index %d: %q <= %q", i, entries[i].Key, entries[i-1].Key)
		}
	}
}

func TestSkipList_ManyInserts(t *testing.T) {
	sl := NewSkipList()
	const N = 500
	for i := 0; i < N; i++ {
		key := string(rune('A' + i%26)) + string(rune('0'+i/26%10)) + string(rune('0'+i%10))
		sl.Insert(key, []byte(key))
	}

	entries := sl.InOrder()
	for i := 1; i < len(entries); i++ {
		if entries[i].Key <= entries[i-1].Key {
			t.Errorf("InOrder not sorted at index %d", i)
			break
		}
	}
}

func TestSkipList_Tombstone(t *testing.T) {
	sl := NewSkipList()
	sl.Insert("key1", []byte("val1"))
	sl.InsertTombstone("key2")

	// key2 should not appear in Search.
	if _, ok := sl.Search("key2"); ok {
		t.Error("tombstoned key should not appear in Search")
	}

	// But it should appear in SearchWithTombstone.
	_, deleted, found := sl.SearchWithTombstone("key2")
	if !found {
		t.Error("tombstoned key should be found in SearchWithTombstone")
	}
	if !deleted {
		t.Error("tombstoned key should report deleted=true")
	}
}

func TestSkipList_Clear(t *testing.T) {
	sl := NewSkipList()
	sl.Insert("a", []byte("1"))
	sl.Insert("b", []byte("2"))

	sl.Clear()

	if sl.Size() != 0 {
		t.Errorf("Size() = %d after Clear, want 0", sl.Size())
	}
	if _, ok := sl.Search("a"); ok {
		t.Error("Search found key after Clear")
	}
}
