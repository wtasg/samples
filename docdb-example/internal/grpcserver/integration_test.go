//go:build integration
// +build integration

package grpcserver

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/docdb/client/docdb"
	"github.com/docdb/client/gen/docdb/v1/docdbv1connect"
	"docdb/internal/engine"
)

func TestClientServerIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Setup temp data dir for DocDB
	tempDir, err := os.MkdirTemp("", "docdb_integration_test")
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
	_, handler := docdbv1connect.NewDocDBHandler(svc)

	// 4. Start local test HTTP server
	server := httptest.NewServer(handler)
	defer server.Close()

	// 5. Connect client to test server
	c := docdb.NewClient(server.URL, docdb.WithHTTPClient(server.Client()))
	defer c.Close()

	// 6. Test Ping
	ping, err := c.Ping(ctx)
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if ping.Version != "DocDB 1.0" {
		t.Errorf("Expected version 'DocDB 1.0', got %q", ping.Version)
	}

	// 7. Create Collection
	err = c.CreateCollection(ctx, "tasks")
	if err != nil {
		t.Fatalf("CreateCollection failed: %v", err)
	}

	// 8. Insert Documents
	col := c.Collection("tasks")
	err = col.Insert(ctx, docdb.Doc{"_id": "t1", "title": "Buy groceries", "priority": int64(3)})
	if err != nil {
		t.Errorf("Insert 1 failed: %v", err)
	}
	err = col.Insert(ctx, docdb.Doc{"_id": "t2", "title": "Clean house", "priority": int64(1)})
	if err != nil {
		t.Errorf("Insert 2 failed: %v", err)
	}
	err = col.Insert(ctx, docdb.Doc{"_id": "t3", "title": "Pay bills", "priority": int64(2)})
	if err != nil {
		t.Errorf("Insert 3 failed: %v", err)
	}

	// 9. Fetch by ID
	doc, err := col.Filter(docdb.M{"_id": "t2"}).Find(ctx)
	if err != nil {
		t.Fatalf("Find by ID failed: %v", err)
	}
	if len(doc.Docs) != 1 || doc.Docs[0].Text("title") != "Clean house" || doc.Docs[0].Int("priority") != 1 {
		t.Errorf("Unexpected document content: %+v", doc.Docs)
	}

	// 10. Query (Find) with Operator ($gt)
	res, err := col.Filter(docdb.M{"priority": docdb.M{"$gt": 1}}).Find(ctx)
	if err != nil {
		t.Fatalf("Find with $gt failed: %v", err)
	}
	if len(res.Docs) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(res.Docs))
	}

	// 11. Prefix Query
	res, err = col.Filter(docdb.M{"title": docdb.M{"$prefix": "Buy"}}).Find(ctx)
	if err != nil {
		t.Fatalf("Find with $prefix failed: %v", err)
	}
	if len(res.Docs) != 1 || res.Docs[0].Text("title") != "Buy groceries" {
		t.Errorf("Expected 'Buy groceries', got: %+v", res.Docs)
	}

	// 12. Streaming Query + Sort
	streamCount := 0
	err = col.Sort("priority", 1).FindStream(ctx, func(d docdb.Doc) error {
		streamCount++
		if streamCount == 1 && d.Text("title") != "Clean house" {
			t.Errorf("Expected first document to be priority 1 'Clean house', got %q", d.Text("title"))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("FindStream failed: %v", err)
	}
	if streamCount != 3 {
		t.Errorf("Expected 3 streamed documents, got %d", streamCount)
	}

	// 13. Update
	n, err := col.Filter(docdb.M{"_id": "t3"}).Update(ctx, docdb.M{"$set": docdb.M{"priority": int64(5)}})
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if n != 1 {
		t.Errorf("Expected 1 document updated, got %d", n)
	}
	updatedRes, _ := col.Filter(docdb.M{"_id": "t3"}).Find(ctx)
	if len(updatedRes.Docs) != 1 || updatedRes.Docs[0].Int("priority") != 5 {
		t.Errorf("Expected priority 5, got %d", updatedRes.Docs[0].Int("priority"))
	}

	// 14. Delete
	n, err = col.Filter(docdb.M{"_id": "t1"}).Delete(ctx)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if n != 1 {
		t.Errorf("Expected 1 document deleted, got %d", n)
	}
	deletedRes, _ := col.Filter(docdb.M{"_id": "t1"}).Find(ctx)
	if len(deletedRes.Docs) != 0 {
		t.Errorf("Expected document to be deleted, got: %+v", deletedRes.Docs)
	}

	// 15. Drop Collection
	err = c.DropCollection(ctx, "tasks")
	if err != nil {
		t.Fatalf("DropCollection failed: %v", err)
	}
}
