# Skip List

## What Is It?

A **Skip List** is a probabilistic ordered list data structure that allows O(log n)
search, insertion, and deletion. It maintains multiple layers of linked lists,
where each layer skips over multiple elements of the layer below, acting as
an "express lane".

```
L2: [Head] ───────────────────────────▶ [15] ──────────────────────▶ [Nil]
L1: [Head] ─────────────▶ [5] ────────▶ [15] ─────────▶ [30] ──────▶ [Nil]
L0: [Head] ──▶ [3] ─────▶ [5] ──▶ [10] ──▶ [15] ──▶ [20] ──▶ [30] ──▶ [Nil]
```

### Key Properties
- **Probabilistic height** — heights of nodes are determined randomly using a geometric distribution.
- **Ordered iteration** — leaf layer (L0) is a sorted singly-linked list.
- **Simplicity** — easier to implement and concurrent-safe compared to Red-Black or AVL trees.

## Complexity

| Operation | Average (Expected) | Worst Case |
|---|---|---|
| Search | O(log n) | O(n) |
| Insert | O(log n) | O(n) |
| Delete | O(log n) | O(n) |

## Significance in Databases

Skip Lists are highly valued in storage engines for their ease of concurrent implementation. Used in:
- **LevelDB / RocksDB** — the default Memtable implementation.
- **Apache HBase** — ConcurrentSkipListMap.
- **Redis** — sorted sets (ZSET).

## Trade-offs

| Pro | Con |
|---|---|
| O(log n) expected operations | Extra memory overhead for forward pointers |
| Highly concurrent / lock-free friendly | Worst-case O(n) possible (rare) |
| Simple sorted range queries | Worse memory cache locality than B-Trees |

## How It Is Used Here

In DocDB, the Skip List (`internal/ds/skiplist.go`) is utilized as:
1. **The Memtable** inside the LSM Tree to buffer and sort incoming writes before flushing to disk.
2. **A Sorted Index Builder** to execute sorted query outputs (e.g. `.sort({"price": 1})`) efficiently.
