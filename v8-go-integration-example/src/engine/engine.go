package engine

// ScriptRequest represents the payload for executing Javascript code.
type ScriptRequest struct {
	Script string `json:"script"`
}

// ScriptResponse contains the output of execution, along with any captured logs.
type ScriptResponse struct {
	Success    bool     `json:"success"`
	Result     string   `json:"result,omitempty"`
	Error      string   `json:"error,omitempty"`
	Logs       []string `json:"logs"`
	DurationMs int64    `json:"duration_ms"`
}

// ScriptRunner defines an abstraction for executing Javascript code.
type ScriptRunner interface {
	Run(req ScriptRequest) ScriptResponse
}
