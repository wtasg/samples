package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"path", "status"},
	)
	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "status"},
	)
)

type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Method    string `json:"method"`
	Path      string `json:"path"`
	Status    int    `json:"status"`
	Duration  string `json:"duration_ms"`
	IP        string `json:"ip"`
	UserAgent string `json:"user_agent"`
}

var logFile *os.File

func initLogger() {
	logPath := "/var/log/app/access.log"
	// Ensure directory exists
	dir := filepath.Dir(logPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("failed to create log directory: %v", err)
	}

	var err error
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("failed to open log file: %v", err)
	}
}

func logRequest(r *http.Request, status int, duration time.Duration) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Method:    r.Method,
		Path:      r.URL.Path,
		Status:    status,
		Duration:  fmt.Sprintf("%.3f", float64(duration.Nanoseconds())/1e6),
		IP:        r.RemoteAddr,
		UserAgent: r.UserAgent(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("failed to marshal log entry: %v", err)
		return
	}

	if _, err := logFile.Write(append(data, '\n')); err != nil {
		log.Printf("failed to write to log file: %v", err)
	}
}

func main() {
	initLogger()
	defer logFile.Close()

	mux := http.NewServeMux()

	// Decorator/Middleware to collect metrics and log request
	instrument := func(path string, handler http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			
			// We intercept the status using a custom ResponseWriter
			lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			
			handler(lrw, r)
			
			duration := time.Since(start)
			statusStr := strconv.Itoa(lrw.statusCode)
			
			httpRequestsTotal.WithLabelValues(path, statusStr).Inc()
			httpRequestDuration.WithLabelValues(path, statusStr).Observe(duration.Seconds())
			
			logRequest(r, lrw.statusCode, duration)
		}
	}

	mux.HandleFunc("/", instrument("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "not found"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"message": "Hello from the Go Application inside the sidecar stack!"}`))
	}))

	mux.HandleFunc("/compute", instrument("/compute", func(w http.ResponseWriter, r *http.Request) {
		// Simulate CPU work
		count := 0
		for i := 2; i < 200000; i++ {
			isPrime := true
			for j := 2; j*j <= i; j++ {
				if i%j == 0 {
					isPrime = false
					break
				}
			}
			if isPrime {
				count++
			}
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"message": "computation complete", "primes_found": %d}`, count)
	}))

	mux.HandleFunc("/error", instrument("/error", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "simulated internal server error"}`))
	}))

	// /metrics endpoint is handled by Prometheus directly
	mux.Handle("/metrics", promhttp.Handler())

	serverAddr := "127.0.0.1:8080"
	log.Printf("Server listening on %s", serverAddr)
	if err := http.ListenAndServe(serverAddr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
