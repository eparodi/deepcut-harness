package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"deepcut-harness/internal/llm"
)

func TestCompleteSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"hi"}}],"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15}}`)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL}, Options{})
	resp, err := c.Complete(context.Background(), llm.Request{Messages: []llm.Message{{Role: "user", Content: "hello"}}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "hi" {
		t.Fatalf("content = %q", resp.Content)
	}
	if resp.Usage.TotalTokens != 15 {
		t.Fatalf("total tokens = %d", resp.Usage.TotalTokens)
	}
}

func TestCompleteJSONModeSetsResponseFormat(t *testing.T) {
	var gotFormat bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["response_format"]; ok {
			gotFormat = true
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"name\":\"x\"}"}}],"usage":{}}`)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL}, Options{})
	var out map[string]string
	if err := c.CompleteJSON(context.Background(), llm.Request{Messages: []llm.Message{{Role: "user", Content: "make json"}}}, &out); err != nil {
		t.Fatal(err)
	}
	if !gotFormat {
		t.Error("response_format not set for JSON mode")
	}
	if out["name"] != "x" {
		t.Fatalf("out = %+v", out)
	}
}

func TestCompleteRetry429(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"choices":[{"message":{"content":"ok"}}],"usage":{}}`)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL}, Options{RetryBase: time.Millisecond, MaxRetries: 2})
	if _, err := c.Complete(context.Background(), llm.Request{}); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2 (one retry)", calls)
	}
}

func TestCompleteNoRetry400(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL}, Options{RetryBase: time.Millisecond, MaxRetries: 5})
	_, err := c.Complete(context.Background(), llm.Request{})
	var he *HTTPError
	if !errors.As(err, &he) || he.Status != 400 {
		t.Fatalf("err = %v, want HTTPError 400", err)
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1 (no retry on 400)", calls)
	}
}

func TestCompleteJSONRepairsFencedPrefix(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"choices":[{"message":{"content":"Sure! {\"name\":\"gardener\"}"}}],"usage":{}}`)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL}, Options{AllowRepair: true, AllowReask: false})
	var out map[string]string
	if err := c.CompleteJSON(context.Background(), llm.Request{Messages: []llm.Message{{Role: "user", Content: "x"}}}, &out); err != nil {
		t.Fatal(err)
	}
	if out["name"] != "gardener" {
		t.Fatalf("out = %+v", out)
	}
}

func TestCompleteJSONReask(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
		if calls == 1 {
			fmt.Fprint(w, `{"choices":[{"message":{"content":"not json at all"}}],"usage":{}}`)
		} else {
			fmt.Fprint(w, `{"choices":[{"message":{"content":"{\"name\":\"x\"}"}}],"usage":{}}`)
		}
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL}, Options{AllowRepair: true, AllowReask: true, RetryBase: time.Millisecond})
	var out map[string]string
	if err := c.CompleteJSON(context.Background(), llm.Request{Messages: []llm.Message{{Role: "user", Content: "x"}}}, &out); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("calls = %d, want 2 (one re-ask)", calls)
	}
	if out["name"] != "x" {
		t.Fatalf("out = %+v", out)
	}
}
