package v8_test

import (
	"strings"
	"testing"

	"v8-go-integration/src/engine"
	"v8-go-integration/src/v8"
)

func TestV8Runner_Run_BasicMath(t *testing.T) {
	runner := v8.NewV8Runner()

	req := engine.ScriptRequest{
		Script: `const a = 10; const b = 20; console.log("Variables initialized"); a + b;`,
	}

	resp := runner.Run(req)

	if !resp.Success {
		t.Fatalf("Expected success, got error: %s", resp.Error)
	}

	if resp.Result != "30" {
		t.Errorf("Expected result '30', got '%s'", resp.Result)
	}

	if len(resp.Logs) != 1 || resp.Logs[0] != "Variables initialized" {
		t.Errorf("Expected logs ['Variables initialized'], got %v", resp.Logs)
	}
}

func TestV8Runner_Run_GoComputeCallback(t *testing.T) {
	runner := v8.NewV8Runner()

	req := engine.ScriptRequest{
		Script: `console.log("Calling goCompute"); const val = goCompute(4, 5); console.log("Received result"); val;`,
	}

	resp := runner.Run(req)

	if !resp.Success {
		t.Fatalf("Expected success, got error: %s", resp.Error)
	}

	if resp.Result != "20" {
		t.Errorf("Expected result '20', got '%s'", resp.Result)
	}

	expectedLogs := []string{
		"Calling goCompute",
		"[Go Callback] goCompute: Multiplying 4 * 5 in Go -> 20",
		"Received result",
	}

	if len(resp.Logs) != 3 {
		t.Fatalf("Expected 3 logs, got %d: %v", len(resp.Logs), resp.Logs)
	}

	for i, expected := range expectedLogs {
		if resp.Logs[i] != expected {
			t.Errorf("At index %d: expected log '%s', got '%s'", i, expected, resp.Logs[i])
		}
	}
}

func TestV8Runner_Run_GoFetchCallback(t *testing.T) {
	runner := v8.NewV8Runner()

	req := engine.ScriptRequest{
		Script: `const data = JSON.parse(goFetch("http://example.com")); data.data.items[0];`,
	}

	resp := runner.Run(req)

	if !resp.Success {
		t.Fatalf("Expected success, got error: %s", resp.Error)
	}

	if resp.Result != "Go" {
		t.Errorf("Expected result 'Go', got '%s'", resp.Result)
	}

	if len(resp.Logs) != 1 || !strings.Contains(resp.Logs[0], "goFetch: Simulating Go HTTP") {
		t.Errorf("Expected simulated fetch log, got %v", resp.Logs)
	}
}

func TestV8Runner_Run_JSError(t *testing.T) {
	runner := v8.NewV8Runner()

	req := engine.ScriptRequest{
		Script: `function fail() { nonExistent(); } fail();`,
	}

	resp := runner.Run(req)

	if resp.Success {
		t.Fatal("Expected run to fail, but it succeeded")
	}

	if !strings.Contains(resp.Error, "ReferenceError: nonExistent is not defined") {
		t.Errorf("Expected ReferenceError, got: %s", resp.Error)
	}

	if !strings.Contains(resp.Error, "Location: user_script.js") {
		t.Errorf("Expected location user_script.js, got: %s", resp.Error)
	}

	if !strings.Contains(resp.Error, "Stack: ") {
		t.Errorf("Expected stack trace, got: %s", resp.Error)
	}
}
