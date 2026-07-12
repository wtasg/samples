package ds

import (
	"reflect"
	"testing"
)

func TestRabinKarp_BasicSearch(t *testing.T) {
	tests := []struct {
		text, pattern string
		want          []int
	}{
		{"abcabc", "abc", []int{0, 3}},
		{"hello world", "world", []int{6}},
		{"aaaa", "aa", []int{0, 1, 2}},
		{"abcdef", "xyz", nil},
		{"", "a", nil},
		{"a", "", nil},
		{"abc", "abcd", nil},
	}
	for _, tc := range tests {
		got := Search(tc.text, tc.pattern)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("Search(%q, %q) = %v, want %v", tc.text, tc.pattern, got, tc.want)
		}
	}
}

func TestRabinKarp_Contains(t *testing.T) {
	if !Contains("the quick brown fox", "brown") {
		t.Error("Contains(brown) should be true")
	}
	if Contains("hello", "world") {
		t.Error("Contains(world) in hello should be false")
	}
}

func TestRabinKarp_HasSuffix(t *testing.T) {
	if !HasSuffix("foobar", "bar") {
		t.Error("HasSuffix(bar) should be true")
	}
	if HasSuffix("foobar", "foo") {
		t.Error("HasSuffix(foo) should be false")
	}
	if !HasSuffix("abc", "") {
		t.Error("HasSuffix('') should always be true")
	}
}

func TestRabinKarp_HasPrefix(t *testing.T) {
	if !HasPrefix("Alice", "Al") {
		t.Error("HasPrefix(Al) in Alice should be true")
	}
	if HasPrefix("Alice", "Bo") {
		t.Error("HasPrefix(Bo) in Alice should be false")
	}
}

func TestRabinKarp_LongText(t *testing.T) {
	// Build a text of 10k 'a's, embed "needle" in the middle.
	text := make([]byte, 10000)
	for i := range text {
		text[i] = 'a'
	}
	needle := "needle"
	pos := 5000
	copy(text[pos:], needle)

	matches := Search(string(text), needle)
	if len(matches) != 1 || matches[0] != pos {
		t.Errorf("Long text search: got %v, want [%d]", matches, pos)
	}
}

func TestRabinKarp_MultiSearch(t *testing.T) {
	text := "the cat sat on the mat"
	patterns := []string{"cat", "mat", "bat"}
	results := MultiSearch(text, patterns)

	if _, ok := results["cat"]; !ok {
		t.Error("cat should be found")
	}
	if _, ok := results["mat"]; !ok {
		t.Error("mat should be found")
	}
	if _, ok := results["bat"]; ok {
		t.Error("bat should not be found")
	}
}
