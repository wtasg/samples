# DocDB Go Client Library

A Go client library for [DocDB](../docdb-example/) — the toy NoSQL document database
that demonstrates LSM Trees, Robin Hood Hashing, Skip Lists, Bloom Filters, and
Inverted Indexes.

## Installation

```bash
go get github.com/docdb/client
```

## Quick Start

```go
import "github.com/docdb/client/docdb"

// Connect (default: http://localhost:60013)
c := docdb.NewClient("http://your-server:60013")
defer c.Close()

// Ping
info, err := c.Ping(ctx)

// Create a collection
err = c.CreateCollection(ctx, "users")

// Insert a document
col := c.Collection("users")
err = col.Insert(ctx, docdb.Doc{"_id": "alice", "name": "Alice", "age": 30})

// Queries
res, err := col.Filter(docdb.M{"age": docdb.M{"$gt": 25}}).Find(ctx)

// Sorted Queries (Skip List sorting)
res, err = col.Sort("age", -1).Find(ctx)

// Streaming (server-streaming Connect/gRPC)
err = col.Filter(docdb.M{"age": docdb.M{"$gt": 20}}).FindStream(ctx, func(doc docdb.Doc) error {
    fmt.Printf("name=%s age=%d\n", doc.Text("name"), doc.Int("age"))
    return nil
})

// Update and Delete
n, err := col.Filter(docdb.M{"_id": "alice"}).Update(ctx, docdb.M{"$set": docdb.M{"age": 31}})
n, err  = col.Filter(docdb.M{"age": docdb.M{"$lt": 18}}).Delete(ctx)
```

## API Reference

### `Client`

| Method | Description |
|---|---|
| `NewClient(addr, opts...)` | Create a client |
| `Ping(ctx)` | Health check + version |
| `Execute(ctx, command)` | Run any DocDB command string |
| `Query(ctx, command)` | Execute find via streaming RPC → buffered `*Result` |
| `QueryStream(ctx, command, fn)` | Execute find via streaming RPC → callback |
| `CreateCollection(ctx, name)` | Create a collection |
| `DropCollection(ctx, name)` | Drop a collection |
| `ListCollections(ctx)` | List all collection names |
| `DescribeCollection(ctx, name)` | Get collection statistics |
| `Collection(name)` | Return a `CollectionQuery` builder |

### `CollectionQuery` (fluent builder)

| Method | Description |
|---|---|
| `.Filter(filter)` | Set query filter |
| `.Sort(field, order)` | Set sort configurations |
| `.Find(ctx)` | Execute find query → `*Result` |
| `.FindStream(ctx, fn)` | Execute find query → callback |
| `.Insert(ctx, doc)` | Insert a document |
| `.Update(ctx, updates)` | Update matching documents |
| `.Delete(ctx)` | Delete matching documents |

### `Doc` helper methods

```go
doc.Int("field")    // → int64
doc.Float("field")  // → float64
doc.Text("field")   // → string
doc.Bool("field")   // → bool
doc.IsNull("field") // → bool
doc.Array("field")  // → []any
```

## Run the Example

```bash
cd docdb-client-lib-example
go run ./example
```
