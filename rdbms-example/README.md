# ToyDB — A Toy RDBMS in Go

A deliberately small, single-user relational database engine that showcases
five core data structures used inside real databases. Built in pure Go with
zero external dependencies.

```
╔══════════════════════════════════════════════╗
║            ToyDB — Toy RDBMS in Go           ║
║  B+Tree · RedBlack · Trie · Bloom · R-Karp   ║
╚══════════════════════════════════════════════╝
```

## Quick Start

```bash
cd rdbms-example
go run .          # starts the REPL (data stored in ./data/)
go run . mydb/    # use a custom data directory
```

## REPL Demo

```sql
toydb> CREATE TABLE products (id INT, name TEXT, price INT);
Table "products" created.

toydb> INSERT INTO products VALUES (1, 'Apple', 100);
1 row inserted.
toydb> INSERT INTO products VALUES (2, 'Avocado', 250);
1 row inserted.
toydb> INSERT INTO products VALUES (3, 'Banana', 50);
1 row inserted.
toydb> INSERT INTO products VALUES (4, 'Blueberry', 300);
1 row inserted.

-- Full table scan
toydb> SELECT * FROM products;
├────┼───────────┼───────┤
│ id │ name      │ price │
├────┼───────────┼───────┤
│ 1  │ Apple     │ 100   │
│ 2  │ Avocado   │ 250   │
│ 3  │ Banana    │ 50    │
│ 4  │ Blueberry │ 300   │
├────┼───────────┼───────┤

-- B+ Tree point lookup (Bloom Filter → B+ Tree → Pager)
toydb> SELECT * FROM products WHERE id = 2;

-- B+ Tree range scan
toydb> SELECT * FROM products WHERE id BETWEEN 2 AND 4;

-- Trie prefix search (LIKE 'prefix%')
toydb> SELECT * FROM products WHERE name LIKE 'A%';

-- Rabin-Karp substring search (LIKE '%substr%')
toydb> SELECT * FROM products WHERE name LIKE '%an%';

-- Rabin-Karp suffix search (LIKE '%suffix')
toydb> SELECT * FROM products WHERE name LIKE '%erry';

-- Red-Black Tree ORDER BY (in-order traversal)
toydb> SELECT * FROM products ORDER BY price DESC;

-- Update and Delete
toydb> UPDATE products SET price = 120 WHERE id = 1;
toydb> DELETE FROM products WHERE id = 3;

-- Cleanup
toydb> DROP TABLE products;
toydb> \q
```

## SQL Subset

| Statement | Syntax |
|-----------|--------|
| Create    | `CREATE TABLE t (col TYPE, ...)` |
| Insert    | `INSERT INTO t VALUES (v1, v2, ...)` |
| Select    | `SELECT */cols FROM t [WHERE expr] [ORDER BY col [DESC]]` |
| Update    | `UPDATE t SET col=val,... WHERE expr` |
| Delete    | `DELETE FROM t WHERE expr` |
| Drop      | `DROP TABLE t` |

**WHERE operators**: `=`, `!=`, `<`, `>`, `<=`, `>=`, `BETWEEN lo AND hi`, `LIKE 'pat'`

**Column types**: `INT`, `TEXT`, `FLOAT`, `BOOL`

> The **first column must be INT** — it is the primary key used by all indexes.

### Multi-Statement Blocks

Both the CLI and gRPC server support executing multiple SQL queries separated by semicolons (`;`) in a single query string. This is useful for executing multiple `INSERT` statements or scripts (such as those populated by UI templates). The engine splits them on semicolons at depth 0, executing each sequentially, and returning the aggregated outputs.

## Data Structures

| Structure | File | Role |
|-----------|------|------|
| **B+ Tree** | `internal/ds/bptree.go` | Primary-key index: PK→rowID, range scans |
| **Red-Black Tree** | `internal/ds/rbtree.go` | ORDER BY: sorted in-memory result set |
| **Trie** | `internal/ds/trie.go` | LIKE 'prefix%' + schema catalog lookup |
| **Bloom Filter** | `internal/ds/bloom.go` | Existence gate — skip disk for absent keys |
| **Rabin-Karp** | `internal/ds/rabinkarp.go` | LIKE '%substr%' rolling-hash search |

## Query Dispatch

```
WHERE id = 42         → ①Bloom (O(1)) → ②B+Tree (O(log n)) → ③Pager (O(1))
WHERE id BETWEEN 1 10 → B+Tree range scan → Pager   O(log n + k)
WHERE name LIKE 'A%'  → Trie.PrefixSearch() → Pager  O(m + k)
WHERE name LIKE '%an%'→ Rabin-Karp full scan          O(n·(N+M)) avg
ORDER BY price DESC   → Red-Black Tree InOrderDesc()  O(n log n)
```

## Architecture

```
main.go        REPL — reads SQL, calls Execute, pretty-prints results
parser/        Tokenizer + recursive-descent parser → AST (no deps)
engine/
  executor.go  Walks AST, dispatches to optimal data-structure path
  table.go     Per-table CRUD using all 5 data structures + pager
catalog/       Schema store (uses Trie internally), persisted to catalog.json
storage/       Page/row file I/O — newline-delimited JSON .rows files
internal/ds/   Pure data-structure implementations + tests
```

## Documentation

See `docs/` for in-depth explanations of each data structure:

- [Overview](docs/overview.md) — architecture diagram, query dispatch table
- [B+ Tree](docs/bplus_tree.md) — structure, splits, real-world use
- [Red-Black Tree](docs/red_black_tree.md) — rotations, invariants, vs AVL
- [Trie](docs/trie.md) — prefix search, schema catalog
- [Bloom Filter](docs/bloom_filter.md) — false positives, optimal parameters
- [Rabin-Karp](docs/rabin_karp.md) — rolling hash, vs KMP, collision handling

## Running Tests

```bash
go test ./internal/ds/... -v    # 27 tests across all 5 data structures
go test ./...                   # full test suite
```

## Persistence

Data is stored in the `data/` directory (created automatically):

```
data/
├── catalog.json          # table schemas (JSON)
├── <table>.rows          # newline-delimited JSON rows (append-only)
```

The B+ Tree, Bloom Filter, and Trie are all **rebuilt from `.rows` at startup**
— the rows file is the single source of truth.

## Limitations (by design)

- Single user — no locking or transactions
- No secondary indexes on INT columns (only TEXT Trie secondary indexes)
- No SQL JOINs, GROUP BY, or aggregates
- Primary key must be the first INT column
- No WAL (write-ahead log) or crash recovery

---

## gRPC / Connect-RPC Server

ToyDB exposes all operations over a **binary gRPC / Connect-RPC** service.

### Start the server

```bash
# Default: port 9090, data/ directory
go run ./cmd/server

# Custom address and data directory
go run ./cmd/server --addr :9090 --data ./mydata
```

The server speaks **three protocols on the same port** (no separate config needed):

- **gRPC** — binary protobuf over HTTP/2 (compatible with any gRPC client)
- **Connect** — binary or JSON over HTTP/1.1 or HTTP/2
- **gRPC-Web** — binary protobuf over HTTP/1.1

### Proto service

Defined in [`proto/toydb.proto`](proto/toydb.proto):

```protobuf
service ToyDB {
  rpc Execute(ExecuteRequest) returns (ExecuteResponse);         // any SQL
  rpc Query(QueryRequest) returns (stream QueryStreamResponse);  // SELECT streaming
  rpc Ping(PingRequest) returns (PingResponse);
  rpc ListTables(ListTablesRequest) returns (ListTablesResponse);
  rpc DescribeTable(DescribeTableRequest) returns (DescribeTableResponse);
}
```

### Go client library

See [`rdbms-client-lib-example/`](../rdbms-client-lib-example/) for the full client library.

```go
c := toydb.NewClient("http://localhost:9090")
rows, err := c.Table("users").Where("name LIKE 'Al%'").Select(ctx)
```

### Regenerate proto code

```bash
cd rdbms-example
protoc \
  --go_out=. --go_opt=paths=source_relative \
  --connect-go_out=. --connect-go_opt=paths=source_relative \
  proto/toydb.proto
mv proto/toydb.pb.go gen/toydb/v1/toydb.pb.go
mv proto/toydbv1connect/toydb.connect.go gen/toydb/v1/toydbv1connect/toydb.connect.go
```
