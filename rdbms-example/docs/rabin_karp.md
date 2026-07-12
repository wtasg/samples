# Rabin-Karp Rolling Hash Search

## What Is It?

**Rabin-Karp** is a string-searching algorithm that uses a **rolling hash**
to slide a fixed-size window over the text in amortised O(1) per step.
It efficiently finds all occurrences of a pattern of length m in text of length n.

### The Naive Approach (Why We Need Rolling Hash)

Brute force: for each of the n-m+1 starting positions, compare m characters.
Worst case: O(n·m) — e.g., searching "aaa...a" for "aaa...ab".

### Polynomial Hash

Treat each character as a digit in base B:

```
hash("abc") = 'a'·B² + 'b'·B¹ + 'c'·B⁰  (mod P)
            =  97·961  +  98·31  +  99·1   (mod 10⁹+7)
```

**Rolling**: remove the leftmost character, add the rightmost:

```
Window "abc" → "bcd":

new_hash = (old_hash - 'a'·B^(m-1)) · B + 'd'
         = (old_hash - text[i] · power) · B + text[i+m]   (mod P)
```

This update is O(1) — no need to recompute the whole window hash!

### Hash Collision

Two different strings may hash to the same value (collision). When
`hash(window) == hash(pattern)`, we verify by direct string comparison.
This keeps the algorithm correct while remaining fast on average.

```
Rabin-Karp("hello world", "world"):

power = B^(m-1) = 31^4 mod (10⁹+7) = 923521

patHash = hash("world") = 119764658  (example)

i=0: hash("hello") = 99162322 ≠ patHash → roll
i=1: hash("ello ") = 97273410 ≠ patHash → roll
...
i=6: hash("world") = 119764658 == patHash → verify text[6:11] == "world" ✓ → match!
```

## Complexity

| Case        | Time      | Notes |
|-------------|-----------|-------|
| Average     | O(n + m)  | Few hash collisions |
| Worst case  | O(n · m)  | All windows collide (extremely rare with good hash) |
| Space       | O(1)      | Only 3–4 scalar variables |

The worst case occurs only with adversarially crafted inputs; a large prime
modulus (10⁹+7) makes it negligible in practice.

## Significance in Databases

Rabin-Karp and rolling hashes appear in:
- **PostgreSQL `pg_trgm` extension**: trigram-based LIKE/ILIKE acceleration
- **Oracle Text**: full-text substring search
- **rsync**: Rabin fingerprinting to find changed blocks between files (Adler-32)
- **Git**: content-defined chunking in pack files uses rolling hashes
- **Plagiarism detection**: Rabin fingerprinting of document n-grams
- **grep / agrep**: line-level pattern matching

In a database context, `LIKE '%pattern%'` cannot use a B+ Tree index because
the leading `%` wildcard means there is no prefix to search by. Rabin-Karp
provides an efficient algorithm for this case without building a full inverted
index (which would cost O(n·m) space).

### vs KMP (Knuth-Morris-Pratt)

| | Rabin-Karp | KMP |
|--|------------|-----|
| Preprocessing | O(m) | O(m) |
| Search | O(n+m) avg, O(nm) worst | O(n+m) always |
| Multiple patterns | O(n + Σm) avg | Needs Aho-Corasick |
| Implementation | Simple | Moderate |

KMP is better when worst-case guarantees matter (e.g., security-sensitive
input). Rabin-Karp is simpler and better for **multiple pattern search**
(compare one window hash against many pattern hashes in O(1) each).

## Trade-offs

| Pro | Con |
|-----|-----|
| O(n+m) average — fast | O(nm) worst case (rare) |
| Simple to implement | Requires hash collision verification |
| O(1) space | Modular arithmetic adds overhead vs. direct compare |
| Extends naturally to multiple patterns | Not as cache-friendly as SSE4.2 `strstr` |
| Language/charset agnostic | Sensitive to hash function quality |

## How It Is Used Here

ToyDB uses Rabin-Karp (`internal/ds/rabinkarp.go`) for `LIKE` clauses that
start with `%` — patterns where no prefix index can help.

```
SELECT * FROM products WHERE name LIKE '%erry';   ← suffix
SELECT * FROM products WHERE name LIKE '%an%';    ← substring
SELECT * FROM products WHERE name LIKE '%Berry%'; ← substring (case-sensitive)

Each:
  ↓
Full table scan → for each row:
  ds.Contains(row["name"], "erry")   ← Rabin-Karp rolling hash
  ds.HasSuffix(row["name"], "erry")  ← Rabin-Karp on last window
```

The executor dispatches based on the LIKE pattern:

```
LIKE 'prefix%'  → Trie.PrefixSearch()      O(m + k)   ← fast
LIKE '%substr%' → Rabin-Karp full scan     O(n · (N+M)) avg
LIKE '%suffix'  → Rabin-Karp suffix check  O(n · M)
```

In a production DB this would be optimised with trigram indexes (GIN index in
PostgreSQL), but for a toy DB, Rabin-Karp is an excellent O(n) average-case
solution with no index overhead.

### Implementation Notes (`internal/ds/rabinkarp.go`)

- Base = 31, Mod = 1_000_000_007 (Mersenne-adjacent prime)
- `power = B^(m-1) mod P` precomputed once per search call
- Collision verified by `text[i:i+m] == pattern` (Go string comparison is O(m))
- `Contains`, `HasSuffix`, `HasPrefix` are convenience wrappers
- `MultiSearch` runs Rabin-Karp for multiple patterns in one pass
