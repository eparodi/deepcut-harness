# LLM Provider Layer (Draft)

**Feature slug:** `llm-provider-layer`
**Status:** Approved
**Owner:** PM
**Created:** 2026-09-08

## Requirements

### User stories

- As a developer, I can configure multiple LLM providers by name
  (`deepseek`, `openai`, `ollama`, `openrouter`, …) with just a base URL,
  an API-key env var, and a model.
- As a developer, I can call any configured provider through ONE Go
  interface, so the rest of the codebase never hardcodes a provider.
- As a developer, I can ask for a JSON-mode completion and get strict,
  valid JSON back (with repair/retry on malformed output).
- As a developer, I have retries/backoff and a circuit breaker so a flaky
  provider fails fast instead of hanging.
- As a developer, I have a daily token/cost budget so a runaway agent
  can't burn unbounded spend.

### Acceptance criteria

- `internal/llm` exposes a `Provider` interface (the port).
- A single OpenAI-compatible HTTP client implements it, parameterized by
  base URL + key + model — covering DeepSeek, OpenAI, Groq, Together,
  OpenRouter, Mistral, Ollama/vLLM, and Anthropic/Gemini compat endpoints.
- Config carries a `providers` map (name → base_url, api_key_env) and the
  reliability settings (retries, breaker, budget).
- JSON mode is pinned by a table test against a real payload fixture
  (success, malformed JSON, empty JSON-mode content, 429).
- Malformed responses follow the ladder: strict parse → cheap repair →
  one-shot re-ask → HOLD.
- Retry ONLY 429/500/503 + network errors; never 400/401/402/422.
- `make build` / `make vet` / `make test` green.

## Explicit Non-Goals

- No streaming (non-streaming only this step; streaming is a follow-up if
  the wizard needs it).
- No function/tool calling, no vision, no embeddings/vector memory.
- No native Anthropic `/v1/messages` or Gemini `/v1beta` adapters — they
  are reachable via their OpenAI-compat endpoints, and native adapters
  are future adapters behind the same `Provider` interface.
- No wizard (that's Step B).
- No config editing via UI — a `/settings` page to view/edit providers and
  LLM settings is a separate follow-up step (this step is the provider
  client + file-based config).

## Design

### The port

```go
// Provider is one LLM backend. The rest of the codebase depends only on
// this interface, never on a concrete provider.
type Provider interface {
	Complete(ctx context.Context, req Request) (Response, error)
}

type Message struct {
	Role    string // "system" | "user" | "assistant"
	Content string
}

type Request struct {
	Model       string    // resolved per call, or a default
	Messages    []Message
	JSON        bool      // request JSON mode
	Temperature float64
	MaxTokens   int
}

type Response struct {
	Content string
	Usage   Usage
}

type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}
```

### The adapter (covers the most providers)

One `Client` implements `Provider` over `POST {base_url}/chat/completions`
(OpenAI-compatible), with `Authorization: Bearer <key>`. A provider is
selected by name from the config registry; changing base URL + key + model
is what reaches DeepSeek, OpenAI, Groq, Together, OpenRouter, Mistral,
Ollama/vLLM, and the Anthropic/Gemini compat endpoints.

### JSON mode + malformed-response ladder

`Request.JSON` sets `response_format: {"type":"json_object"}` and appends
"Respond with only valid JSON" + a format example to the prompt. The
ladder (per the `ai-engineer` skill):

1. Strict parse (unknown fields rejected, enum checks).
2. Cheap local repair (strip fences, extract first balanced object).
3. One-shot re-ask with a compact "your last response was invalid" suffix.
4. HOLD + warning (always the final fallback).

### Resilience

- Retry only 429/500/503 and network timeouts, exponential backoff + full
  jitter, honoring `Retry-After`.
- Circuit breaker: after N consecutive failures, skip calls and fail fast,
  probe after a cooldown, close on success.
- Every request bounded by a context timeout.
- Stable prompt prefix first, variable data last (maximize KV-cache hits).

### Cost budget

A daily token ceiling (config value; 0 = unlimited). The enforcement
mechanism lives here; the budget NUMBER is a config decision. Log
cache-hit/miss tokens on every call.

### Config

```go
type Config struct {
	// … existing dashboard + store …
	LLM       LLM                      `json:"llm"`
	Providers map[string]Provider      `json:"providers"`
}

type LLM struct {
	MaxRetries         int     `json:"max_retries"`
	RetryBaseMS        int     `json:"retry_base_ms"`
	BreakerFailures    int     `json:"breaker_consecutive_failures"`
	BreakerCooldownS   int     `json:"breaker_cooldown_s"`
	AllowRepair        bool    `json:"ladder_allow_repair"`
	AllowReask         bool    `json:"ladder_allow_reask"`
	DailyTokenBudget   int     `json:"daily_token_budget"`
	BudgetWarnFraction float64 `json:"budget_warn_fraction"`
}

type Provider struct {
	BaseURL   string `json:"base_url"`
	APIKeyEnv string `json:"api_key_env"`
}
```

API keys are read from the environment via `os.Getenv(api_key_env)`, never
committed.

### Packages

```
internal/llm            // Provider interface, Message/Request/Response, Usage
internal/llm/openai     // the OpenAI-compatible Client (+ registry of providers)
internal/config          // llm + providers config
```

## Task Checklist

- [ ] `internal/llm` — `Provider`, `Message`, `Request`, `Response`, `Usage`
- [ ] `internal/llm/openai` — the OpenAI-compatible `Client` (base URL + key + model)
- [ ] Config: `llm` + `providers` maps (defaults + validation)
- [ ] JSON-mode request building + strict parsing
- [ ] Malformed-response ladder (repair / re-ask / HOLD)
- [ ] Retry + backoff (429/500/503/network) + circuit breaker
- [ ] Cost budget + token telemetry
- [ ] Table-driven tests with pinned payload fixtures (success, malformed, empty JSON, 429)
- [ ] `config.json.example` + `.env.example` for the default providers (godotenv loads `.env`)
- [ ] `make build` / `make vet` / `make test` green

## Decisions (resolved at the gate)

- **API-key loading:** `godotenv` (loads a gitignored `.env` file at startup).
- **Default providers:** `deepseek`, `openai`, `ollama` pre-listed in
  `config.json.example`; OpenRouter documented as the "one key, many
  models" path.
- **Streaming:** deferred (non-streaming only this step).
