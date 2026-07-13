package grpcserver

import (
	"context"
	"os"
	"testing"

	"connectrpc.com/connect"

	docdbv1 "github.com/docdb/client/gen/docdb/v1"
	"docdb/internal/engine"
)

func TestGrpcServer(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "docdb_grpc_server_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ex, err := engine.NewExecutor(tempDir)
	if err != nil {
		t.Fatalf("NewExecutor failed: %v", err)
	}
	defer ex.Close()

	svc := New(ex, tempDir)
	ctx := context.Background()

	// 1. Ping
	pingResp, err := svc.Ping(ctx, connect.NewRequest(&docdbv1.PingRequest{}))
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if pingResp.Msg.Version != version {
		t.Errorf("Expected version %q, got %q", version, pingResp.Msg.Version)
	}

	// 2. Execute db.createCollection("items")
	createReq := &docdbv1.ExecuteRequest{
		Command: `db.createCollection("items")`,
	}
	execResp, err := svc.Execute(ctx, connect.NewRequest(createReq))
	if err != nil {
		t.Fatalf("Execute CREATE failed: %v", err)
	}
	if !execResp.Msg.Ok {
		t.Errorf("Execute CREATE returned error: %s", execResp.Msg.Error)
	}

	// 3. ListCollections
	listResp, err := svc.ListCollections(ctx, connect.NewRequest(&docdbv1.ListCollectionsRequest{}))
	if err != nil {
		t.Fatalf("ListCollections failed: %v", err)
	}
	if len(listResp.Msg.Collections) != 1 || listResp.Msg.Collections[0] != "items" {
		t.Errorf("Unexpected ListCollections response: %v", listResp.Msg.Collections)
	}

	// 4. DescribeCollection
	descResp, err := svc.DescribeCollection(ctx, connect.NewRequest(&docdbv1.DescribeCollectionRequest{Collection: "items"}))
	if err != nil {
		t.Fatalf("DescribeCollection failed: %v", err)
	}
	if descResp.Msg.Collection != "items" || descResp.Msg.DocCount != 0 {
		t.Errorf("Unexpected DescribeCollection response: %+v", descResp.Msg)
	}

	// 5. Execute INSERT
	insertReq := &docdbv1.ExecuteRequest{
		Command: `db.items.insert({"_id": "item1", "name": "Shield", "weight": 5.5})`,
	}
	execResp, err = svc.Execute(ctx, connect.NewRequest(insertReq))
	if err != nil {
		t.Fatalf("Execute INSERT failed: %v", err)
	}
	if !execResp.Msg.Ok {
		t.Errorf("Unexpected INSERT response: %+v", execResp.Msg)
	}

	// 6. Execute FIND (single document)
	findReq := &docdbv1.ExecuteRequest{
		Command: `db.items.find({"_id": "item1"})`,
	}
	execResp, err = svc.Execute(ctx, connect.NewRequest(findReq))
	if err != nil {
		t.Fatalf("Execute FIND failed: %v", err)
	}
	if !execResp.Msg.Ok || len(execResp.Msg.Docs) != 1 {
		t.Fatalf("FIND failed: %+v", execResp.Msg)
	}
	doc := execResp.Msg.Docs[0]
	val := doc.Fields["name"]
	if val.GetTextVal() != "Shield" {
		t.Errorf("Expected 'Shield', got %q", val.GetTextVal())
	}
}

func TestSplitStatements(t *testing.T) {
	input := `db.users.insert({"name": "Smith; John"}); db.users.find({});`
	stmts := SplitStatements(input)

	if len(stmts) != 2 {
		t.Fatalf("expected 2 statements, got %d: %v", len(stmts), stmts)
	}
	if stmts[0] != `db.users.insert({"name": "Smith; John"})` {
		t.Errorf("expected first statement to have name with semicolon, got %q", stmts[0])
	}
	if stmts[1] != `db.users.find({})` {
		t.Errorf("expected second statement to be find, got %q", stmts[1])
	}
}
