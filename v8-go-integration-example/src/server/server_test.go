package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"v8-go-integration/src/engine"
	"v8-go-integration/src/v8"
)

// mockRunner is a test helper that implements engine.ScriptRunner
type mockRunner struct {
	runFunc func(req engine.ScriptRequest) engine.ScriptResponse
}

func (m *mockRunner) Run(req engine.ScriptRequest) engine.ScriptResponse {
	if m.runFunc != nil {
		return m.runFunc(req)
	}
	return engine.ScriptResponse{}
}

// TestAPIServer_HandleRunScript_OPTIONS tests the CORS preflight request.
func TestAPIServer_HandleRunScript_OPTIONS(t *testing.T) {
	server := &APIServer{runner: &mockRunner{}}

	req := httptest.NewRequest(http.MethodOptions, "/api/run", nil)
	w := httptest.NewRecorder()

	server.handleRunScript(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	headers := []struct {
		key, expected string
	}{
		{"Access-Control-Allow-Origin", "*"},
		{"Access-Control-Allow-Methods", "POST, OPTIONS"},
		{"Access-Control-Allow-Headers", "Content-Type"},
	}

	for _, h := range headers {
		if val := resp.Header.Get(h.key); val != h.expected {
			t.Errorf("Expected header %s to be %q, got %q", h.key, h.expected, val)
		}
	}
}

// TestAPIServer_HandleRunScript_MethodNotAllowed tests non-POST requests.
func TestAPIServer_HandleRunScript_MethodNotAllowed(t *testing.T) {
	server := &APIServer{runner: &mockRunner{}}

	req := httptest.NewRequest(http.MethodGet, "/api/run", nil)
	w := httptest.NewRecorder()

	server.handleRunScript(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 Method Not Allowed, got %d", resp.StatusCode)
	}
}

// TestAPIServer_HandleRunScript_BadRequest tests malformed JSON body.
func TestAPIServer_HandleRunScript_BadRequest(t *testing.T) {
	server := &APIServer{runner: &mockRunner{}}

	req := httptest.NewRequest(http.MethodPost, "/api/run", strings.NewReader("{invalid-json"))
	w := httptest.NewRecorder()

	server.handleRunScript(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 Bad Request, got %d", resp.StatusCode)
	}
}

// TestAPIServer_HandleRunScript_Success tests handling of valid run script request.
func TestAPIServer_HandleRunScript_Success(t *testing.T) {
	expectedResponse := engine.ScriptResponse{
		Success:    true,
		Result:     "hello from mock",
		Logs:       []string{"mock log 1"},
		DurationMs: 12,
	}

	runner := &mockRunner{
		runFunc: func(req engine.ScriptRequest) engine.ScriptResponse {
			if req.Script != "console.log('test')" {
				t.Errorf("Expected script %q, got %q", "console.log('test')", req.Script)
			}
			return expectedResponse
		},
	}

	server := &APIServer{runner: runner}

	payload, err := json.Marshal(engine.ScriptRequest{Script: "console.log('test')"})
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/run", bytes.NewReader(payload))
	w := httptest.NewRecorder()

	server.handleRunScript(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	if contentType := resp.Header.Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %q", contentType)
	}

	var gotResponse engine.ScriptResponse
	if err := json.NewDecoder(resp.Body).Decode(&gotResponse); err != nil {
		t.Fatalf("Failed to decode response body: %v", err)
	}

	if gotResponse.Success != expectedResponse.Success {
		t.Errorf("Expected success %t, got %t", expectedResponse.Success, gotResponse.Success)
	}
	if gotResponse.Result != expectedResponse.Result {
		t.Errorf("Expected result %q, got %q", expectedResponse.Result, gotResponse.Result)
	}
	if len(gotResponse.Logs) != len(expectedResponse.Logs) || gotResponse.Logs[0] != expectedResponse.Logs[0] {
		t.Errorf("Expected logs %v, got %v", expectedResponse.Logs, gotResponse.Logs)
	}
	if gotResponse.DurationMs != expectedResponse.DurationMs {
		t.Errorf("Expected duration %d, got %d", expectedResponse.DurationMs, gotResponse.DurationMs)
	}
}

// TestAPIServer_Integration tests the API Server handler with a real V8Runner.
func TestAPIServer_Integration(t *testing.T) {
	// Instantiate actual V8Runner
	realRunner := v8.NewV8Runner()
	server := &APIServer{runner: realRunner}

	// Create request with JS script utilizing console.log and goCompute
	jsScript := `
		console.log("integration start");
		const res = goCompute(5, 6);
		console.log("integration compute result: " + res);
		res;
	`
	payload, err := json.Marshal(engine.ScriptRequest{Script: jsScript})
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	ts := httptest.NewServer(http.HandlerFunc(server.handleRunScript))
	defer ts.Close()

	resp, err := http.Post(ts.URL, "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 OK, got %d", resp.StatusCode)
	}

	var runResp engine.ScriptResponse
	if err := json.NewDecoder(resp.Body).Decode(&runResp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !runResp.Success {
		t.Fatalf("Integration execution failed: %s", runResp.Error)
	}

	if runResp.Result != "30" {
		t.Errorf("Expected execution result '30', got %q", runResp.Result)
	}

	expectedLogs := []string{
		"integration start",
		"[Go Callback] goCompute: Multiplying 5 * 6 in Go -> 30",
		"integration compute result: 30",
	}

	if len(runResp.Logs) != len(expectedLogs) {
		t.Fatalf("Expected %d logs, got %d: %v", len(expectedLogs), len(runResp.Logs), runResp.Logs)
	}

	for i, expected := range expectedLogs {
		if runResp.Logs[i] != expected {
			t.Errorf("Log at index %d: expected %q, got %q", i, expected, runResp.Logs[i])
		}
	}
}
