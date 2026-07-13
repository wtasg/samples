// cmd/client/main.go — DocDB Client Web Server.
//
// This server connects to DocDB gRPC backend via github.com/docdb/client/docdb,
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

	"github.com/docdb/client/docdb"
)

type QueryRequest struct {
	Command string `json:"command"`
}

type QueryResponse struct {
	Ok      bool        `json:"ok"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
	Docs    []docdb.Doc `json:"docs,omitempty"`
}

type CollectionSchema struct {
	Name     string `json:"name"`
	DocCount int64  `json:"doc_count"`
	Size     int64  `json:"size_bytes"`
}

var db *docdb.Client

func main() {
	port := flag.String("port", "8080", "web server port")
	dbAddr := flag.String("db", "http://localhost:60013", "DocDB server address")
	flag.Parse()

	// Address from environment overrides.
	envDBAddr := os.Getenv("DOCDB_ADDR")
	if envDBAddr != "" {
		*dbAddr = envDBAddr
	}
	envPort := os.Getenv("PORT")
	if envPort != "" {
		*port = envPort
	}

	log.Printf("Connecting to DocDB backend at %s...", *dbAddr)
	db = docdb.NewClient(*dbAddr)
	defer db.Close()

	// Verify connection on startup.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	info, err := db.Ping(ctx)
	cancel()
	if err != nil {
		log.Printf("Warning: cannot ping DocDB server: %v", err)
	} else {
		log.Printf("Successfully connected to database. Version: %s, Go: %s", info.Version, info.GoVersion)
	}

	// Serve static UI.
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	// API endpoints.
	http.HandleFunc("/api/query", handleQuery)
	http.HandleFunc("/api/collections", handleCollections)
	http.HandleFunc("/api/ping", handlePing)

	log.Printf("DocDB Web UI listening on port %s...", *port)
	if err := http.ListenAndServe(":"+*port, nil); err != nil {
		log.Fatalf("ListenAndServe: %v", err)
	}
}

// handleQuery executes the requested command on DocDB and returns JSON.
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

	result, err := db.Execute(ctx, req.Command)
	if err != nil {
		json.NewEncoder(w).Encode(QueryResponse{Ok: false, Error: err.Error()})
		return
	}

	resp := QueryResponse{
		Ok:      true,
		Message: result.Message,
		Docs:    result.Docs,
	}
	json.NewEncoder(w).Encode(resp)
}

// handleCollections fetches collection schemas/stats from DocDB catalog.
func handleCollections(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	collections, err := db.ListCollections(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list collections: %v", err), http.StatusInternalServerError)
		return
	}

	schemas := make([]CollectionSchema, 0, len(collections))
	for _, name := range collections {
		info, err := db.DescribeCollection(ctx, name)
		if err != nil {
			continue
		}
		schemas = append(schemas, CollectionSchema{
			Name:     info.Name,
			DocCount: info.DocCount,
			Size:     info.Size,
		})
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
