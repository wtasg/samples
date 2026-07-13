# DocDB — Data Structures & Algorithms Overview

DocDB is a toy single-user NoSQL document database built in Go that deliberately
exposes the data structures and algorithms that power real document databases.
Each structure was chosen because it solves a real problem that arises when
building a NoSQL database.

## Architecture at a Glance

```
NoSQL Input (REPL)
      │
      ▼
 parser/parser.go  — hand-written parser → Command AST
      │
      ▼
engine/executor.go — resolves collection, dispatches to collection layer
      │
   ┌──┴──────────────────────────────────────────────┐
   │                  Per-Collection                 │
   │                                                 │
   │  ┌────────────────┐     ┌──────────────────────┐│
   │  │   Hash Map     │────▶│      LSM Tree        ││
   │  │  (O(1) Get)    │     │  (Skip List memtable)││
   │  └────────────────┘     └──────────┬───────────┘│
   │                                    │            │
   │  ┌────────────────┐                ▼            │
   │  │ Inverted Index │     ┌──────────────────────┐│
   │  │ (Secondary Qs) │────▶│       Store          ││
   │  └────────────────┘     │  .docs file (JSON)   ││
   │                         └──────────────────────┘│
   │                                                 │
   │  Skip List: ordered insertion & sorted traversal│
   │                                                 │
   │  Bloom Filter: per-SST lookup gate in LSM Tree  │
   └─────────────────────────────────────────────────┘
```

## Query Dispatch Table

| Command / Filter | Structure used | Complexity |
|---|---|---|
| `db.col.find({"_id": X})` | Hash Map | O(1) |
| `db.col.find({"field": val})` | Inverted Index → Hash Map | O(k) |
| `db.col.find({"field": {"$prefix": "pre"}})` | Inverted Index prefix search | O(k) |
| `db.col.find({"field": {"$contains": "sub"}})` | Inverted Index contains search | O(k) |
| `db.col.find({"field": {"$gt": val}})` | Full scan + predicate | O(n) |
| `.sort({"field": 1})` | Skip List in-order traversal | O(n log n) |

Where `n` = documents in collection, `k` = result set size.

## Document Index

| File | Covers |
|---|---|
| [lsm_tree.md](lsm_tree.md) | LSM Tree — primary write-optimized storage engine |
| [hash_map.md](hash_map.md) | Hash Map — O(1) document ID index |
| [skip_list.md](skip_list.md) | Skip List — ordered memtable & sorting |
| [bloom_filter.md](bloom_filter.md) | Bloom Filter — per-SST lookup gate |
| [inverted_index.md](inverted_index.md) | Inverted Index — secondary field indexes |
