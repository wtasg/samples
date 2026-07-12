// Package grpcserver implements the ToyDB Connect-RPC service.
//
// The service wraps engine.Executor and translates between protobuf messages
// and the engine's Row/Result types.  It supports:
//   - Execute  — any SQL statement (unary)
//   - Query    — SELECT with server-streaming (for large result sets)
//   - Ping     — health check
//   - ListTables / DescribeTable — schema introspection
package grpcserver

import (
	"context"
	"fmt"
	"runtime"

	"connectrpc.com/connect"

	toydbv1 "rdbms/gen/toydb/v1"
	"rdbms/gen/toydb/v1/toydbv1connect"
	"rdbms/internal/engine"
	"rdbms/internal/parser"
)

const version = "ToyDB 1.0"

// Service implements ToyDBHandler.
type Service struct {
	toydbv1connect.UnimplementedToyDBHandler
	ex      *engine.Executor
	dataDir string
}

// New returns a new Service backed by ex.
func New(ex *engine.Executor, dataDir string) *Service {
	return &Service{ex: ex, dataDir: dataDir}
}

// ── Execute ───────────────────────────────────────────────────────────────────

func (s *Service) Execute(
	_ context.Context,
	req *connect.Request[toydbv1.ExecuteRequest],
) (*connect.Response[toydbv1.ExecuteResponse], error) {

	stmt, err := parser.Parse(req.Msg.Sql)
	if err != nil {
		return connect.NewResponse(&toydbv1.ExecuteResponse{
			Ok:    false,
			Error: fmt.Sprintf("parse error: %v", err),
		}), nil
	}
	if stmt == nil {
		return connect.NewResponse(&toydbv1.ExecuteResponse{Ok: true}), nil
	}

	result, err := s.ex.Execute(stmt)
	if err != nil {
		return connect.NewResponse(&toydbv1.ExecuteResponse{
			Ok:    false,
			Error: err.Error(),
		}), nil
	}

	resp := &toydbv1.ExecuteResponse{
		Ok:      true,
		Message: result.Message,
	}
	if len(result.Columns) > 0 {
		resp.Result = encodeResultSet(result)
	}
	return connect.NewResponse(resp), nil
}

// ── Query (server-streaming) ──────────────────────────────────────────────────

func (s *Service) Query(
	_ context.Context,
	req *connect.Request[toydbv1.QueryRequest],
	stream *connect.ServerStream[toydbv1.QueryStreamResponse],
) error {

	stmt, err := parser.Parse(req.Msg.Sql)
	if err != nil {
		return stream.Send(&toydbv1.QueryStreamResponse{
			Error: fmt.Sprintf("parse error: %v", err),
		})
	}
	if stmt == nil {
		return nil
	}

	result, err := s.ex.Execute(stmt)
	if err != nil {
		return stream.Send(&toydbv1.QueryStreamResponse{Error: err.Error()})
	}
	if result == nil || len(result.Columns) == 0 {
		// DDL / DML — nothing to stream.
		return nil
	}

	// First message carries column names.
	first := true
	for _, row := range result.Rows {
		msg := &toydbv1.QueryStreamResponse{
			Row: encodeRow(row, result.Columns),
		}
		if first {
			msg.Columns = result.Columns
			first = false
		}
		if err := stream.Send(msg); err != nil {
			return err
		}
	}
	return nil
}

// ── Ping ──────────────────────────────────────────────────────────────────────

func (s *Service) Ping(
	_ context.Context,
	_ *connect.Request[toydbv1.PingRequest],
) (*connect.Response[toydbv1.PingResponse], error) {

	return connect.NewResponse(&toydbv1.PingResponse{
		Version:   version,
		GoVersion: runtime.Version(),
		DataDir:   s.dataDir,
	}), nil
}

// ── ListTables ────────────────────────────────────────────────────────────────

func (s *Service) ListTables(
	_ context.Context,
	_ *connect.Request[toydbv1.ListTablesRequest],
) (*connect.Response[toydbv1.ListTablesResponse], error) {

	tables := s.ex.TableNames()
	return connect.NewResponse(&toydbv1.ListTablesResponse{Tables: tables}), nil
}

// ── DescribeTable ─────────────────────────────────────────────────────────────

func (s *Service) DescribeTable(
	_ context.Context,
	req *connect.Request[toydbv1.DescribeTableRequest],
) (*connect.Response[toydbv1.DescribeTableResponse], error) {

	schema, err := s.ex.TableSchema(req.Msg.Table)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	cols := make([]*toydbv1.ColumnDef, len(schema.Columns))
	for i, c := range schema.Columns {
		cols[i] = &toydbv1.ColumnDef{
			Name:       c.Name,
			Type:       string(c.Type),
			PrimaryKey: i == 0,
		}
	}
	return connect.NewResponse(&toydbv1.DescribeTableResponse{
		Table:   schema.Name,
		Columns: cols,
	}), nil
}

// ── Encoding helpers ──────────────────────────────────────────────────────────

// encodeResultSet converts an engine.Result into a protobuf ResultSet.
func encodeResultSet(r *engine.Result) *toydbv1.ResultSet {
	rs := &toydbv1.ResultSet{
		Columns: r.Columns,
		Rows:    make([]*toydbv1.Row, len(r.Rows)),
	}
	for i, row := range r.Rows {
		rs.Rows[i] = encodeRow(row, r.Columns)
	}
	return rs
}

// encodeRow converts a map[string]any row into a protobuf Row.
func encodeRow(row engine.Row, cols []string) *toydbv1.Row {
	pb := &toydbv1.Row{Values: make(map[string]*toydbv1.Value, len(cols))}
	for _, col := range cols {
		pb.Values[col] = encodeValue(row[col])
	}
	return pb
}

// encodeValue converts a Go value to a protobuf Value.
func encodeValue(v any) *toydbv1.Value {
	if v == nil {
		return &toydbv1.Value{IsNull: true}
	}
	switch v := v.(type) {
	case int64:
		return &toydbv1.Value{Kind: &toydbv1.Value_IntVal{IntVal: v}}
	case float64:
		return &toydbv1.Value{Kind: &toydbv1.Value_FloatVal{FloatVal: v}}
	case string:
		return &toydbv1.Value{Kind: &toydbv1.Value_TextVal{TextVal: v}}
	case bool:
		return &toydbv1.Value{Kind: &toydbv1.Value_BoolVal{BoolVal: v}}
	default:
		return &toydbv1.Value{Kind: &toydbv1.Value_TextVal{TextVal: fmt.Sprintf("%v", v)}}
	}
}
