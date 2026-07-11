package monitor_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	monitorv1 "grpc-go-example/gen/monitor/v1"
	"grpc-go-example/gen/monitor/v1/monitorv1connect"
	monitor "grpc-go-example/internal/monitor"
)

// setupTestServer creates a test HTTP server backed by the MonitorServiceServer.
func setupTestServer(t *testing.T) (monitorv1connect.MonitorServiceClient, *httptest.Server) {
	t.Helper()

	svc := monitor.NewMonitorServiceServer()
	mux := http.NewServeMux()
	path, handler := monitorv1connect.NewMonitorServiceHandler(svc)
	mux.Handle(path, handler)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	client := monitorv1connect.NewMonitorServiceClient(
		http.DefaultClient,
		srv.URL,
		connect.WithSendGzip(),
	)
	return client, srv
}

// ── GetStatus ─────────────────────────────────────────────────

func TestGetStatus_ReturnsAlive(t *testing.T) {
	client, _ := setupTestServer(t)
	ctx := context.Background()

	resp, err := client.GetStatus(ctx, connect.NewRequest(&monitorv1.GetStatusRequest{}))
	if err != nil {
		t.Fatalf("GetStatus returned error: %v", err)
	}

	if resp.Msg.Status != "alive" {
		t.Errorf("expected status='alive', got %q", resp.Msg.Status)
	}
}

func TestGetStatus_UptimeIsNonNegative(t *testing.T) {
	client, _ := setupTestServer(t)
	ctx := context.Background()

	resp, err := client.GetStatus(ctx, connect.NewRequest(&monitorv1.GetStatusRequest{}))
	if err != nil {
		t.Fatalf("GetStatus error: %v", err)
	}
	if resp.Msg.UptimeS < 0 {
		t.Errorf("uptime should be >= 0, got %f", resp.Msg.UptimeS)
	}
}

func TestGetStatus_HasGoVersionAndCPUCount(t *testing.T) {
	client, _ := setupTestServer(t)
	ctx := context.Background()

	resp, err := client.GetStatus(ctx, connect.NewRequest(&monitorv1.GetStatusRequest{}))
	if err != nil {
		t.Fatalf("GetStatus error: %v", err)
	}
	if resp.Msg.GoVersion == "" {
		t.Error("expected non-empty GoVersion")
	}
	if resp.Msg.CpuCount <= 0 {
		t.Errorf("expected CpuCount > 0, got %d", resp.Msg.CpuCount)
	}
	if resp.Msg.Timestamp == "" {
		t.Error("expected non-empty Timestamp")
	}
}

// ── StreamCPU ─────────────────────────────────────────────────

func TestStreamCPU_ReceivesAtLeastOneSample(t *testing.T) {
	client, _ := setupTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.StreamCPU(ctx, connect.NewRequest(&monitorv1.StreamCPURequest{
		IntervalMs: 200,
	}))
	if err != nil {
		t.Fatalf("StreamCPU start error: %v", err)
	}
	defer stream.Close()

	if !stream.Receive() {
		t.Fatal("expected to receive at least one CPU sample")
	}
	sample := stream.Msg()
	if sample.CpuPercent < 0 || sample.CpuPercent > 100 {
		t.Errorf("CPU percent out of range: %f", sample.CpuPercent)
	}
	if sample.Goroutines <= 0 {
		t.Errorf("expected goroutines > 0, got %d", sample.Goroutines)
	}
	if sample.Timestamp == "" {
		t.Error("expected non-empty timestamp in CPU sample")
	}
}

func TestStreamCPU_ClampsIntervalBelow200ms(t *testing.T) {
	// Interval of 10ms should be clamped to 200ms internally.
	// We verify we still receive samples (the stream works) — the clamping
	// means the sample arrives within 1s even if requested as 10ms.
	client, _ := setupTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stream, err := client.StreamCPU(ctx, connect.NewRequest(&monitorv1.StreamCPURequest{
		IntervalMs: 10, // should be clamped to 200
	}))
	if err != nil {
		t.Fatalf("StreamCPU error: %v", err)
	}
	defer stream.Close()

	if !stream.Receive() {
		t.Fatal("expected to receive a sample even with clamped interval")
	}
}

func TestStreamCPU_ReceivesMultipleSamples(t *testing.T) {
	client, _ := setupTestServer(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.StreamCPU(ctx, connect.NewRequest(&monitorv1.StreamCPURequest{
		IntervalMs: 200,
	}))
	if err != nil {
		t.Fatalf("StreamCPU error: %v", err)
	}
	defer stream.Close()

	count := 0
	for stream.Receive() {
		count++
		if count >= 3 {
			break
		}
	}
	if count < 3 {
		t.Errorf("expected at least 3 samples, got %d", count)
	}
}

// ── Echo ──────────────────────────────────────────────────────

func TestEcho_ReturnsEchoedMessage(t *testing.T) {
	client, _ := setupTestServer(t)
	ctx := context.Background()

	resp, err := client.Echo(ctx, connect.NewRequest(&monitorv1.EchoRequest{
		Message: "hello-grpc",
	}))
	if err != nil {
		t.Fatalf("Echo error: %v", err)
	}
	if resp.Msg.Message == "" {
		t.Error("expected non-empty echo message")
	}
	if resp.Msg.Length != int64(len("hello-grpc")) {
		t.Errorf("expected length=%d, got %d", len("hello-grpc"), resp.Msg.Length)
	}
	if resp.Msg.ServerTime == "" {
		t.Error("expected non-empty server time")
	}
}

func TestEcho_HandlesEmptyString(t *testing.T) {
	client, _ := setupTestServer(t)
	ctx := context.Background()

	resp, err := client.Echo(ctx, connect.NewRequest(&monitorv1.EchoRequest{Message: ""}))
	if err != nil {
		t.Fatalf("Echo error on empty string: %v", err)
	}
	if resp.Msg.Length != 0 {
		t.Errorf("expected length=0 for empty string, got %d", resp.Msg.Length)
	}
}

func TestEcho_TruncatesLongMessages(t *testing.T) {
	client, _ := setupTestServer(t)
	ctx := context.Background()

	// Send a message longer than 1024 chars
	long := string(make([]byte, 2000))
	resp, err := client.Echo(ctx, connect.NewRequest(&monitorv1.EchoRequest{Message: long}))
	if err != nil {
		t.Fatalf("Echo error: %v", err)
	}
	if resp.Msg.Length > 1024 {
		t.Errorf("expected message truncated to <=1024 chars, got length=%d", resp.Msg.Length)
	}
}

// ── Concurrency ───────────────────────────────────────────────

func TestGetStatus_Concurrent(t *testing.T) {
	client, _ := setupTestServer(t)
	const n = 10
	errs := make(chan error, n)

	for i := 0; i < n; i++ {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, err := client.GetStatus(ctx, connect.NewRequest(&monitorv1.GetStatusRequest{}))
			errs <- err
		}()
	}

	for i := 0; i < n; i++ {
		if err := <-errs; err != nil {
			t.Errorf("concurrent GetStatus error: %v", err)
		}
	}
}
