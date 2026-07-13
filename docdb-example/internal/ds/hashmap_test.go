package ds

import (
	"fmt"
	"testing"
)

func TestHashMap_PutAndGet(t *testing.T) {
	m := NewHashMap()
	m.Put("alpha", []byte("1"))
	m.Put("bravo", []byte("2"))
	m.Put("charlie", []byte("3"))

	tests := []struct {
		key  string
		want string
	}{
		{"alpha", "1"},
		{"bravo", "2"},
		{"charlie", "3"},
	}
	for _, tt := range tests {
		v, ok := m.Get(tt.key)
		if !ok {
			t.Errorf("Get(%q) not found", tt.key)
			continue
		}
		if string(v) != tt.want {
			t.Errorf("Get(%q) = %q, want %q", tt.key, v, tt.want)
		}
	}

	if m.Size() != 3 {
		t.Errorf("Size() = %d, want 3", m.Size())
	}
}

func TestHashMap_UpdateExisting(t *testing.T) {
	m := NewHashMap()
	m.Put("key", []byte("v1"))
	m.Put("key", []byte("v2"))

	v, ok := m.Get("key")
	if !ok || string(v) != "v2" {
		t.Errorf("expected updated value 'v2', got %q (ok=%v)", v, ok)
	}
	if m.Size() != 1 {
		t.Errorf("Size() = %d, want 1", m.Size())
	}
}

func TestHashMap_Delete(t *testing.T) {
	m := NewHashMap()
	m.Put("a", []byte("1"))
	m.Put("b", []byte("2"))
	m.Put("c", []byte("3"))

	if !m.Delete("b") {
		t.Fatal("Delete('b') returned false")
	}
	if _, ok := m.Get("b"); ok {
		t.Error("Get('b') found after Delete")
	}
	if m.Size() != 2 {
		t.Errorf("Size() = %d, want 2", m.Size())
	}

	if m.Delete("z") {
		t.Error("Delete('z') returned true for non-existent key")
	}
}

func TestHashMap_Has(t *testing.T) {
	m := NewHashMap()
	m.Put("exists", []byte("yes"))

	if !m.Has("exists") {
		t.Error("Has('exists') = false, want true")
	}
	if m.Has("nope") {
		t.Error("Has('nope') = true, want false")
	}
}

func TestHashMap_Keys(t *testing.T) {
	m := NewHashMap()
	m.Put("x", []byte("1"))
	m.Put("y", []byte("2"))
	m.Put("z", []byte("3"))

	keys := m.Keys()
	if len(keys) != 3 {
		t.Errorf("Keys() len = %d, want 3", len(keys))
	}

	seen := make(map[string]bool)
	for _, k := range keys {
		seen[k] = true
	}
	for _, k := range []string{"x", "y", "z"} {
		if !seen[k] {
			t.Errorf("Keys() missing %q", k)
		}
	}
}

func TestHashMap_Resize(t *testing.T) {
	m := NewHashMap()
	const N = 100
	for i := 0; i < N; i++ {
		m.Put(fmt.Sprintf("key-%d", i), []byte(fmt.Sprintf("val-%d", i)))
	}

	if m.Size() != N {
		t.Errorf("Size() = %d, want %d", m.Size(), N)
	}

	// Verify all entries survive resize.
	for i := 0; i < N; i++ {
		key := fmt.Sprintf("key-%d", i)
		want := fmt.Sprintf("val-%d", i)
		v, ok := m.Get(key)
		if !ok || string(v) != want {
			t.Errorf("Get(%q) = (%q, %v), want (%q, true)", key, v, ok, want)
		}
	}
}

func TestHashMap_LoadFactor(t *testing.T) {
	m := NewHashMap()
	if m.LoadFactor() != 0 {
		t.Errorf("LoadFactor() = %f, want 0", m.LoadFactor())
	}

	m.Put("a", []byte("1"))
	lf := m.LoadFactor()
	if lf <= 0 {
		t.Errorf("LoadFactor() = %f after insert, want > 0", lf)
	}
}

func TestHashMap_Entries(t *testing.T) {
	m := NewHashMap()
	m.Put("a", []byte("1"))
	m.Put("b", []byte("2"))

	entries := m.Entries()
	if len(entries) != 2 {
		t.Errorf("Entries() len = %d, want 2", len(entries))
	}
}
