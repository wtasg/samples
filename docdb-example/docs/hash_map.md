# Robin Hood Hash Map

## What Is It?

A **Robin Hood Hash Map** is an open-addressing hash table that resolves collisions
using linear probing but minimizes search times by displacing elements that are
"rich" (close to their home buckets) in favor of elements that are "poor" (far
from their home buckets).

```
Inserting key "charlie" (hashes to index 2):
Index:    0      1      2          3          4
Buckets: [ ]   [A(0)]  [B(0)]   [C(1)]      [ ]
                        ▲
                     Home slot (occupied by B, probe distance 0)
                     C is displaced to index 3 (probe distance 1)
```

### Key Properties
- **Open addressing** — all entries live in a single flat array; no linked lists or external chaining.
- **Probe Distance Relative Fairness** — elements farther from their starting hash slots can steal slots from closer ones.
- **Low Variance** — search times are tightly bounded.

## Complexity

| Operation | Average | Worst Case |
|---|---|---|
| Search | O(1) | O(n) |
| Insert | O(1) | O(n) |
| Delete | O(1) | O(n) |

## Significance in Databases

In-memory hash tables are crucial for secondary/primary indexes, hash joins,
aggregation tables, and lock managers.
Robin Hood hashing is used in modern platforms like Rust's standard library
HashMap (up to 1.36) and Swift's dictionary due to cache locality and speed.

## Trade-offs

| Pro | Con |
|---|---|
| Excellent CPU cache locality (flat array) | Resizing requires full table rehash |
| Low lookup latency variance | Performance degrades if load factor > 80% |
| Space efficient (no list node pointer overhead) | Tombstones required for deletion |

## How It Is Used Here

In DocDB, the Robin Hood Hash Map (`internal/ds/hashmap.go`) manages O(1) primary key
lookups (`GET` by `_id`). When a document is fetched directly by its ID, it utilizes
this map instead of traversing the LSM Tree layers on disk, speeding up NoSQL access.
