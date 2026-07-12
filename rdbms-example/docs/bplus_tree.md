# B+ Tree

## What Is It?

A **B+ Tree** is a self-balancing, ordered tree data structure that stores all
data values in its **leaf nodes** while internal nodes hold only routing keys.
Leaf nodes are linked together as a doubly-linked list, enabling efficient
sequential access without returning to the root.

```
Order-4 B+ Tree example (max 3 keys per leaf, max 4 children per internal node):

             [ 20 | 40 ]           ← Internal node (routing keys only)
            /      |      \
  [5|10|15]  [20|25|35]  [40|50|60]  ← Leaf nodes (actual PK→rowID pairs)
      ↕            ↕           ↕      ← Doubly-linked list
```

### Key Properties
- **All data in leaves** — internal nodes are navigation-only.
- **Sorted leaves linked** — O(1) to move to the next/previous leaf.
- **Order m**: every internal node has ⌈m/2⌉ to m children; every leaf holds
  ⌈(m-1)/2⌉ to m-1 entries.
- **Height**: O(log_m n) — extremely shallow for large m (e.g., m=100 means
  a million-row table is at most 3 levels deep).

## Complexity

| Operation   | Average    | Worst Case  |
|-------------|-----------|-------------|
| Search      | O(log n)  | O(log n)    |
| Insert      | O(log n)  | O(log n)    |
| Delete      | O(log n)  | O(log n)    |
| Range scan  | O(log n + k) | O(log n + k) |

(n = entries, k = result set size)

## Significance in Databases

The B+ Tree is **the** canonical database index structure. It is used by:
- **MySQL InnoDB** — primary and secondary indexes
- **PostgreSQL** — default index type (`CREATE INDEX`)
- **SQLite** — entire database is one B-Tree file
- **Oracle** — B*-Tree indexes

Reasons it dominates:
1. **Disk-friendly**: A high order (e.g., m = 200+) means few I/O operations
   per lookup — each node fits in one disk page.
2. **Range queries** are efficient because leaf nodes are linked; a range scan
   finds the starting leaf in O(log n) and then iterates forward.
3. **Predictable performance** — all lookups hit the same depth.

## Trade-offs

| Pro | Con |
|-----|-----|
| O(log n) everything | More complex than a hash map |
| Excellent range queries | Not the fastest for pure point lookup (hash beats it) |
| Ordered traversal for free | Wasted space in partially-filled nodes |
| Disk-page aligned (high order) | Node splits during insert can cascade |

**vs Hash Index**: Hash gives O(1) point lookup but cannot do range scans.
**vs AVL Tree**: AVL has lower constant but poor disk locality for large data.

## How It Is Used Here

In ToyDB every table has one B+ Tree (`internal/ds/bptree.go`) that maps the
**primary key (INT)** to a **rowID** (byte offset index in the `.rows` file).

```
INSERT INTO users VALUES (42, 'Alice', 30);
  ↓
1. Check Bloom Filter: MightContain(42)?
2. B+ Tree Insert(42, rid)
3. Pager.Write(row) → rid

SELECT * FROM users WHERE id = 42;
  ↓
1. Bloom Filter: MightContain(42) → false? → stop (definitely absent)
2. B+ Tree Search(42) → rid
3. Pager.Read(rid) → row

SELECT * FROM users WHERE id BETWEEN 10 AND 50;
  ↓
B+ Tree RangeScan(10, 50) → walks leaf linked list → []BPEntry{rid...}
```

The implementation uses **lazy deletion** (tombstone flags in leaf nodes)
for simplicity — deleted entries are marked but not removed until a full
tree rebuild. This avoids the complexity of merge/redistribution on underflow
while still delivering correct query results.

### Implementation Notes (`internal/ds/bptree.go`)

- Order: `BPOrder = 4` (configurable constant)
- Leaf split: left keeps `keys[:mid]`, right gets `keys[mid:]`, push up `right.keys[0]`
- Internal split: left keeps `keys[:mid]`, right gets `keys[mid+1:]`, push up `keys[mid]`
- Parent lookup during split uses DFS `findParent` (acceptable for toy scale)
- `AllEntries()` walks the leaf linked list from the leftmost leaf — O(n)
