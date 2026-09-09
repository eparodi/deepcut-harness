// Package llm defines the provider-agnostic LLM port: the Provider
// interface plus the message/request/response types. The rest of the
// codebase depends only on this package, never on a concrete provider —
// that is what lets one interface cover many backends.
package llm

import "context"

// Provider is one LLM backend. Implementations (the OpenAI-compatible
// client, and any future native adapters) satisfy this interface.
type Provider interface {
	Complete(ctx context.Context, req Request) (Response, error)
}

// Message is one chat message.
type Message struct {
	Role    string // "system", "user", or "assistant"
	Content string
}

// Request is a single completion request.
type Request struct {
	Model       string
	Messages    []Message
	JSON        bool // request JSON-mode output
	Temperature float64
	MaxTokens   int
}

// Response is a single completion response.
type Response struct {
	Content string
	Usage   Usage
}

// Usage reports token counts for cost telemetry.
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	// CacheHitTokens / CacheMissTokens are prompt-cache telemetry
	// (DeepSeek KV-cache and similar); 0 when the provider doesn't
	// report them.
	CacheHitTokens  int
	CacheMissTokens int
}
