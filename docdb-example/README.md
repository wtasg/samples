# DocDB — A Toy NoSQL Document Database in Go

A deliberately small, single-user NoSQL document database engine that showcases
five core data structures used inside real document databases. Built in pure Go
with zero external runtime dependencies.

```
╔══════════════════════════════════════════════╗
║          DocDB — Toy NoSQL DB in Go          ║
║   LSMTree · RobinHood · SkipList · Inverted  ║
╚══════════════════════════════════════════════╝
```

## Quick Start

```bash
cd docdb-example
go run .          # starts the REPL (data stored in ./data/)
go run . mydb/    # use a custom data directory
```

## REPL Demo

```javascript
docdb> db.createCollection("products");
Collection "products" created.

docdb> db.products.insert({"name": "Apple", "price": 100, "tags": ["fruit"]});
Document inserted (id=4a8fd74c2bc0).

docdb> db.products.insert({"name": "Avocado", "price": 250, "tags": ["fruit", "healthy"]});
Document inserted (id=9b8ea74c2bc1).

docdb> db.products.insert({"name": "Banana", "price": 50, "tags": ["fruit"]});
Document inserted (id=1c8df74c2bc2).

// Find all documents
docdb> db.products.find({});
{
  "_id": "4a8fd74c2bc0",
  "name": "Apple",
  "price": 100,
  "tags": [
    "fruit"
  ]
}
...

// Find with greater than operator
docdb> db.products.find({"price": {"$gt": 50}});

// Find with prefix operator
docdb> db.products.find({"name": {"$prefix": "Av"}});

// Find with substring operator
docdb> db.products.find({"name": {"$contains": "pp"}});

// Find and sort descending
docdb> db.products.find({}).sort({"price": -1});

// Update and Delete
docdb> db.products.update({"_id": "4a8fd74c2bc0"}, {"$set": {"price": 120}});
docdb> db.products.delete({"_id": "1c8df74c2bc2"});

// Cleanup
docdb> db.dropCollection("products");
docdb> \q
```

## Query Language API

| Operator | Description | Usage |
|---|---|---|
| `$eq` | Field equality | `{"name": {"$eq": "Apple"}}` |
| `$ne` | Field inequality | `{"name": {"$ne": "Apple"}}` |
| `$gt` / `$gte` | Greater than (or equal) | `{"price": {"$gt": 50}}` |
| `$lt` / `$lte` | Less than (or equal) | `{"price": {"$lt": 100}}` |
| `$prefix` | Text prefix search | `{"name": {"$prefix": "Ap"}}` |
| `$contains` | Substring search | `{"name": {"$contains": "ppl"}}` |

### Multi-Statement Blocks

Both the CLI and gRPC server support executing multiple commands separated by semicolons (`;`) in a single command string. This is useful for executing multiple `insert` statements or scripts (such as those populated by UI templates). The engine splits them on semicolons at depth 0, executing each sequentially, and returning the aggregated outputs.

## Data Structures

| Structure | File | Role |
|---|---|---|
| **LSM Tree** | `internal/ds/lsmtree.go` | Primary write-optimized storage engine: memtable + SST runs |
| **Hash Map** | `internal/ds/hashmap.go` | O(1) document ID index: `GET` by `_id` lookup |
| **Skip List** | `internal/ds/skiplist.go` | Ordered memtable for LSM Tree & sorted query builder |
| **Bloom Filter** | `internal/ds/bloom.go` | Per-SSTable existence gate: avoids unnecessary disk reads |
| **Inverted Index** | `internal/ds/inverted.go` | Secondary index on arbitrary fields & prefix/contains search |

## Query Dispatch

```text
GET by _id           → ①Hash Map O(1) → ②LSM Get (memtable → SST levels)
FIND by field=value  → Inverted Index lookup → Hash Map → documents
FIND with $prefix    → Inverted Index prefix search → documents
FIND with $contains  → Full collection scan with string matching
FIND with $gt/$lt    → Full scan with typed predicate
SORT                 → Skip List ordered insertion → in-order traversal
```

## Architecture

```text
main.go        REPL — reads commands, calls Execute, prints pretty JSON
parser/        Lexer + parser → AST for DocDB commands
engine/
  collection.go  CRUD on collections using LSM Tree, Hash Map, Inverted Index
  executor.go    Resolves collection name, dispatches queries to engine paths
catalog/       Collection registry, persisted to catalog.json
storage/       On-disk document store (.docs files)
internal/ds/   Pure Go data-structure implementations + tests
```

## Running Tests

```bash
go test ./internal/... -v    # Runs unit tests for all internal packages
go test -tags=integration ./internal/grpcserver/... -v # Runs integration tests
```

## Persistence

Data is stored in the `data/` directory:

```text
data/
├── catalog.json       # registry of collections
└── <collection>.docs  # newline-delimited JSON document store (append-only)
```

The LSM Tree, Hash Map, and Inverted Index are all **rebuilt from `.docs` at startup** — the `.docs` file is the single source of truth.

---

## gRPC / Connect-RPC Server

DocDB exposes all operations over a **binary gRPC / Connect-RPC** service.

### Start the server

```bash
# Default: port 60013, data/ directory
go run ./cmd/server

# Custom address and data directory
go run ./cmd/server --addr :60013 --data ./mydata
```

The server speaks Connect, gRPC, and gRPC-Web on the same port.

### Go Client Library

See [`docdb-client-lib-example/`](../docdb-client-lib-example/) for the client library.

```go
c := docdb.NewClient("http://localhost:60013")
docs, err := c.Collection("products").Filter(docdb.M{"price": docdb.M{"$gt": 50}}).Find(ctx)
```
