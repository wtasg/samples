package v8

import (
	"fmt"
	"strings"
	"time"

	"rogchap.com/v8go"
	"v8-go-integration/src/engine"
)

// V8Runner implements the engine.ScriptRunner interface using rogchap.com/v8go.
type V8Runner struct{}

// NewV8Runner instantiates a new runner.
func NewV8Runner() *V8Runner {
	return &V8Runner{}
}

// Run executes the Javascript code inside a sandboxed V8 Isolate/Context.
func (r *V8Runner) Run(req engine.ScriptRequest) engine.ScriptResponse {
	startTime := time.Now()
	var logs []string

	// Create new isolate for sandbox isolation per execution.
	iso := v8go.NewIsolate()
	defer iso.Dispose()

	// Create global object template.
	global := v8go.NewObjectTemplate(iso)

	// Create __go_log__ function template.
	goLogFn := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		var msg []string
		for _, arg := range args {
			msg = append(msg, arg.String())
		}
		logs = append(logs, strings.Join(msg, " "))
		return nil
	})
	if err := global.Set("__go_log__", goLogFn); err != nil {
		return engine.ScriptResponse{
			Success:    false,
			Error:      fmt.Sprintf("Failed to bind __go_log__ callback: %v", err),
			DurationMs: time.Since(startTime).Milliseconds(),
		}
	}

	// Predefined Go Callbacks to demonstrate bidirectional integration

	// 1. goCompute(x, y) - Multiply numbers in Go
	goComputeFn := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 2 {
			val, _ := v8go.NewValue(iso, "Error: goCompute requires exactly 2 numeric arguments")
			return val
		}
		x := args[0].Integer()
		y := args[1].Integer()
		result := x * y
		logs = append(logs, fmt.Sprintf("[Go Callback] goCompute: Multiplying %d * %d in Go -> %d", x, y, result))
		val, _ := v8go.NewValue(iso, result)
		return val
	})
	if err := global.Set("goCompute", goComputeFn); err != nil {
		return engine.ScriptResponse{
			Success:    false,
			Error:      fmt.Sprintf("Failed to bind goCompute: %v", err),
			DurationMs: time.Since(startTime).Milliseconds(),
		}
	}

	// 2. goFetch(url) - Simulate a secure Go-side HTTP Fetch
	goFetchFn := v8go.NewFunctionTemplate(iso, func(info *v8go.FunctionCallbackInfo) *v8go.Value {
		args := info.Args()
		if len(args) < 1 {
			val, _ := v8go.NewValue(iso, "Error: goFetch requires a URL string argument")
			return val
		}
		url := args[0].String()
		logs = append(logs, fmt.Sprintf("[Go Callback] goFetch: Simulating Go HTTP client fetch for URL: %s", url))
		mockResponse := fmt.Sprintf(`{"status":"success","url":"%s","data":{"items":["Go","V8","Integration"]}}`, url)
		val, _ := v8go.NewValue(iso, mockResponse)
		return val
	})
	if err := global.Set("goFetch", goFetchFn); err != nil {
		return engine.ScriptResponse{
			Success:    false,
			Error:      fmt.Sprintf("Failed to bind goFetch: %v", err),
			DurationMs: time.Since(startTime).Milliseconds(),
		}
	}

	// Create V8 execution context with our global template
	ctx := v8go.NewContext(iso, global)
	defer ctx.Close()

	// Redefine standard console.log/warn/error inside JavaScript to call __go_log__
	bootstrap := `
		if (typeof console === 'undefined') {
			console = {
				log: (...args) => __go_log__(...args.map(String)),
				warn: (...args) => __go_log__('[WARN]', ...args.map(String)),
				error: (...args) => __go_log__('[ERROR]', ...args.map(String))
			};
		} else {
			console.log = (...args) => __go_log__(...args.map(String));
			console.warn = (...args) => __go_log__('[WARN]', ...args.map(String));
			console.error = (...args) => __go_log__('[ERROR]', ...args.map(String));
		}
	`
	if _, err := ctx.RunScript(bootstrap, "bootstrap.js"); err != nil {
		return engine.ScriptResponse{
			Success:    false,
			Error:      fmt.Sprintf("Failed to bootstrap console.log: %v", err),
			DurationMs: time.Since(startTime).Milliseconds(),
		}
	}

	// Run script
	val, err := ctx.RunScript(req.Script, "user_script.js")
	durationMs := time.Since(startTime).Milliseconds()

	if err != nil {
		var errMsg string
		if jsErr, ok := err.(*v8go.JSError); ok {
			errMsg = fmt.Sprintf("JavaScript Error: %s\nLocation: %s\nStack: %s", jsErr.Message, jsErr.Location, jsErr.StackTrace)
		} else {
			errMsg = err.Error()
		}
		return engine.ScriptResponse{
			Success:    false,
			Error:      errMsg,
			Logs:       logs,
			DurationMs: durationMs,
		}
	}

	return engine.ScriptResponse{
		Success:    true,
		Result:     val.String(),
		Logs:       logs,
		DurationMs: durationMs,
	}
}
