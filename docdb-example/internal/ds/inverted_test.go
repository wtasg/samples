package ds

import (
	"sort"
	"testing"
)

func TestInvertedIndex_AddAndSearch(t *testing.T) {
	idx := NewInvertedIndex()
	idx.Add("category", "electronics", "doc1")
	idx.Add("category", "electronics", "doc2")
	idx.Add("category", "clothing", "doc3")
	idx.Add("category", "electronics", "doc4")

	got := idx.Search("category", "electronics")
	if len(got) != 3 {
		t.Fatalf("Search returned %d results, want 3", len(got))
	}

	sort.Strings(got)
	expected := []string{"doc1", "doc2", "doc4"}
	for i, id := range got {
		if id != expected[i] {
			t.Errorf("result[%d] = %q, want %q", i, id, expected[i])
		}
	}
}

func TestInvertedIndex_SearchEmpty(t *testing.T) {
	idx := NewInvertedIndex()
	got := idx.Search("nonexistent", "value")
	if len(got) != 0 {
		t.Errorf("Search on empty index returned %d results", len(got))
	}
}

func TestInvertedIndex_PrefixSearch(t *testing.T) {
	idx := NewInvertedIndex()
	idx.Add("name", "Alice", "doc1")
	idx.Add("name", "Albert", "doc2")
	idx.Add("name", "Bob", "doc3")
	idx.Add("name", "Alvin", "doc4")

	got := idx.PrefixSearch("name", "Al")
	if len(got) != 3 {
		t.Fatalf("PrefixSearch('Al') returned %d results, want 3", len(got))
	}

	sort.Strings(got)
	expected := []string{"doc1", "doc2", "doc4"}
	for i, id := range got {
		if id != expected[i] {
			t.Errorf("result[%d] = %q, want %q", i, id, expected[i])
		}
	}
}

func TestInvertedIndex_ContainsSearch(t *testing.T) {
	idx := NewInvertedIndex()
	idx.Add("description", "red apple", "doc1")
	idx.Add("description", "green apple pie", "doc2")
	idx.Add("description", "banana split", "doc3")

	got := idx.ContainsSearch("description", "apple")
	if len(got) != 2 {
		t.Fatalf("ContainsSearch('apple') returned %d results, want 2", len(got))
	}
}

func TestInvertedIndex_Delete(t *testing.T) {
	idx := NewInvertedIndex()
	idx.Add("tag", "sale", "doc1")
	idx.Add("tag", "sale", "doc2")

	if !idx.Delete("tag", "sale", "doc1") {
		t.Fatal("Delete returned false")
	}

	got := idx.Search("tag", "sale")
	if len(got) != 1 || got[0] != "doc2" {
		t.Errorf("after delete, Search returned %v, want [doc2]", got)
	}
}

func TestInvertedIndex_DeleteDoc(t *testing.T) {
	idx := NewInvertedIndex()
	idx.Add("name", "Alice", "doc1")
	idx.Add("category", "user", "doc1")
	idx.Add("name", "Bob", "doc2")
	idx.Add("category", "admin", "doc2")

	idx.DeleteDoc("doc1")

	if got := idx.Search("name", "Alice"); len(got) != 0 {
		t.Errorf("after DeleteDoc, name=Alice has %d results, want 0", len(got))
	}
	if got := idx.Search("category", "user"); len(got) != 0 {
		t.Errorf("after DeleteDoc, category=user has %d results, want 0", len(got))
	}

	// doc2 should be unaffected.
	if got := idx.Search("name", "Bob"); len(got) != 1 {
		t.Errorf("doc2 should survive, got %d results", len(got))
	}
}

func TestInvertedIndex_DuplicateAdd(t *testing.T) {
	idx := NewInvertedIndex()
	idx.Add("tag", "sale", "doc1")
	idx.Add("tag", "sale", "doc1") // duplicate

	got := idx.Search("tag", "sale")
	if len(got) != 1 {
		t.Errorf("duplicate Add created %d entries, want 1", len(got))
	}
}

func TestInvertedIndex_FieldTerms(t *testing.T) {
	idx := NewInvertedIndex()
	idx.Add("color", "red", "d1")
	idx.Add("color", "blue", "d2")
	idx.Add("color", "green", "d3")

	terms := idx.FieldTerms("color")
	if len(terms) != 3 {
		t.Errorf("FieldTerms returned %d terms, want 3", len(terms))
	}
}

func TestInvertedIndex_FieldCount(t *testing.T) {
	idx := NewInvertedIndex()
	idx.Add("a", "1", "d1")
	idx.Add("b", "2", "d2")

	if idx.FieldCount() != 2 {
		t.Errorf("FieldCount() = %d, want 2", idx.FieldCount())
	}
}

func TestPostingList_Contains(t *testing.T) {
	pl := &PostingList{}
	pl.Add("doc1")
	pl.Add("doc2")

	if !pl.Contains("doc1") {
		t.Error("Contains('doc1') = false, want true")
	}
	if pl.Contains("doc3") {
		t.Error("Contains('doc3') = true, want false")
	}
}
