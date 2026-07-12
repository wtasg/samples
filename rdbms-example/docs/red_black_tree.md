# Red-Black Tree

## What Is It?

A **Red-Black Tree** (RBT) is a self-balancing binary search tree where each
node carries a one-bit "color" (RED or BLACK). The coloring rules ensure the
tree stays approximately balanced after every mutation.

```
Example (key: color):
              13·B
           /        \
         8·R          17·B
        /   \        /    \
      1·B   11·B  15·R   25·R
        \
        6·R
```

### Five Invariants (from CLRS)
1. Every node is RED or BLACK.
2. The root is BLACK.
3. Every nil sentinel is BLACK.
4. If a node is RED, both its children are BLACK (no two consecutive reds).
5. All paths from any node to its nil descendants have the **same black-height**.

These invariants guarantee: height ≤ 2·log₂(n+1).

### Rotations

**Left rotation** on x:           **Right rotation** on y:
```
  x                 y               y                 x
 / \               / \             / \               / \
A   y     →      x   C           x   C     →      A   y
   / \          / \             / \                   / \
  B   C        A   B           A   B                 B   C
```

## Complexity

| Operation | Time      | Space |
|-----------|-----------|-------|
| Search    | O(log n)  | O(1)  |
| Insert    | O(log n)  | O(1)  |
| Delete    | O(log n)  | O(1)  |
| In-order  | O(n)      | O(n)  |

## Significance in Databases

Red-Black Trees appear in:
- **Linux kernel**: `struct rb_root` — completely fair scheduler, virtual memory areas
- **Java `TreeMap`/`TreeSet`**: underlying data structure
- **C++ `std::map`/`std::set`**: typically RBT
- **Databases**: in-memory secondary indexes, active transaction tables,
  lock manager tables (sorted by transaction ID)

The RBT is preferred over AVL for databases because:
- **Fewer rotations on insert/delete** than AVL (AVL is stricter → more rebalancing)
- **In-order traversal is O(n)** — used to emit sorted results

### vs AVL Tree

| | Red-Black Tree | AVL Tree |
|--|----------------|----------|
| Balance factor | ≤ 2× height difference | Strict height balance |
| Insert rotations | ≤ 2 | ≤ 2 |
| Delete rotations | ≤ 3 | O(log n) |
| Search speed | Slightly slower (less balanced) | Slightly faster |
| Use case | More writes | More reads |

## Trade-offs

| Pro | Con |
|-----|-----|
| O(log n) insert/delete | More complex than a skip list |
| No rebalancing needed after most ops | Cannot do range scans as efficiently as B+ Tree |
| Stable sorted order | Not cache-friendly (pointer chasing) |
| Supports duplicate keys (accumulate) | Memory overhead (color bit + 3 pointers per node) |

## How It Is Used Here

In ToyDB, the Red-Black Tree (`internal/ds/rbtree.go`) is used for **in-memory
ORDER BY sorting** of query results.

```
SELECT * FROM products ORDER BY price DESC;
  ↓
1. Execute query → collect matching rows []Row
2. For each row, RBTree.Insert(price, row)
   - Rows with equal prices accumulate in the node's []any slice
3. RBTree.InOrderDesc() → walks tree right→root→left
4. Return sorted []Row to pretty-printer
```

This demonstrates the RBT's key strength: in-order traversal yields a sorted
sequence in O(n), making it ideal for producing sorted query results without
a separate pass.

### Why Not Use sort.Slice?

We **do** use `sort.Slice` for TEXT column ORDER BY (via `orderByString`).
For INT columns we use the RBT as a deliberate showcase of the data structure.
In a production database, the optimizer would choose the best physical sort
strategy (external merge sort, index scan in order, or in-memory sort) based
on available indexes and memory budget.

### Implementation Notes (`internal/ds/rbtree.go`)

- Sentinel nil node avoids nil pointer checks throughout (standard CLRS approach)
- Duplicate keys accumulate values in `[]any` slice on the node
- `InOrder()` and `InOrderDesc()` both traverse the whole tree in O(n)
- Delete uses successor-based transplant (CLRS `RB-DELETE`)
