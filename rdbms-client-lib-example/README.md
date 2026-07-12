# ToyDB Go Client Library

A Go client library for [ToyDB](../rdbms-example/) — the toy RDBMS that
demonstrates B+ Trees, Red-Black Trees, Tries, Bloom Filters, and Rabin-Karp.

## Protocol

The library communicates over **Connect-RPC** (binary protobuf), which is
wire-compatible with:
- Standard gRPC clients (HTTP/2 + protobuf binary)
- gRPC-Web clients (HTTP/1.1)
- Connect protocol clients (HTTP/1.1 or HTTP/2, JSON or binary)

## Installation

```bash
go get github.com/toydb/client
```

## Quick Start

```go
import "github.com/toydb/client/toydb"

// Connect (default: http://localhost:9090)
c := toydb.NewClient("http://your-server:9090")
defer c.Close()

// Ping
info, err := c.Ping(ctx)

// Create a table (first column must be INT — it's the primary key)
err = c.CreateTable(ctx, "users", toydb.Schema{
    {Name: "id",   Type: toydb.INT},
    {Name: "name", Type: toydb.TEXT},
    {Name: "age",  Type: toydb.INT},
})

// Insert rows
tbl := c.Table("users")
err = tbl.Insert(ctx, toydb.Row{"id": 1, "name": "Alice", "age": 30})
err = tbl.Insert(ctx, toydb.Row{"id": 2, "name": "Bob",   "age": 25})

// ── Queries — each uses a different data structure on the server ─────────────

// B+ Tree point lookup (Bloom Filter → B+ Tree → Pager)
row, err := tbl.Get(ctx, 1)              // id = 1

// B+ Tree range scan
rows, err := tbl.Where("id BETWEEN 1 AND 10").Select(ctx)

// Trie prefix search (LIKE 'prefix%')
rows, err = tbl.Where("name LIKE 'Al%'").Select(ctx)

// Rabin-Karp substring search (LIKE '%substr%')
rows, err = tbl.Where("name LIKE '%ice%'").Select(ctx)

// Red-Black Tree ORDER BY
rows, err = tbl.OrderBy("age").Select(ctx)
rows, err = tbl.OrderBy("age").Desc().Select(ctx)

// General comparison
rows, err = tbl.Where("age > 25").Select(ctx)

// ── Streaming (server-streaming gRPC) ────────────────────────────────────────
err = tbl.Where("age > 20").SelectStream(ctx, func(cols []string, row toydb.Row) error {
    fmt.Printf("name=%s  age=%d\n", row.Text("name"), row.Int("age"))
    return nil
})

// ── Update and Delete ─────────────────────────────────────────────────────────
n, err := tbl.Where("id = 1").Update(ctx, toydb.Row{"age": 31})
n, err  = tbl.Where("age < 18").Delete(ctx)

// ── Schema introspection ──────────────────────────────────────────────────────
tables, err := c.ListTables(ctx)
schema, err := c.DescribeTable(ctx, "users")
err          = c.DropTable(ctx, "users")

// ── Raw SQL ────────────────────────────────────────────────────────────────────
result, err := c.Execute(ctx, "SELECT * FROM users WHERE age > 25")
result, err  = c.Query(ctx,   "SELECT * FROM users ORDER BY age DESC")
```

## API Reference

### `Client`

| Method | Description |
|--------|-------------|
| `NewClient(addr, opts...)` | Create a client |
| `Ping(ctx)` | Health check + version |
| `Execute(ctx, sql)` | Run any SQL (DDL or DML) |
| `Query(ctx, sql)` | SELECT via streaming RPC → buffered `*Result` |
| `QueryStream(ctx, sql, fn)` | SELECT via streaming RPC → row-by-row callback |
| `CreateTable(ctx, name, schema)` | Create a table |
| `DropTable(ctx, name)` | Drop a table |
| `ListTables(ctx)` | List all table names |
| `DescribeTable(ctx, name)` | Get column schema |
| `Table(name)` | Return a `TableQuery` builder |

### `TableQuery` (fluent builder)

| Method | Description |
|--------|-------------|
| `.Where(expr)` | Set WHERE clause (raw SQL expression) |
| `.OrderBy(col)` | Set ORDER BY column |
| `.Desc()` | Add DESC to ORDER BY |
| `.Columns(cols...)` | Restrict SELECT to these columns |
| `.Select(ctx)` | Execute SELECT → `*Result` |
| `.SelectStream(ctx, fn)` | Execute SELECT → streaming |
| `.Get(ctx, pk)` | Fetch one row by primary key |
| `.Insert(ctx, row)` | INSERT a row |
| `.Update(ctx, updates)` | UPDATE matching rows |
| `.Delete(ctx)` | DELETE matching rows |

### `Row` helper methods

```go
row.Int("column")    // → int64
row.Float("column")  // → float64
row.Text("column")   // → string
row.Bool("column")   // → bool
row.IsNull("column") // → bool
```

### Connection options

```go
// Use gRPC wire protocol (for non-Connect gRPC servers)
c := toydb.NewClient(addr, toydb.WithGRPC())

// Custom HTTP client (e.g. with TLS, timeouts)
c := toydb.NewClient(addr, toydb.WithHTTPClient(myHTTPClient))
```

## Start the Server

```bash
cd rdbms-example

# Build and run the gRPC server
go run ./cmd/server --addr :9090 --data ./data

# Or build a binary
go build -o toydb-server ./cmd/server
./toydb-server --addr :9090 --data ./data
```

## Run the Example

```bash
cd rdbms-client-lib-example

# Default: connects to localhost:9090
go run ./example

# Custom address
go run ./example --addr http://myserver:9090
TOYDB_ADDR=http://myserver:9090 go run ./example
```

## Project Layout

```
rdbms-client-lib-example/
├── go.mod
├── README.md
├── toydb/
│   ├── client.go    Client + raw SQL methods + schema operations
│   ├── table.go     TableQuery fluent builder (WHERE/ORDER BY/CRUD)
│   └── types.go     Row, Schema, Column, Result, ColumnType
├── gen/
│   └── toydb/v1/
│       ├── toydb.pb.go              generated protobuf types
│       └── toydbv1connect/
│           └── toydb.connect.go     generated Connect-RPC client/server
└── example/
    └── main.go      Full end-to-end demo (CRUD + streaming + schema ops)
```

## Proto definition

The service is defined in [`rdbms-example/proto/toydb.proto`](../rdbms-example/proto/toydb.proto).
Regenerate with:

```bash
cd rdbms-example
protoc \
  --go_out=../rdbms-client-lib-example \
  --go_opt=paths=source_relative \
  --go_opt=Mproto/toydb.proto=github.com/toydb/client/gen/toydb/v1 \
  --connect-go_out=../rdbms-client-lib-example \
  --connect-go_opt=paths=source_relative \
  --connect-go_opt=Mproto/toydb.proto=github.com/toydb/client/gen/toydb/v1 \
  proto/toydb.proto

# Move generated files
mv proto/toydb.pb.go gen/toydb/v1/toydb.pb.go
mv proto/toydbv1connect/toydb.connect.go gen/toydb/v1/toydbv1connect/toydb.connect.go
rmdir proto/toydbv1connect proto
```
