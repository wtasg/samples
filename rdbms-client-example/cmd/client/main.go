// cmd/client/main.go — ToyDB Client Web Server.
//
// This server connects to ToyDB gRPC backend via github.com/toydb/client/toydb,
// serves static files (HTML/CSS/JS), and exposes REST endpoints for the UI.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/toydb/client/toydb"
)

type QueryRequest struct {
	SQL string `json:"sql"`
}

type QueryResponse struct {
	Ok      bool         `json:"ok"`
	Message string       `json:"message,omitempty"`
	Error   string       `json:"error,omitempty"`
	Columns []string     `json:"columns,omitempty"`
	Rows    []toydb.Row  `json:"rows,omitempty"`
}

type ColumnSchema struct {
	Name string `json:"name"`
	Type string `json:"type"`
	PK   bool   `json:"pk"`
}

type TableSchema struct {
	Name    string         `json:"name"`
	Columns []ColumnSchema `json:"columns"`
}

var db *toydb.Client

func main() {
	port := flag.String("port", "8080", "web server port")
	dbAddr := flag.String("db", "http://localhost:9090", "ToyDB server address")
	flag.Parse()

	// Address from environment overrides.
	envDBAddr := os.Getenv("TOYDB_ADDR")
	if envDBAddr != "" {
		*dbAddr = envDBAddr
	}
	envPort := os.Getenv("PORT")
	if envPort != "" {
		*port = envPort
	}

	log.Printf("Connecting to ToyDB backend at %s...", *dbAddr)
	db = toydb.NewClient(*dbAddr)
	defer db.Close()

	// Verify connection on startup.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	info, err := db.Ping(ctx)
	cancel()
	if err != nil {
		log.Printf("Warning: cannot ping ToyDB server: %v", err)
	} else {
		log.Printf("Successfully connected to database. Version: %s, Go: %s", info.Version, info.GoVersion)
	}

	// Serve static UI.
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// API endpoints.
	http.HandleFunc("/api/query", handleQuery)
	http.HandleFunc("/api/tables", handleTables)
	http.HandleFunc("/api/ping", handlePing)

	log.Printf("ToyDB Web UI listening on port %s...", *port)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}

// handleQuery executes the requested SQL query on ToyDB and returns JSON.
func handleQuery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(QueryResponse{Ok: false, Error: "Invalid JSON payload"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := db.Execute(ctx, req.SQL)
	if err != nil {
		json.NewEncoder(w).Encode(QueryResponse{Ok: false, Error: err.Error()})
		return
	}

	resp := QueryResponse{
		Ok:      true,
		Message: result.Message,
		Columns: result.Columns,
		Rows:    result.Rows,
	}
	json.NewEncoder(w).Encode(resp)
}

// handleTables fetches table schemas from ToyDB catalog and returns JSON.
func handleTables(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	tables, err := db.ListTables(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list tables: %v", err), http.StatusInternalServerError)
		return
	}

	schemas := make([]TableSchema, 0, len(tables))
	for _, name := range tables {
		ts, err := db.DescribeTable(ctx, name)
		if err != nil {
			continue
		}
		cols := make([]ColumnSchema, len(ts.Columns))
		for i, col := range ts.Columns {
			cols[i] = ColumnSchema{
				Name: col.Name,
				Type: string(col.Type),
				PK:   col.PrimaryKey,
			}
		}
		schemas = append(schemas, TableSchema{Name: ts.Name, Columns: cols})
	}

	json.NewEncoder(w).Encode(schemas)
}

// handlePing returns database info.
func handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	info, err := db.Ping(ctx)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{
			"online": false,
			"error":  err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]any{
		"online":     true,
		"version":    info.Version,
		"go_version": info.GoVersion,
		"data_dir":   info.DataDir,
	})
}
