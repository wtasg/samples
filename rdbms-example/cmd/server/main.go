// cmd/server/main.go — ToyDB gRPC / Connect-RPC server.
//
// Usage:
//
//	go run ./cmd/server [flags]
//
// Flags:
//
//	--addr  :9090          listen address (default :9090)
//	--data  ./data         data directory (default ./data)
//
// The server speaks three wire protocols on the same port:
//   - Connect protocol  (JSON or binary, HTTP/1.1 or HTTP/2)
//   - gRPC protocol     (binary protobuf over HTTP/2) ← standard gRPC clients
//   - gRPC-Web protocol (binary protobuf, works over HTTP/1.1)
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"rdbms/gen/toydb/v1/toydbv1connect"
	"rdbms/internal/engine"
	"rdbms/internal/grpcserver"
)

func main() {
	addr := flag.String("addr", ":9090", "listen address")
	dataDir := flag.String("data", "data", "data directory")
	flag.Parse()

	// Ensure data directory exists.
	if err := os.MkdirAll(*dataDir, 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	// Open the query executor (loads catalog + opens all tables).
	ex, err := engine.NewExecutor(*dataDir)
	if err != nil {
		log.Fatalf("open executor: %v", err)
	}
	defer ex.Close()

	// Build the ConnectRPC handler.
	svc := grpcserver.New(ex, *dataDir)
	mux := http.NewServeMux()

	// NewToyDBHandler returns the path prefix and an http.Handler.
	// We use connect.WithCompressMinBytes to enable gzip on large responses.
	path, handler := toydbv1connect.NewToyDBHandler(
		svc,
		connect.WithCompressMinBytes(1024),
	)
	mux.Handle(path, handler)

	// Healthcheck endpoint (useful for load-balancers / Kubernetes probes).
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})

	// h2c allows HTTP/2 over plain TCP (no TLS) — standard for local gRPC.
	srv := &http.Server{
		Addr:              *addr,
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("shutting down…")
		ex.Close()
		os.Exit(0)
	}()

	fmt.Printf("╔══════════════════════════════════════════════╗\n")
	fmt.Printf("║         ToyDB gRPC / Connect server          ║\n")
	fmt.Printf("╚══════════════════════════════════════════════╝\n")
	fmt.Printf("  addr    : %s\n", *addr)
	fmt.Printf("  data    : %s\n", *dataDir)
	fmt.Printf("  proto   : toydb.v1.ToyDB\n")
	fmt.Printf("  protocols: gRPC · Connect · gRPC-Web\n\n")

	log.Printf("listening on %s", *addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serve: %v", err)
	}
}
