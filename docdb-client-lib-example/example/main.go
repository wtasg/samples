// example/main.go — demonstrates the DocDB Go client library.
//
// Start the server first:
//
//	cd docdb-example && go run ./cmd/server --addr :60013 --data ./data
//
// Then run this example:
//
//	cd docdb-client-lib-example && go run ./example
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/docdb/client/docdb"
)

func main() {
	addr := flag.String("addr", "", "server address (default: DOCDB_ADDR env, then http://localhost:60013)")
	flag.Parse()

	serverAddr := *addr
	if serverAddr == "" {
		serverAddr = os.Getenv("DOCDB_ADDR")
	}
	if serverAddr == "" {
		serverAddr = "http://localhost:60013"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// ── Connect ──
	c := docdb.NewClient(serverAddr)
	defer c.Close()

	// Ping
	info, err := c.Ping(ctx)
	if err != nil {
		log.Fatalf("ping %s: %v\n\nIs the server running?\n  cd docdb-example && go run ./cmd/server", serverAddr, err)
	}
	fmt.Printf("Connected to %s (%s) at %s\n\n", info.Version, info.GoVersion, serverAddr)

	// ── Collection operations ──
	fmt.Println("── CREATE COLLECTION ──────────────────────────")

	// Drop if exists (ignore error).
	_ = c.DropCollection(ctx, "employees")

	err = c.CreateCollection(ctx, "employees")
	must(err, "create collection")
	fmt.Println("Collection 'employees' created.")

	// ── Insert documents ──
	fmt.Println("\n── INSERT ─────────────────────────────────────")

	employees := []docdb.Doc{
		{"_id": "e1", "name": "Alice", "dept": "Engineering", "salary": 95000},
		{"_id": "e2", "name": "Bob", "dept": "Engineering", "salary": 88000},
		{"_id": "e3", "name": "Carol", "dept": "Design", "salary": 82000},
		{"_id": "e4", "name": "Dave", "dept": "Engineering", "salary": 91000},
		{"_id": "e5", "name": "Eve", "dept": "Design", "salary": 78000},
		{"_id": "e6", "name": "Frank", "dept": "Marketing", "salary": 72000},
		{"_id": "e7", "name": "Grace", "dept": "Marketing", "salary": 75000},
		{"_id": "e8", "name": "Heidi", "dept": "Engineering", "salary": 105000},
	}

	col := c.Collection("employees")
	for _, emp := range employees {
		must(col.Insert(ctx, emp), "insert")
	}
	fmt.Printf("Inserted %d employees.\n", len(employees))

	// ── Find all ──
	fmt.Println("\n── FIND ALL ───────────────────────────────────")
	result, err := col.Find(ctx)
	must(err, "find all")
	printResult(result)

	// ── Get by ID ──
	fmt.Println("\n── GET BY ID (Hash Map lookup) ────────────────")
	doc, err := col.Filter(docdb.M{"_id": "e3"}).Find(ctx)
	must(err, "get by ID")
	if len(doc.Docs) > 0 {
		emp := doc.Docs[0]
		fmt.Printf("  _id=e3 → name=%q  dept=%q  salary=%v\n", emp.Text("name"), emp.Text("dept"), emp.Int("salary"))
	}

	// ── Find using $gt ──
	fmt.Println("\n── FIND WITH $gt (salary > 90000) ─────────────")
	result, err = col.Filter(docdb.M{"salary": docdb.M{"$gt": 90000}}).Find(ctx)
	must(err, "find with $gt")
	printResult(result)

	// ── Inverted Index prefix search ──
	fmt.Println("\n── FIND WITH $prefix (dept starts with 'Eng') ──")
	result, err = col.Filter(docdb.M{"dept": docdb.M{"$prefix": "Eng"}}).Find(ctx)
	must(err, "find with $prefix")
	printResult(result)

	// ── Skip List SORT ──
	fmt.Println("\n── SORT BY salary DESC (Skip List) ────────────")
	result, err = col.Sort("salary", -1).Find(ctx)
	must(err, "sort descending")
	printResult(result)

	// ── Streaming query ──
	fmt.Println("\n── STREAMING FIND (gRPC server-streaming) ─────")
	docCount := 0
	err = col.Filter(docdb.M{"salary": docdb.M{"$gt": 85000}}).FindStream(ctx, func(d docdb.Doc) error {
		docCount++
		fmt.Printf("  [stream doc %d] name=%-10s  salary=%v\n", docCount, d.Text("name"), d.Int("salary"))
		return nil
	})
	must(err, "stream find")

	// ── UPDATE ──
	fmt.Println("\n── UPDATE ─────────────────────────────────────")
	n, err := col.Filter(docdb.M{"_id": "e6"}).Update(ctx, docdb.M{"$set": docdb.M{"salary": 76000}})
	must(err, "update")
	fmt.Printf("Updated %d document(s).\n", n)

	// ── DELETE ──
	fmt.Println("\n── DELETE ─────────────────────────────────────")
	n, err = col.Filter(docdb.M{"dept": "Marketing"}).Delete(ctx)
	must(err, "delete")
	fmt.Printf("Deleted %d Marketing employee(s).\n", n)

	// ── DescribeCollection ──
	fmt.Println("\n── DESCRIBE COLLECTION ────────────────────────")
	meta, err := c.DescribeCollection(ctx, "employees")
	must(err, "describe collection")
	fmt.Printf("Collection: %s\n  Documents: %d\n  Size:      %d bytes\n", meta.Name, meta.DocCount, meta.Size)

	// ── ListCollections ──
	fmt.Println("\n── LIST COLLECTIONS ───────────────────────────")
	collections, err := c.ListCollections(ctx)
	must(err, "list collections")
	for _, colName := range collections {
		fmt.Printf("  - %s\n", colName)
	}

	// ── Cleanup ──
	fmt.Println("\n── DROP COLLECTION ────────────────────────────")
	must(c.DropCollection(ctx, "employees"), "drop collection")
	fmt.Println("Done. employees collection dropped.")
}

func must(err error, op string) {
	if err != nil {
		log.Fatalf("%s: %v", op, err)
	}
}

func printResult(r *docdb.Result) {
	if r == nil {
		return
	}
	if r.Message != "" {
		fmt.Println(" ", r.Message)
	}
	if len(r.Docs) == 0 {
		return
	}
	for _, doc := range r.Docs {
		b, err := json.Marshal(doc)
		if err != nil {
			fmt.Printf("  %+v\n", doc)
		} else {
			fmt.Printf("  %s\n", string(b))
		}
	}
	fmt.Printf("  (%d documents)\n", len(r.Docs))
}
