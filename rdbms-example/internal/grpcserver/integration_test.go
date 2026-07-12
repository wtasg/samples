//go:build integration
// +build integration

package grpcserver

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/toydb/client/toydb"
	"rdbms/gen/toydb/v1/toydbv1connect"
	"rdbms/internal/engine"
)

func TestClientServerIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Setup temp data dir for ToyDB
	tempDir, err := os.MkdirTemp("", "toydb_integration_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 2. Start query executor
	ex, err := engine.NewExecutor(tempDir)
	if err != nil {
		t.Fatalf("failed to start executor: %v", err)
	}
	defer ex.Close()

	// 3. Create ConnectRPC service and http handler
	svc := New(ex, tempDir)
	_, handler := toydbv1connect.NewToyDBHandler(svc)

	// 4. Start local test HTTP server
	server := httptest.NewServer(handler)
	defer server.Close()

	// 5. Connect client to test server
	c := toydb.NewClient(server.URL, toydb.WithHTTPClient(server.Client()))
	defer c.Close()

	// 6. Test Ping
	ping, err := c.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if ping.Version != "ToyDB 1.0" {
		t.Errorf("Expected version 'ToyDB 1.0', got %q", ping.Version)
	}

	// 7. Create Table
	err = c.CreateTable(ctx, "tasks", toydb.Schema{
		{Name: "id", Type: toydb.INT},
		{Name: "title", Type: toydb.TEXT},
		{Name: "priority", Type: toydb.INT},
	})
	if err != nil {
		t.Fatalf("CreateTable failed: %v", err)
	}

	// 8. Insert Rows
	tbl := c.Table("tasks")
	err = tbl.Insert(ctx, toydb.Row{"id": int64(1), "title": "Buy groceries", "priority": int64(3)})
	if err != nil {
		t.Errorf("Insert 1 failed: %v", err)
	}
	err = tbl.Insert(ctx, toydb.Row{"id": int64(2), "title": "Clean house", "priority": int64(1)})
	if err != nil {
		t.Errorf("Insert 2 failed: %v", err)
	}
	err = tbl.Insert(ctx, toydb.Row{"id": int64(3), "title": "Pay bills", "priority": int64(2)})
	if err != nil {
		t.Errorf("Insert 3 failed: %v", err)
	}

	// 9. Fetch by PK
	row, err := tbl.Get(ctx, 2)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if row.Text("title") != "Clean house" || row.Int("priority") != 1 {
		t.Errorf("Unexpected row content: %+v", row)
	}

	// 10. Query (Select) with Range
	res, err := tbl.Where("id BETWEEN 2 AND 3").Select(ctx)
	if err != nil {
		t.Fatalf("Select range failed: %v", err)
	}
	if len(res.Rows) != 2 {
		t.Errorf("Expected 2 rows, got %d", len(res.Rows))
	}

	// 11. Prefix Query
	res, err = tbl.Where("title LIKE 'Buy%'").Select(ctx)
	if err != nil {
		t.Fatalf("Select prefix failed: %v", err)
	}
	if len(res.Rows) != 1 || res.Rows[0].Text("title") != "Buy groceries" {
		t.Errorf("Expected 'Buy groceries', got: %+v", res.Rows)
	}

	// 12. Streaming Query
	streamCount := 0
	err = tbl.OrderBy("priority").SelectStream(ctx, func(cols []string, r toydb.Row) error {
		streamCount++
		if streamCount == 1 && r.Text("title") != "Clean house" {
			t.Errorf("Expected first row to be priority 1 'Clean house', got %q", r.Text("title"))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("SelectStream failed: %v", err)
	}
	if streamCount != 3 {
		t.Errorf("Expected 3 streamed rows, got %d", streamCount)
	}

	// 13. Update
	n, err := tbl.Where("id = 3").Update(ctx, toydb.Row{"priority": int64(5)})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if n != 1 {
		t.Errorf("Expected 1 row updated, got %d", n)
	}
	row, _ = tbl.Get(ctx, 3)
	if row.Int("priority") != 5 {
		t.Errorf("Expected priority 5, got %d", row.Int("priority"))
	}

	// 14. Delete
	n, err = tbl.Where("id = 1").Delete(ctx)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if n != 1 {
		t.Errorf("Expected 1 row deleted, got %d", n)
	}
	row, _ = tbl.Get(ctx, 1)
	if row != nil {
		t.Errorf("Expected row to be deleted, got: %+v", row)
	}

	// 15. Drop Table
	err = c.DropTable(ctx, "tasks")
	if err != nil {
		t.Fatalf("DropTable failed: %v", err)
	}
}
