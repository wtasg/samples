package ds

import (
	"fmt"
	"testing"
)

func TestLSMTree_PutAndGet(t *testing.T) {
	lsm := NewLSMTree(10)
	lsm.Put("apple", []byte("red"))
	lsm.Put("banana", []byte("yellow"))
	lsm.Put("cherry", []byte("dark red"))

	tests := []struct {
		key  string
		want string
	}{
		{"apple", "red"},
		{"banana", "yellow"},
		{"cherry", "dark red"},
	}
	for _, tt := range tests {
		v, ok := lsm.Get(tt.key)
		if !ok {
			t.Errorf("Get(%q) not found", tt.key)
			continue
		}
		if string(v) != tt.want {
			t.Errorf("Get(%q) = %q, want %q", tt.key, v, tt.want)
		}
	}
}

func TestLSMTree_UpdateOverwrite(t *testing.T) {
	lsm := NewLSMTree(10)
	lsm.Put("key", []byte("v1"))
	lsm.Put("key", []byte("v2"))

	v, ok := lsm.Get("key")
	if !ok || string(v) != "v2" {
		t.Errorf("expected 'v2', got %q (ok=%v)", v, ok)
	}
}

func TestLSMTree_Delete(t *testing.T) {
	lsm := NewLSMTree(10)
	lsm.Put("a", []byte("1"))
	lsm.Put("b", []byte("2"))

	lsm.Delete("a")

	if _, ok := lsm.Get("a"); ok {
		t.Error("Get('a') found after Delete")
	}
	v, ok := lsm.Get("b")
	if !ok || string(v) != "2" {
		t.Errorf("Get('b') = (%q, %v), want ('2', true)", v, ok)
	}
}

func TestLSMTree_DeleteAcrossFlush(t *testing.T) {
	lsm := NewLSMTree(3)
	lsm.Put("a", []byte("1"))
	lsm.Put("b", []byte("2"))
	lsm.Put("c", []byte("3"))
	// This triggers flush (memtable size = 3).

	// Now delete a key that's in the SSTable.
	lsm.Delete("b")

	if _, ok := lsm.Get("b"); ok {
		t.Error("Get('b') found after Delete across flush")
	}

	// Verify others survive.
	if _, ok := lsm.Get("a"); !ok {
		t.Error("Get('a') not found after sibling delete")
	}
}

func TestLSMTree_Flush(t *testing.T) {
	lsm := NewLSMTree(5)
	for i := 0; i < 5; i++ {
		lsm.Put(fmt.Sprintf("key-%d", i), []byte(fmt.Sprintf("val-%d", i)))
	}

	// After 5 inserts with threshold 5, memtable should have flushed.
	if lsm.L0Count() == 0 {
		t.Error("expected L0 SSTables after flush")
	}

	// All entries should still be readable.
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key-%d", i)
		want := fmt.Sprintf("val-%d", i)
		v, ok := lsm.Get(key)
		if !ok || string(v) != want {
			t.Errorf("Get(%q) = (%q, %v), want (%q, true)", key, v, ok, want)
		}
	}
}

func TestLSMTree_Compaction(t *testing.T) {
	lsm := NewLSMTree(3) // small threshold to trigger frequent flushes
	const N = 20
	for i := 0; i < N; i++ {
		lsm.Put(fmt.Sprintf("k%02d", i), []byte(fmt.Sprintf("v%02d", i)))
	}

	// After many flushes, compaction should have occurred.
	// All entries should be readable.
	for i := 0; i < N; i++ {
		key := fmt.Sprintf("k%02d", i)
		v, ok := lsm.Get(key)
		if !ok {
			t.Errorf("Get(%q) not found after compaction", key)
			continue
		}
		if string(v) != fmt.Sprintf("v%02d", i) {
			t.Errorf("Get(%q) = %q, want %q", key, v, fmt.Sprintf("v%02d", i))
		}
	}
}

func TestLSMTree_Scan(t *testing.T) {
	lsm := NewLSMTree(100) // high threshold to keep everything in memtable
	for _, k := range []string{"a", "b", "c", "d", "e"} {
		lsm.Put(k, []byte(k))
	}

	got := lsm.Scan("b", "d")
	if len(got) != 3 {
		t.Fatalf("Scan('b','d') returned %d entries, want 3", len(got))
	}
	expected := []string{"b", "c", "d"}
	for i, e := range got {
		if e.Key != expected[i] {
			t.Errorf("entry[%d].Key = %q, want %q", i, e.Key, expected[i])
		}
	}
}

func TestLSMTree_ScanAcrossLevels(t *testing.T) {
	lsm := NewLSMTree(3)
	// Write enough to trigger flush.
	lsm.Put("a", []byte("1"))
	lsm.Put("b", []byte("2"))
	lsm.Put("c", []byte("3"))
	// After flush, write more to memtable.
	lsm.Put("d", []byte("4"))
	lsm.Put("e", []byte("5"))

	entries := lsm.AllEntries()
	if len(entries) < 5 {
		t.Errorf("AllEntries returned %d, want at least 5", len(entries))
	}
}

func TestLSMTree_AllEntries(t *testing.T) {
	lsm := NewLSMTree(100)
	for i := 0; i < 10; i++ {
		lsm.Put(fmt.Sprintf("key-%02d", i), []byte(fmt.Sprintf("val-%02d", i)))
	}
	lsm.Delete("key-05")

	entries := lsm.AllEntries()
	if len(entries) != 9 {
		t.Errorf("AllEntries returned %d, want 9 (after 1 delete)", len(entries))
	}

	// Verify sorted order.
	for i := 1; i < len(entries); i++ {
		if entries[i].Key <= entries[i-1].Key {
			t.Errorf("AllEntries not sorted at index %d", i)
		}
	}
}

func TestSSTable_BinarySearch(t *testing.T) {
	entries := []SSTEntry{
		{Key: "a", Val: []byte("1")},
		{Key: "b", Val: []byte("2")},
		{Key: "c", Val: []byte("3")},
		{Key: "d", Val: []byte("4")},
		{Key: "e", Val: []byte("5")},
	}
	sst := newSSTable(entries, 0)

	v, deleted, found := sst.Search("c")
	if !found || deleted || string(v) != "3" {
		t.Errorf("SSTable.Search('c') = (%q, %v, %v), want ('3', false, true)", v, deleted, found)
	}

	_, _, found = sst.Search("z")
	if found {
		t.Error("SSTable.Search('z') should not find absent key")
	}
}
