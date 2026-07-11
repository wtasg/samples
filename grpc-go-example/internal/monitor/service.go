package monitor

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"sync"
	"time"

	"connectrpc.com/connect"
	monitorv1 "grpc-go-example/gen/monitor/v1"
	"grpc-go-example/gen/monitor/v1/monitorv1connect"
)

// Ensure MonitorServiceServer implements the generated interface.
var _ monitorv1connect.MonitorServiceHandler = (*MonitorServiceServer)(nil)

// MonitorServiceServer implements the MonitorService gRPC/Connect handler.
type MonitorServiceServer struct {
	startTime time.Time
	mu        sync.Mutex

	// CPU sampling state
	lastCPUSample cpuSample
}

// NewMonitorServiceServer creates a new service instance.
func NewMonitorServiceServer() *MonitorServiceServer {
	s := &MonitorServiceServer{startTime: time.Now()}
	s.lastCPUSample = takeCPUSample()
	return s
}

// ── GetStatus ─────────────────────────────────────────────────

func (s *MonitorServiceServer) GetStatus(
	ctx context.Context,
	req *connect.Request[monitorv1.GetStatusRequest],
) (*connect.Response[monitorv1.GetStatusResponse], error) {
	resp := &monitorv1.GetStatusResponse{
		Status:    "alive",
		Version:   "1.0.0",
		UptimeS:   time.Since(s.startTime).Seconds(),
		CpuCount:  int32(runtime.NumCPU()),
		GoVersion: runtime.Version(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	return connect.NewResponse(resp), nil
}

// ── StreamCPU ─────────────────────────────────────────────────

func (s *MonitorServiceServer) StreamCPU(
	ctx context.Context,
	req *connect.Request[monitorv1.StreamCPURequest],
	stream *connect.ServerStream[monitorv1.CPUSample],
) error {
	intervalMs := req.Msg.IntervalMs
	if intervalMs < 200 {
		intervalMs = 200
	}
	if intervalMs > 5000 {
		intervalMs = 5000
	}

	ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			sample := s.cpuPercent()
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)

			msg := &monitorv1.CPUSample{
				CpuPercent:  sample,
				MemUsedMb:   float64(memStats.Alloc) / 1024 / 1024,
				MemTotalMb:  float64(memStats.Sys) / 1024 / 1024,
				Goroutines:  int64(runtime.NumGoroutine()),
				Timestamp:   time.Now().UTC().Format(time.RFC3339Nano),
			}
			if err := stream.Send(msg); err != nil {
				return err
			}
		}
	}
}

// cpuPercent returns a rough CPU % by measuring goroutine count change as a proxy.
// On Linux we use /proc/stat for a real reading.
func (s *MonitorServiceServer) cpuPercent() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := takeCPUSample()
	prev := s.lastCPUSample
	s.lastCPUSample = now

	totalDiff := (now.user + now.system + now.idle) - (prev.user + prev.system + prev.idle)
	idleDiff := now.idle - prev.idle

	if totalDiff <= 0 {
		return 0
	}
	pct := 100.0 * float64(totalDiff-idleDiff) / float64(totalDiff)
	return math.Round(pct*100) / 100
}

// ── Echo ──────────────────────────────────────────────────────

func (s *MonitorServiceServer) Echo(
	ctx context.Context,
	req *connect.Request[monitorv1.EchoRequest],
) (*connect.Response[monitorv1.EchoResponse], error) {
	msg := req.Msg.Message
	if len(msg) > 1024 {
		msg = msg[:1024]
	}
	resp := &monitorv1.EchoResponse{
		Message:    fmt.Sprintf("Echo: %s", msg),
		ServerTime: time.Now().UTC().Format(time.RFC3339Nano),
		Length:     int64(len(msg)),
	}
	return connect.NewResponse(resp), nil
}
