package grpcserver

import (
	"context"
	"os"
	"testing"

	"connectrpc.com/connect"

	toydbv1 "rdbms/gen/toydb/v1"
	"rdbms/internal/engine"
)

func TestGrpcServer(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "toydb_grpc_server_test")
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
	pingResp, err := svc.Ping(ctx, connect.NewRequest(&toydbv1.PingRequest{}))
	if err != nil {
		t.Fatalf("Ping failed: %v", err)
	}
	if pingResp.Msg.Version != version {
		t.Errorf("Expected version %q, got %q", version, pingResp.Msg.Version)
	}

	// 2. Execute CREATE TABLE
	createReq := &toydbv1.ExecuteRequest{
		Sql: "CREATE TABLE items (id INT, name TEXT, weight FLOAT);",
	}
	execResp, err := svc.Execute(ctx, connect.NewRequest(createReq))
	if err != nil {
		t.Fatalf("Execute CREATE failed: %v", err)
	}
	if !execResp.Msg.Ok {
		t.Errorf("Execute CREATE returned error: %s", execResp.Msg.Error)
	}

	// 3. ListTables
	listResp, err := svc.ListTables(ctx, connect.NewRequest(&toydbv1.ListTablesRequest{}))
	if err != nil {
		t.Fatalf("ListTables failed: %v", err)
	}
	if len(listResp.Msg.Tables) != 1 || listResp.Msg.Tables[0] != "items" {
		t.Errorf("Unexpected ListTables response: %v", listResp.Msg.Tables)
	}

	// 4. DescribeTable
	descResp, err := svc.DescribeTable(ctx, connect.NewRequest(&toydbv1.DescribeTableRequest{Table: "items"}))
	if err != nil {
		t.Fatalf("DescribeTable failed: %v", err)
	}
	if descResp.Msg.Table != "items" || len(descResp.Msg.Columns) != 3 {
		t.Errorf("Unexpected DescribeTable response: %+v", descResp.Msg)
	}

	// 5. Execute INSERT
	insertReq := &toydbv1.ExecuteRequest{
		Sql: "INSERT INTO items VALUES (1, 'Shield', 5.5);",
	}
	execResp, err = svc.Execute(ctx, connect.NewRequest(insertReq))
	if err != nil {
		t.Fatalf("Execute INSERT failed: %v", err)
	}
	if !execResp.Msg.Ok || execResp.Msg.Message != "1 row inserted." {
		t.Errorf("Unexpected INSERT response: %+v", execResp.Msg)
	}

	// 6. Execute SELECT
	selectReq := &toydbv1.ExecuteRequest{
		Sql: "SELECT * FROM items WHERE id = 1;",
	}
	execResp, err = svc.Execute(ctx, connect.NewRequest(selectReq))
	if err != nil {
		t.Fatalf("Execute SELECT failed: %v", err)
	}
	if !execResp.Msg.Ok || execResp.Msg.Result == nil {
		t.Fatalf("SELECT failed: %+v", execResp.Msg)
	}
	rows := execResp.Msg.Result.Rows
	if len(rows) != 1 {
		t.Fatalf("Expected 1 row, got %d", len(rows))
	}
	val := rows[0].Values["name"]
	if val.GetTextVal() != "Shield" {
		t.Errorf("Expected 'Shield', got %q", val.GetTextVal())
	}
}
