// Package openai implements the OpenAI-compatible Provider: a single
// HTTP client that reaches OpenAI, DeepSeek, Groq, Together, OpenRouter,
// Mistral, Ollama/vLLM, and the Anthropic/Gemini compat endpoints by
// varying the base URL, API key, and model.
package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"deepcut-harness/internal/llm"
)

// ErrCircuitOpen is returned when the circuit breaker has tripped.
var ErrCircuitOpen = errors.New("openai: circuit open")

// Config configures one provider.
type Config struct {
	BaseURL string
	APIKey  string
	// DefaultModel is used when a Request leaves Model empty.
	DefaultModel string
}

// Options carries the reliability knobs (from config).
type Options struct {
	MaxRetries         int
	RetryBase          time.Duration
	BreakerFailures    int
	BreakerCooldown    time.Duration
	AllowRepair        bool
	AllowReask         bool
	DailyTokenBudget   int
	BudgetWarnFraction float64
	Logger             *slog.Logger
}

// Client is an OpenAI-compatible llm.Provider.
type Client struct {
	cfg  Config
	opts Options
	hc   *http.Client
	log  *slog.Logger

	mu       sync.Mutex
	failures int
	lastFail time.Time

	budgetMu        sync.Mutex
	budgetDay       string
	budgetUsed      int
	budgetWarnedDay string
}

// New returns a configured Client. Zero-value Options get safe defaults.
func New(cfg Config, opts Options) *Client {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.RetryBase == 0 {
		opts.RetryBase = 250 * time.Millisecond
	}
	if opts.MaxRetries == 0 {
		opts.MaxRetries = 2
	}
	if opts.BreakerFailures == 0 {
		opts.BreakerFailures = 5
	}
	if opts.BreakerCooldown == 0 {
		opts.BreakerCooldown = 60 * time.Second
	}
	if opts.BudgetWarnFraction == 0 {
		opts.BudgetWarnFraction = 0.8
	}
	return &Client{
		cfg:  cfg,
		opts: opts,
		hc:   &http.Client{Timeout: 5 * time.Minute},
		log:  opts.Logger,
	}
}

// Complete runs one completion: breaker check, budget veto, then the
// HTTP call with retries, then budget accounting.
func (c *Client) Complete(ctx context.Context, req llm.Request) (llm.Response, error) {
	if err := c.checkBreaker(); err != nil {
		return llm.Response{}, err
	}
	if err := c.checkBudget(); err != nil {
		return llm.Response{}, err
	}
	resp, err := c.doWithRetries(ctx, req)
	if err != nil {
		return llm.Response{}, err
	}
	c.account(resp.Usage)
	c.logUsage(req, resp.Usage)
	return resp, nil
}

// CompleteJSON asks for JSON-mode output and unmarshals it into target,
// applying the malformed-response ladder (repair → one-shot re-ask).
func (c *Client) CompleteJSON(ctx context.Context, req llm.Request, target any) error {
	req.JSON = true
	resp, err := c.Complete(ctx, req)
	if err != nil {
		return err
	}
	if json.Unmarshal([]byte(resp.Content), target) == nil {
		return nil
	}
	if c.opts.AllowRepair {
		if repaired, ok := repairJSON(resp.Content); ok {
			if json.Unmarshal([]byte(repaired), target) == nil {
				return nil
			}
		}
	}
	if c.opts.AllowReask {
		req.Messages = append(req.Messages,
			llm.Message{Role: "assistant", Content: resp.Content},
			llm.Message{Role: "user", Content: "Your previous response was invalid. Return ONLY a valid JSON object."},
		)
		resp2, err := c.Complete(ctx, req)
		if err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(resp2.Content), target); err != nil {
			return fmt.Errorf("openai: malformed JSON after re-ask: %w", err)
		}
		return nil
	}
	return fmt.Errorf("openai: malformed JSON: %w", errors.New("content is not a JSON object"))
}

func (c *Client) doWithRetries(ctx context.Context, req llm.Request) (llm.Response, error) {
	for attempt := 0; ; attempt++ {
		resp, err := c.doOnce(ctx, req)
		if err == nil {
			c.noteSuccess()
			return resp, nil
		}
		if !retryable(err) || attempt >= c.opts.MaxRetries {
			c.noteFailure()
			return llm.Response{}, err
		}
		select {
		case <-time.After(c.backoff(attempt)):
		case <-ctx.Done():
			return llm.Response{}, ctx.Err()
		}
	}
}

func (c *Client) doOnce(ctx context.Context, req llm.Request) (llm.Response, error) {
	body, err := c.buildBody(req)
	if err != nil {
		return llm.Response{}, err
	}
	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return llm.Response{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}
	httpResp, err := c.hc.Do(httpReq)
	if err != nil {
		return llm.Response{}, err
	}
	defer httpResp.Body.Close()
	data, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return llm.Response{}, err
	}
	if httpResp.StatusCode != http.StatusOK {
		return llm.Response{}, &HTTPError{Status: httpResp.StatusCode, Body: truncate(string(data))}
	}
	return c.parseResponse(data)
}

func (c *Client) buildBody(req llm.Request) ([]byte, error) {
	messages := append([]llm.Message(nil), req.Messages...)
	if req.JSON && len(messages) > 0 {
		messages[len(messages)-1].Content += "\n\nRespond with only valid JSON."
	}
	model := req.Model
	if model == "" {
		model = c.cfg.DefaultModel
	}
	api := apiRequest{
		Model:    model,
		Messages: make([]apiMessage, len(messages)),
	}
	for i, m := range messages {
		api.Messages[i] = apiMessage{Role: m.Role, Content: m.Content}
	}
	t := req.Temperature
	api.Temperature = &t
	if req.MaxTokens != 0 {
		api.MaxTokens = req.MaxTokens
	}
	if req.JSON {
		api.ResponseFormat = &apiResponseFormat{Type: "json_object"}
	}
	return json.Marshal(api)
}

func (c *Client) parseResponse(data []byte) (llm.Response, error) {
	var ar apiResponse
	if err := json.Unmarshal(data, &ar); err != nil {
		return llm.Response{}, fmt.Errorf("openai: parse response: %w", err)
	}
	if len(ar.Choices) == 0 {
		return llm.Response{}, errors.New("openai: response has no choices")
	}
	return llm.Response{
		Content: ar.Choices[0].Message.Content,
		Usage: llm.Usage{
			PromptTokens:     ar.Usage.PromptTokens,
			CompletionTokens: ar.Usage.CompletionTokens,
			TotalTokens:      ar.Usage.TotalTokens,
			CacheHitTokens:   ar.Usage.PromptCacheHitTokens,
			CacheMissTokens:  ar.Usage.PromptCacheMissTokens,
		},
	}, nil
}

func (c *Client) backoff(attempt int) time.Duration {
	max := c.opts.RetryBase * time.Duration(1<<uint(attempt))
	if max <= 0 {
		return c.opts.RetryBase
	}
	return time.Duration(rand.Int63n(int64(max)))
}

func (c *Client) checkBreaker() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.failures < c.opts.BreakerFailures {
		return nil
	}
	if time.Since(c.lastFail) > c.opts.BreakerCooldown {
		return nil
	}
	return ErrCircuitOpen
}

func (c *Client) noteSuccess() {
	c.mu.Lock()
	c.failures = 0
	c.mu.Unlock()
}

func (c *Client) noteFailure() {
	c.mu.Lock()
	c.failures++
	c.lastFail = time.Now()
	c.mu.Unlock()
}

func (c *Client) checkBudget() error {
	c.budgetMu.Lock()
	defer c.budgetMu.Unlock()
	if c.opts.DailyTokenBudget <= 0 {
		return nil
	}
	c.rollBudgetLocked()
	if c.budgetUsed >= c.opts.DailyTokenBudget {
		return fmt.Errorf("openai: daily token budget exceeded (%d/%d)", c.budgetUsed, c.opts.DailyTokenBudget)
	}
	return nil
}

func (c *Client) account(usage llm.Usage) {
	c.budgetMu.Lock()
	defer c.budgetMu.Unlock()
	c.rollBudgetLocked()
	c.budgetUsed += usage.TotalTokens
	c.warnBudgetLocked()
}

func (c *Client) rollBudgetLocked() {
	today := time.Now().UTC().Format("2006-01-02")
	if c.budgetDay != today {
		c.budgetDay = today
		c.budgetUsed = 0
	}
}

// warnBudgetLocked logs one warning per UTC day once the used token count
// crosses budget_warn_fraction of the daily budget. It runs under budgetMu
// (called from account); the warn-once guard is the budget day.
func (c *Client) warnBudgetLocked() {
	if c.opts.DailyTokenBudget <= 0 || c.opts.BudgetWarnFraction <= 0 {
		return
	}
	if c.budgetWarnedDay == c.budgetDay {
		return
	}
	threshold := int(float64(c.opts.DailyTokenBudget) * c.opts.BudgetWarnFraction)
	if c.budgetUsed >= threshold {
		c.budgetWarnedDay = c.budgetDay
		c.log.Warn("llm daily token budget warning",
			"used", c.budgetUsed,
			"budget", c.opts.DailyTokenBudget,
			"fraction", c.opts.BudgetWarnFraction,
		)
	}
}

func (c *Client) logUsage(req llm.Request, u llm.Usage) {
	c.log.Info("llm completion",
		"model", req.Model,
		"prompt_tokens", u.PromptTokens,
		"completion_tokens", u.CompletionTokens,
		"cache_hit_tokens", u.CacheHitTokens,
		"cache_miss_tokens", u.CacheMissTokens,
	)
}

// HTTPError is a non-200 provider response.
type HTTPError struct {
	Status int
	Body   string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("openai: status %d: %s", e.Status, e.Body)
}

func retryable(err error) bool {
	var he *HTTPError
	if errors.As(err, &he) {
		return he.Status == http.StatusTooManyRequests ||
			he.Status == http.StatusInternalServerError ||
			he.Status == http.StatusServiceUnavailable
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true // network / truncated-body errors are transient
}

func truncate(s string) string {
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}

func repairJSON(s string) (string, bool) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || end <= start {
		return "", false
	}
	return s[start : end+1], true
}

// apiRequest is the OpenAI-compatible request body.
type apiRequest struct {
	Model          string             `json:"model"`
	Messages       []apiMessage       `json:"messages"`
	Temperature    *float64           `json:"temperature,omitempty"`
	MaxTokens      int                `json:"max_tokens,omitempty"`
	ResponseFormat *apiResponseFormat `json:"response_format,omitempty"`
}

type apiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type apiResponseFormat struct {
	Type string `json:"type"`
}

type apiResponse struct {
	Choices []struct {
		Message apiMessage `json:"message"`
	} `json:"choices"`
	Usage apiUsage `json:"usage"`
}

type apiUsage struct {
	PromptTokens          int `json:"prompt_tokens"`
	CompletionTokens      int `json:"completion_tokens"`
	TotalTokens           int `json:"total_tokens"`
	PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens"`
	PromptCacheMissTokens int `json:"prompt_cache_miss_tokens"`
}
