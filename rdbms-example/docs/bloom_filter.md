# Bloom Filter

## What Is It?

A **Bloom Filter** is a space-efficient probabilistic data structure that
answers set-membership queries with the guarantee:

- **False negatives**: IMPOSSIBLE — if `MightContain` returns `false`, the key
  is **definitely absent**.
- **False positives**: Possible — `MightContain` may return `true` for a key
  that was never inserted.

This asymmetry is exactly what a database needs: we can safely skip any disk
read when the filter says "no", but we must verify on "maybe".

### Structure

A bit array of `m` bits + `k` independent hash functions:

```
bit array (m=16, k=3), after inserting "foo" and "bar":

Index:  0  1  2  3  4  5  6  7  8  9 10 11 12 13 14 15
Bits:   0  0  1  0  1  0  0  1  0  0  0  1  0  0  0  1
             ↑        ↑        ↑              ↑        ↑
          h1("foo") h2("foo")  h3("foo")   h1("bar")  ...
```

**Add(x)**: compute h₁(x), h₂(x), …, hₖ(x) → set those bits to 1.
**MightContain(x)**: check bits at h₁(x), …, hₖ(x) — if ALL are 1, return true.

### Optimal Parameters

Given n expected insertions and desired false-positive rate p:

```
m (bits)          = -n · ln(p) / (ln 2)²
k (hash functions)= (m/n) · ln 2
```

For n=1000, p=0.01: m≈9585 bits (~1.2 KB), k≈7 functions.

## Complexity

| Operation     | Time   | Space         |
|---------------|--------|---------------|
| Add           | O(k)   | O(m) total    |
| MightContain  | O(k)   | O(1) extra    |

k is a small constant (typically 3–10). Both operations are effectively O(1).

## Significance in Databases

Bloom Filters are widely used:
- **Apache Cassandra**: per-SSTable Bloom Filters eliminate 99% of disk reads for absent keys
- **HBase**: row-level Bloom Filters for `Get` operations
- **Google Bigtable**: original paper mentions Bloom Filters for SSTable lookups
- **RocksDB / LevelDB**: per-level Bloom Filters to gate SST file searches
- **PostgreSQL**: unused in core but proposed for outer join acceleration

The core insight: in a Log-Structured Merge Tree (LSM), data may be in any of
dozens of SSTables on disk. Without a Bloom Filter, every `SELECT pk=X` must
open and binary-search each file. With a filter per file, only the file(s)
that "might" contain the key are searched.

### Classic Use Case

```
GET key=42:
  For each SSTable level (L0→L1→L2→...):
    if bloom[level].MightContain(42):   ← O(k) in memory
      read SSTable, binary search         ← disk I/O (expensive)
    else:
      skip entirely                       ← saved disk I/O!
```

## Trade-offs

| Pro | Con |
|-----|-----|
| Extremely space-efficient (bits, not bytes) | Cannot delete (bits can't be unset) |
| Constant time O(k) regardless of set size | False positives cause wasted disk reads |
| Zero false negatives | Not suitable when false positives are unacceptable |
| Works across disk / network boundaries | Optimal parameters need upfront capacity estimate |

**Counting Bloom Filter** extends the design to support deletion by using
small counters instead of bits — at 3–4× the space cost.

**Cuckoo Filter** provides better lookup performance and supports deletion,
making it a modern alternative for write-heavy workloads.

## How It Is Used Here

Every table in ToyDB has a Bloom Filter (`internal/ds/bloom.go`) loaded into
memory at startup. It gates all primary-key existence checks.

```
SELECT * FROM users WHERE id = 999;   ← key that doesn't exist

① Bloom.MightContain(999) → false
   → STOP: skip B+ Tree lookup and disk read entirely

SELECT * FROM users WHERE id = 1;    ← key that exists

① Bloom.MightContain(1) → true (might exist)
② B+ Tree.Search(1) → rid=0 (confirmed!)
③ Pager.Read(0) → row
```

**On Insert**: `bloom.Add(pk)` — sets k bits.

**On Delete**: The filter is NOT updated — Bloom Filters do not support
deletion. After a delete, the filter may still report "might contain" for the
deleted key. The B+ Tree then gives the definitive "not found" answer via its
tombstone flag. This is correct and safe.

### Implementation Notes (`internal/ds/bloom.go`)

- Double hashing: `pos_i = (h1 + i·h2) mod m` with FNV-1a and FNV-1 variants
- `optimalM` and `optimalK` compute bit array size and hash count from n and p
- `FalsePositiveRate()` estimates current FPR from the number of set bits
- `Fill()` measures filter saturation (should stay below ~50% for good FPR)
- Default capacity: 1024 keys, 1% target FPR → ~9.6 KB bit array
