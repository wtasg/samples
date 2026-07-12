// example/main.go — demonstrates the ToyDB Go client library.
//
// Start the server first:
//
//	cd rdbms-example && go run ./cmd/server --addr :9090 --data ./data
//
// Then run this example:
//
//	cd rdbms-client-lib-example && go run ./example
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/toydb/client/toydb"
)

func main() {
	addr := flag.String("addr", "", "server address (default: TOYDB_ADDR env, then http://localhost:9090)")
	flag.Parse()

	serverAddr := *addr
	if serverAddr == "" {
		serverAddr = os.Getenv("TOYDB_ADDR")
	}
	if serverAddr == "" {
		serverAddr = "http://localhost:9090"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// ── Connect ───────────────────────────────────────────────────────────────
	c := toydb.NewClient(serverAddr)
	defer c.Close()

	// Ping
	info, err := c.Ping(ctx)
	if err != nil {
		log.Fatalf("ping %s: %v\n\nIs the server running?\n  cd rdbms-example && go run ./cmd/server", serverAddr, err)
	}
	fmt.Printf("Connected to %s (%s) at %s\n\n", info.Version, info.GoVersion, serverAddr)

	// ── Schema operations ──────────────────────────────────────────────────────
	fmt.Println("── CREATE TABLE ──────────────────────────────")

	// Drop if exists (ignore error).
	_ = c.DropTable(ctx, "employees")

	err = c.CreateTable(ctx, "employees", toydb.Schema{
		{Name: "id",     Type: toydb.INT},
		{Name: "name",   Type: toydb.TEXT},
		{Name: "dept",   Type: toydb.TEXT},
		{Name: "salary", Type: toydb.INT},
	})
	must(err, "create table")
	fmt.Println("Table 'employees' created.")

	// ── Insert rows ────────────────────────────────────────────────────────────
	fmt.Println("\n── INSERT ─────────────────────────────────────")

	employees := []toydb.Row{
		{"id": 1, "name": "Alice",   "dept": "Engineering", "salary": 95000},
		{"id": 2, "name": "Bob",     "dept": "Engineering", "salary": 88000},
		{"id": 3, "name": "Carol",   "dept": "Design",      "salary": 82000},
		{"id": 4, "name": "Dave",    "dept": "Engineering", "salary": 91000},
		{"id": 5, "name": "Eve",     "dept": "Design",      "salary": 78000},
		{"id": 6, "name": "Frank",   "dept": "Marketing",   "salary": 72000},
		{"id": 7, "name": "Grace",   "dept": "Marketing",   "salary": 75000},
		{"id": 8, "name": "Heidi",   "dept": "Engineering", "salary": 105000},
	}

	tbl := c.Table("employees")
	for _, emp := range employees {
		must(tbl.Insert(ctx, emp), "insert")
	}
	fmt.Printf("Inserted %d employees.\n", len(employees))

	// ── SELECT all ────────────────────────────────────────────────────────────
	fmt.Println("\n── SELECT * ───────────────────────────────────")
	result, err := tbl.Select(ctx)
	must(err, "select all")
	printResult(result)

	// ── B+ Tree point lookup ──────────────────────────────────────────────────
	fmt.Println("\n── GET BY PRIMARY KEY (B+ Tree lookup) ────────")
	row, err := tbl.Get(ctx, 3)
	must(err, "get by pk")
	if row != nil {
		fmt.Printf("  id=3 → name=%q  dept=%q  salary=%v\n", row.Text("name"), row.Text("dept"), row.Int("salary"))
	}

	// ── B+ Tree range scan ────────────────────────────────────────────────────
	fmt.Println("\n── BETWEEN (B+ Tree range scan) ───────────────")
	result, err = tbl.Where("id BETWEEN 3 AND 6").Select(ctx)
	must(err, "between")
	printResult(result)

	// ── Trie prefix search ────────────────────────────────────────────────────
	fmt.Println("\n── LIKE 'prefix%' (Trie search) ───────────────")
	result, err = tbl.Where("dept LIKE 'Eng%'").Select(ctx)
	must(err, "like prefix")
	printResult(result)

	// ── Rabin-Karp substring search ───────────────────────────────────────────
	fmt.Println("\n── LIKE *substring* (Rabin-Karp) ────────────")
	result, err = tbl.Where("name LIKE '%e%'").Select(ctx)
	must(err, "like substr")
	printResult(result)

	// ── Red-Black Tree ORDER BY ───────────────────────────────────────────────
	fmt.Println("\n── ORDER BY salary DESC (Red-Black Tree) ───────")
	result, err = tbl.OrderBy("salary").Desc().Select(ctx)
	must(err, "order by")
	printResult(result)

	// ── Streaming query ───────────────────────────────────────────────────────
	fmt.Println("\n── STREAMING SELECT (gRPC server-streaming) ────")
	rowCount := 0
	err = tbl.Where("salary > 85000").SelectStream(ctx, func(cols []string, row toydb.Row) error {
		rowCount++
		fmt.Printf("  [stream row %d] name=%-10s  salary=%v\n", rowCount, row.Text("name"), row.Int("salary"))
		return nil
	})
	must(err, "stream select")

	// ── UPDATE ────────────────────────────────────────────────────────────────
	fmt.Println("\n── UPDATE (direct gRPC Execute) ────────────────")
	n, err := tbl.Where("id = 6").Update(ctx, toydb.Row{"salary": 76000})
	must(err, "update")
	fmt.Printf("Updated %d row(s).\n", n)

	// ── DELETE ────────────────────────────────────────────────────────────────
	fmt.Println("\n── DELETE ──────────────────────────────────────")
	n, err = tbl.Where("dept LIKE '%Marketing%'").Delete(ctx)
	must(err, "delete")
	fmt.Printf("Deleted %d Marketing employee(s).\n", n)

	// ── DescribeTable ─────────────────────────────────────────────────────────
	fmt.Println("\n── DESCRIBE TABLE ──────────────────────────────")
	schema, err := c.DescribeTable(ctx, "employees")
	must(err, "describe table")
	fmt.Printf("Table: %s\n", schema.Name)
	for _, col := range schema.Columns {
		pk := ""
		if col.PrimaryKey {
			pk = " (PK)"
		}
		fmt.Printf("  %-10s %s%s\n", col.Name, col.Type, pk)
	}

	// ── ListTables ────────────────────────────────────────────────────────────
	fmt.Println("\n── LIST TABLES ─────────────────────────────────")
	tables, err := c.ListTables(ctx)
	must(err, "list tables")
	for _, t := range tables {
		fmt.Printf("  - %s\n", t)
	}

	// ── Cleanup ───────────────────────────────────────────────────────────────
	fmt.Println("\n── DROP TABLE ──────────────────────────────────")
	must(c.DropTable(ctx, "employees"), "drop table")
	fmt.Println("Done. employees table dropped.")
}

func must(err error, op string) {
	if err != nil {
		log.Fatalf("%s: %v", op, err)
	}
}

func printResult(r *toydb.Result) {
	if r == nil || len(r.Columns) == 0 {
		if r != nil && r.Message != "" {
			fmt.Println(" ", r.Message)
		}
		return
	}
	// Print a simple aligned table.
	widths := make([]int, len(r.Columns))
	for i, col := range r.Columns {
		widths[i] = len(col)
	}
	for _, row := range r.Rows {
		for i, col := range r.Columns {
			s := fmt.Sprintf("%v", row[col])
			if len(s) > widths[i] {
				widths[i] = len(s)
			}
		}
	}
	// Header
	fmt.Print("  ")
	for i, col := range r.Columns {
		fmt.Printf("%-*s  ", widths[i], col)
	}
	fmt.Println()
	// Separator
	fmt.Print("  ")
	for _, w := range widths {
		fmt.Print(strings.Repeat("-", w) + "  ")
	}
	fmt.Println()
	// Rows
	for _, row := range r.Rows {
		fmt.Print("  ")
		for i, col := range r.Columns {
			fmt.Printf("%-*v  ", widths[i], row[col])
		}
		fmt.Println()
	}
	fmt.Printf("  (%d rows)\n", len(r.Rows))
}
