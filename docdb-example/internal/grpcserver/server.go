// Package grpcserver implements the DocDB Connect-RPC service.
//
// The service wraps engine.Executor and translates between protobuf messages
// and the engine's Doc/Result types. It supports:
//   - Execute  — any command (unary)
//   - Query    — find query with server-streaming
//   - Ping     — health check
//   - ListCollections / DescribeCollection — collection introspection
package grpcserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"connectrpc.com/connect"

	docdbv1 "github.com/docdb/client/gen/docdb/v1"
	"github.com/docdb/client/gen/docdb/v1/docdbv1connect"
	"docdb/internal/engine"
	"docdb/internal/parser"
)

const version = "DocDB 1.0"

// Service implements DocDBHandler.
type Service struct {
	docdbv1connect.UnimplementedDocDBHandler
	ex      *engine.Executor
	dataDir string
}

// New returns a new Service backed by ex.
func New(ex *engine.Executor, dataDir string) *Service {
	return &Service{ex: ex, dataDir: dataDir}
}

// ── Execute ──
func (s *Service) Execute(
	_ context.Context,
	req *connect.Request[docdbv1.ExecuteRequest],
) (*connect.Response[docdbv1.ExecuteResponse], error) {

	stmts := SplitStatements(req.Msg.Command)
	if len(stmts) == 0 {
		return connect.NewResponse(&docdbv1.ExecuteResponse{Ok: true}), nil
	}

	var allDocs []*docdbv1.Document
	var lastMessage string

	for _, stmtStr := range stmts {
		stmt, err := parser.Parse(stmtStr)
		if err != nil {
			return connect.NewResponse(&docdbv1.ExecuteResponse{
				Ok:    false,
				Error: fmt.Sprintf("parse error on %q: %v", stmtStr, err),
			}), nil
		}
		if stmt == nil {
			continue
		}

		result, err := s.ex.Execute(stmt)
		if err != nil {
			return connect.NewResponse(&docdbv1.ExecuteResponse{
				Ok:    false,
				Error: err.Error(),
			}), nil
		}

		if result != nil {
			if result.Message != "" {
				if lastMessage != "" {
					lastMessage += "\n"
				}
				lastMessage += result.Message
			}
			if len(result.Docs) > 0 {
				allDocs = append(allDocs, encodeDocuments(result.Docs)...)
			}
		}
	}

	resp := &docdbv1.ExecuteResponse{
		Ok:      true,
		Message: lastMessage,
		Docs:    allDocs,
	}
	return connect.NewResponse(resp), nil
}

// ── Query (server-streaming) ──
func (s *Service) Query(
	_ context.Context,
	req *connect.Request[docdbv1.QueryRequest],
	stream *connect.ServerStream[docdbv1.QueryStreamResponse],
) error {

	stmts := SplitStatements(req.Msg.Command)
	for _, stmtStr := range stmts {
		stmt, err := parser.Parse(stmtStr)
		if err != nil {
			return stream.Send(&docdbv1.QueryStreamResponse{
				Error: fmt.Sprintf("parse error on %q: %v", stmtStr, err),
			})
		}
		if stmt == nil {
			continue
		}

		result, err := s.ex.Execute(stmt)
		if err != nil {
			return stream.Send(&docdbv1.QueryStreamResponse{Error: err.Error()})
		}
		if result == nil || len(result.Docs) == 0 {
			continue
		}

		for _, doc := range result.Docs {
			msg := &docdbv1.QueryStreamResponse{
				Doc: encodeDocument(doc),
			}
			if err := stream.Send(msg); err != nil {
				return err
			}
		}
	}
	return nil
}

// ── Ping ──
func (s *Service) Ping(
	_ context.Context,
	_ *connect.Request[docdbv1.PingRequest],
) (*connect.Response[docdbv1.PingResponse], error) {

	return connect.NewResponse(&docdbv1.PingResponse{
		Version:   version,
		GoVersion: runtime.Version(),
		DataDir:   s.dataDir,
	}), nil
}

// ── ListCollections ──
func (s *Service) ListCollections(
	_ context.Context,
	_ *connect.Request[docdbv1.ListCollectionsRequest],
) (*connect.Response[docdbv1.ListCollectionsResponse], error) {

	collections := s.ex.CollectionNames()
	return connect.NewResponse(&docdbv1.ListCollectionsResponse{Collections: collections}), nil
}

// ── DescribeCollection ──
func (s *Service) DescribeCollection(
	_ context.Context,
	req *connect.Request[docdbv1.DescribeCollectionRequest],
) (*connect.Response[docdbv1.DescribeCollectionResponse], error) {

	name := req.Msg.Collection
	names := s.ex.CollectionNames()
	found := false
	for _, c := range names {
		if c == name {
			found = true
			break
		}
	}
	if !found {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("collection %q not found", name))
	}

	// We can estimate size by checking store file size
	var sizeBytes int64 = 0
	filePath := filepath.Join(s.dataDir, name+".docs")
	if info, err := os.Stat(filePath); err == nil {
		sizeBytes = info.Size()
	}

	// Count docs using executor/find
	docCount := int64(0)
	findStmt := &parser.FindStmt{Collection: name}
	if res, err := s.ex.Execute(findStmt); err == nil {
		docCount = int64(len(res.Docs))
	}

	return connect.NewResponse(&docdbv1.DescribeCollectionResponse{
		Collection: name,
		DocCount:   docCount,
		SizeBytes:  sizeBytes,
	}), nil
}

// ── Encoding helpers ──

func encodeDocuments(docs []engine.Doc) []*docdbv1.Document {
	pbDocs := make([]*docdbv1.Document, len(docs))
	for i, doc := range docs {
		pbDocs[i] = encodeDocument(doc)
	}
	return pbDocs
}

func encodeDocument(doc engine.Doc) *docdbv1.Document {
	fields := make(map[string]*docdbv1.Value, len(doc))
	for k, v := range doc {
		fields[k] = encodeValue(v)
	}
	return &docdbv1.Document{Fields: fields}
}

func encodeValue(v any) *docdbv1.Value {
	if v == nil {
		return &docdbv1.Value{IsNull: true}
	}
	switch v := v.(type) {
	case int64:
		return &docdbv1.Value{Kind: &docdbv1.Value_IntVal{IntVal: v}}
	case int:
		return &docdbv1.Value{Kind: &docdbv1.Value_IntVal{IntVal: int64(v)}}
	case float64:
		// Check if it's an integer stored as float64
		if v == float64(int64(v)) {
			return &docdbv1.Value{Kind: &docdbv1.Value_IntVal{IntVal: int64(v)}}
		}
		return &docdbv1.Value{Kind: &docdbv1.Value_FloatVal{FloatVal: v}}
	case string:
		return &docdbv1.Value{Kind: &docdbv1.Value_TextVal{TextVal: v}}
	case bool:
		return &docdbv1.Value{Kind: &docdbv1.Value_BoolVal{BoolVal: v}}
	case map[string]any, []any:
		b, _ := json.Marshal(v)
		return &docdbv1.Value{Kind: &docdbv1.Value_TextVal{TextVal: string(b)}}
	default:
		return &docdbv1.Value{Kind: &docdbv1.Value_TextVal{TextVal: fmt.Sprintf("%v", v)}}
	}
}

// SplitStatements splits multiple NoSQL queries by semicolon, ignoring semicolons
// within quotes and parentheses/braces.
func SplitStatements(input string) []string {
	var stmts []string
	var current strings.Builder
	inSingleQuote := false
	inDoubleQuote := false
	depth := 0

	for i := 0; i < len(input); i++ {
		ch := input[i]

		// Handle escape
		if ch == '\\' && i+1 < len(input) {
			current.WriteByte(ch)
			current.WriteByte(input[i+1])
			i++
			continue
		}

		switch ch {
		case '\'':
			if !inDoubleQuote {
				inSingleQuote = !inSingleQuote
			}
		case '"':
			if !inSingleQuote {
				inDoubleQuote = !inDoubleQuote
			}
		case '(', '{', '[':
			if !inSingleQuote && !inDoubleQuote {
				depth++
			}
		case ')', '}', ']':
			if !inSingleQuote && !inDoubleQuote {
				depth--
			}
		case ';':
			if !inSingleQuote && !inDoubleQuote && depth == 0 {
				stmt := strings.TrimSpace(current.String())
				if stmt != "" {
					stmts = append(stmts, stmt)
				}
				current.Reset()
				continue
			}
		}
		current.WriteByte(ch)
	}

	stmt := strings.TrimSpace(current.String())
	if stmt != "" {
		stmts = append(stmts, stmt)
	}
	return stmts
}
