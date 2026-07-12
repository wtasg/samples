# Trie (Prefix Tree)

## What Is It?

A **Trie** (from re*trie*val) is an ordered tree where each node represents
one character of a string. The path from the root to a terminal node spells
out a complete key. No explicit keys are stored in nodes — the position in the
tree *is* the key.

```
Trie storing: "apple", "app", "banana", "band"

root
├─ a
│  └─ p
│     └─ p  [terminal: "app"]
│        └─ l
│           └─ e  [terminal: "apple"]
└─ b
   └─ a
      ├─ n
      │  └─ a
      │     └─ n
      │        └─ a  [terminal: "banana"]
      └─ n
         └─ d  [terminal: "band"]
```

### Key Properties
- Each edge is one character.
- Lookups are O(m) where m is the key length — **independent of the number of keys n**.
- All keys sharing a prefix share the same path prefix in the tree.
- `PrefixSearch(prefix)` collects all terminal nodes in the subtree rooted at
  the prefix endpoint — this is the defining advantage over hash tables.

## Complexity

| Operation       | Time           | Space         |
|-----------------|---------------|---------------|
| Insert          | O(m)          | O(m · Σ)      |
| Exact search    | O(m)          | O(1) extra    |
| HasPrefix       | O(m)          | O(1) extra    |
| PrefixSearch    | O(m + k)      | O(k)          |
| Delete          | O(m)          | O(1) extra    |

(m = key length, k = number of matching entries, Σ = alphabet size)

This implementation uses a fixed array of 128 ASCII children per node.

## Significance in Databases

Tries appear in:
- **Autocomplete / search suggestions**: return all words with a given prefix in O(m+k)
- **IP routing tables**: longest-prefix-match in routers
- **Dictionary compression**: LZW algorithm
- **Full-text inverted indexes**: prefix expansion in `LIKE 'prefix%'`
- **Database system catalogs**: fast column/table name lookup

### vs Hash Map for Lookups
- Hash map: O(1) average, but no ordering and no prefix search.
- Trie: O(m) always, plus O(m+k) prefix search — superior when prefix queries matter.

### vs Sorted Array / B+ Tree for Prefix Search
- Sorted array: O(log n) to find prefix start, O(k) to collect — total O(log n + k).
- Trie: O(m + k) — better when m < log n, which is typical for short column names.

## Trade-offs

| Pro | Con |
|-----|-----|
| O(m) lookup — independent of n | Memory heavy: 128 child pointers per node |
| Prefix search is natural | Poor cache locality (sparse node arrays) |
| No hash collision possible | Not good for long, random keys (DNAs, hashes) |
| Lexicographic order for free | Alphabet size blows up memory for Unicode |

**Memory optimisation** (not in this toy): Compressed Trie (Patricia Trie) merges
single-child chains into one edge to reduce node count from O(n·m) to O(n).

## How It Is Used Here

ToyDB uses the Trie in two places:

### 1. Schema Catalog (`internal/catalog/catalog.go`)

The catalog wraps a Trie to store table names → TableSchema. This gives:
- O(m) table name lookup (m = table name length)
- `TablesWithPrefix("ord")` — find all tables starting with "ord" in O(m + k)

```go
catalog.trie.SetMeta("users", tableSchema)
catalog.trie.GetMeta("users")  // O(m)
```

### 2. TEXT Column Secondary Index (`internal/engine/table.go`)

Every TEXT column gets a Trie that maps **string value → []rowID**.

```
INSERT INTO users VALUES (1, 'Alice', 30);
  ↓
nameTrie.Insert("Alice", rid=0)

SELECT * FROM users WHERE name LIKE 'Al%';
  ↓
nameTrie.PrefixSearch("Al") → [rid=0, ...]   ← O(m + k)
pager.Read(rid=0) → {id:1, name:"Alice", ...}
```

Without the Trie, this query would require a full sequential scan (O(n)).
With the Trie, only rows whose name starts with "Al" are fetched — O(m + k)
where k is the number of matching rows.

### Implementation Notes (`internal/ds/trie.go`)

- Each node: `[128]*trieNode` (fixed ASCII array) + `[]uint32 rowIDs` + `any meta`
- `Insert(word, rowID)` populates the rowID slice at the terminal node
- `PrefixSearch(prefix)` navigates to the prefix node, then DFS-collects all rowIDs
- `Delete(word, rowID)` removes one rowID from the terminal node's list
- `SetMeta/GetMeta` stores arbitrary schema objects (used by catalog)
