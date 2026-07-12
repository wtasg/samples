# ToyDB — Data Structures & Algorithms Overview

ToyDB is a toy single-user RDBMS built in Go that deliberately exposes the
data structures and algorithms that power real databases. Each structure was
chosen because it solves a real problem that arises when building a database.

## Architecture at a Glance

```
SQL Input (REPL)
      │
      ▼
 parser/parser.go  — hand-written lexer + recursive-descent parser → AST
      │
      ▼
engine/executor.go — resolves table, dispatches to optimal DS path
      │
  ┌───┴──────────────────────────────────────────────┐
  │                  Per-Table                        │
  │                                                   │
  │  ┌────────────────┐  ①  ┌──────────────────────┐ │
  │  │  Bloom Filter  │────▶│    B+ Tree (index)   │ │
  │  │  (existence)   │     │  PK → rowID          │ │
  │  └────────────────┘     └──────────┬───────────┘ │
  │                                    │ ②            │
  │  ┌────────────────┐                ▼             │
  │  │  Trie (index)  │     ┌──────────────────────┐ │
  │  │  TEXT col →    │     │  Pager (storage)     │ │
  │  │  rowID list    │────▶│  .rows file (JSON)   │ │
  │  └────────────────┘  ③  └──────────────────────┘ │
  │                                                   │
  │  ┌────────────────┐                               │
  │  │  Red-Black Tree│  ORDER BY — sort results      │
  │  │  (in-memory)   │                               │
  │  └────────────────┘                               │
  │                                                   │
  │  Rabin-Karp rolling hash — LIKE '%substr%' scan   │
  └───────────────────────────────────────────────────┘
```

## Query Dispatch Table

| WHERE clause                | Structure used          | Complexity         |
|-----------------------------|-------------------------|--------------------|
| `pk = X`                    | Bloom Filter → B+ Tree  | O(1) → O(log n)    |
| `pk BETWEEN lo AND hi`      | B+ Tree range scan      | O(log n + k)       |
| `col LIKE 'prefix%'`        | Trie prefix search      | O(m + k)           |
| `col LIKE '%substr%'`       | Rabin-Karp full scan    | O(n·(N+M)) avg     |
| `col LIKE '%suffix'`        | Rabin-Karp suffix check | O(n·M)             |
| `col OP val` (non-PK)       | Full scan + predicate   | O(n)               |
| `ORDER BY col INT`          | Red-Black Tree inorder  | O(n log n)         |

Where n = rows in table, k = result set size, m = pattern length, M = suffix length.

## Document Index

| File | Covers |
|------|--------|
| [bplus_tree.md](bplus_tree.md) | B+ Tree — primary-key index |
| [red_black_tree.md](red_black_tree.md) | Red-Black Tree — ORDER BY sort |
| [trie.md](trie.md) | Trie — prefix search & catalog |
| [bloom_filter.md](bloom_filter.md) | Bloom Filter — existence gate |
| [rabin_karp.md](rabin_karp.md) | Rabin-Karp — substring search |
