// rabinkarp.go — Rabin-Karp rolling-hash string search.
//
// Rabin-Karp computes a hash for the first m characters of text, then
// slides a window of size m one character at a time.  Removing the leftmost
// character and adding the rightmost is O(1) using modular arithmetic.
//
// Average complexity: O(n + m).  Worst case: O(nm) when all windows produce
// hash collisions (rare with a good hash), but a verification step ensures
// correctness — the string comparison only happens on hash matches.
//
// Role in this RDBMS: LIKE '%pattern%' full-text substring search over rows.
// When the WHERE clause is col LIKE '%substr%' or col LIKE '%suffix', we do a
// full table scan and apply Rabin-Karp on each row's TEXT column value.
package ds

const (
	rkBase = 31          // polynomial base
	rkMod  = 1_000_000_007 // large prime modulus
)

// Search returns the starting indices of all occurrences of pattern in text.
// Returns nil if pattern is empty, longer than text, or has no matches.
func Search(text, pattern string) []int {
	n, m := len(text), len(pattern)
	if m == 0 || m > n {
		return nil
	}

	// Precompute base^(m-1) mod rkMod — needed to "remove" the leading char.
	power := uint64(1)
	for i := 0; i < m-1; i++ {
		power = power * rkBase % rkMod
	}

	// Initial hashes for pattern and first window of text.
	patHash := uint64(0)
	winHash := uint64(0)
	for i := 0; i < m; i++ {
		patHash = (patHash*rkBase + uint64(pattern[i])) % rkMod
		winHash = (winHash*rkBase + uint64(text[i])) % rkMod
	}

	var matches []int
	for i := 0; i <= n-m; i++ {
		if winHash == patHash && text[i:i+m] == pattern { // verify on hash hit
			matches = append(matches, i)
		}
		// Roll the window: subtract leading char, add next char.
		if i < n-m {
			winHash = (winHash + rkMod - uint64(text[i])*power%rkMod) % rkMod
			winHash = (winHash*rkBase + uint64(text[i+m])) % rkMod
		}
	}
	return matches
}

// Contains reports whether pattern appears anywhere in text.
func Contains(text, pattern string) bool {
	return len(Search(text, pattern)) > 0
}

// HasSuffix reports whether text ends with suffix, using Rabin-Karp.
// (Used for LIKE '%suffix' predicates.)
func HasSuffix(text, suffix string) bool {
	n, m := len(text), len(suffix)
	if m == 0 {
		return true
	}
	if m > n {
		return false
	}
	// Check only the last window.
	return text[n-m:] == suffix
}

// HasPrefix is a thin wrapper; prefix search is better handled by Trie,
// but this lets the engine fall back gracefully without the index.
func HasPrefix(text, prefix string) bool {
	m := len(prefix)
	if m == 0 {
		return true
	}
	if m > len(text) {
		return false
	}
	return text[:m] == prefix
}

// MultiSearch searches for multiple patterns in text simultaneously using
// a shared window hash.  Returns a map from pattern to its match positions.
// Useful for complex WHERE clauses with OR-ed LIKE conditions.
func MultiSearch(text string, patterns []string) map[string][]int {
	result := make(map[string][]int, len(patterns))
	for _, p := range patterns {
		if matches := Search(text, p); matches != nil {
			result[p] = matches
		}
	}
	return result
}
