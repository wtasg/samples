package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	monitorhandler "grpc-go-example/internal/monitor"
	"grpc-go-example/gen/monitor/v1/monitorv1connect"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "60009"
	}

	// Create the Connect-Go service handler
	svc := monitorhandler.NewMonitorServiceServer()
	mux := http.NewServeMux()

	// Register the Connect handler (serves gRPC, gRPC-Web, and Connect protocol)
	path, handler := monitorv1connect.NewMonitorServiceHandler(
		svc,
		connect.WithCompressMinBytes(1024),
	)
	mux.Handle(path, corsMiddleware(handler))

	// Serve the static web client
	clientDir := resolveClientDir()
	mux.Handle("/", http.FileServer(http.Dir(clientDir)))

	// Use h2c (HTTP/2 cleartext) so the browser can use gRPC-Web over plain HTTP.
	// For production you'd add TLS; h2c works well for local development.
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      h2c.NewHandler(mux, &http2.Server{}),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // disabled for server-streaming RPCs
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("gRPC / Connect server running at http://localhost:%s", port)
	log.Printf("Protocols: gRPC (HTTP/2), gRPC-Web, Connect (HTTP/1.1 + HTTP/2)")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}

// corsMiddleware adds CORS headers required for browser-side Connect/gRPC-Web calls.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers",
			"Content-Type, Connect-Protocol-Version, Connect-Timeout-Ms, "+
				"Grpc-Timeout, X-Grpc-Web, X-User-Agent")
		w.Header().Set("Access-Control-Expose-Headers",
			"Grpc-Status, Grpc-Message, Grpc-Status-Details-Bin")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// resolveClientDir finds the client static files directory.
func resolveClientDir() string {
	// Try relative to the working directory
	candidates := []string{"client", "../../client"}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}
	// Fallback
	return "client"
}
