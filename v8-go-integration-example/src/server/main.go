package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"v8-go-integration/src/engine"
	"v8-go-integration/src/v8"
)

// APIServer handles requests and executes scripts using the engine.ScriptRunner abstraction.
// The handler is decoupled from the concrete V8 engine implementation.
type APIServer struct {
	runner engine.ScriptRunner
}

func main() {
	port := "60001"
	log.Printf("Starting V8-Go Integration Server on port %s...", port)

	// Instantiate the V8 engine runner
	v8Runner := v8.NewV8Runner()

	// Instantiate the API Server, injecting the engine runner interface
	apiServer := &APIServer{
		runner: v8Runner,
	}

	// Determine client directory path dynamically
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}
	clientDir := filepath.Join(cwd, "src", "client")
	log.Printf("Serving client files from: %s", clientDir)

	// Setup handlers
	fs := http.FileServer(http.Dir(clientDir))
	http.Handle("/", fs)
	http.HandleFunc("/api/run", apiServer.handleRunScript)

	log.Printf("Server is ready at http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func (s *APIServer) handleRunScript(w http.ResponseWriter, r *http.Request) {
	// Setup CORS
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req engine.ScriptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON payload: %v", err), http.StatusBadRequest)
		return
	}

	// Execute via decoupled interface
	resp := s.runner.Run(req)

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}
