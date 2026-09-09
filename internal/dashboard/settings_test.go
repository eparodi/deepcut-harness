package dashboard

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"deepcut-harness/internal/config"
	"deepcut-harness/internal/store"
)

// newSettingsEnv builds a handler plus its runtime + editor so a test can
// assert on the mutated config and .env after a POST.
func newSettingsEnv(t *testing.T) (http.Handler, *config.Runtime, *config.Editor, string) {
	t.Helper()
	st, err := store.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })

	dir := t.TempDir()
	cfg := config.Default()
	rt := config.NewRuntime(cfg)
	editor := config.NewEditor(filepath.Join(dir, "config.json"), filepath.Join(dir, ".env"))
	return New(cfg, nil, st, rt, editor).srv.Handler, rt, editor, dir
}

func postForm(t *testing.T, h http.Handler, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestSettingsAddProvider(t *testing.T) {
	h, rt, _, dir := newSettingsEnv(t)

	values := url.Values{}
	values.Set("action", "add-provider")
	values.Set("name", "openrouter")
	values.Set("base_url", "https://openrouter.ai/api/v1")
	values.Set("api_key_env", "OPENROUTER_API_KEY")
	values.Set("key", "sk-test-123")
	rec := postForm(t, h, values)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}

	got := rt.Get()
	if got.Providers["openrouter"].BaseURL != "https://openrouter.ai/api/v1" {
		t.Fatalf("provider not added: %+v", got.Providers)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".env"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "OPENROUTER_API_KEY=sk-test-123") {
		t.Fatalf(".env missing key:\n%s", data)
	}
}

func TestSettingsRemoveProvider(t *testing.T) {
	h, rt, _, _ := newSettingsEnv(t)

	values := url.Values{}
	values.Set("action", "remove-provider")
	values.Set("name", "openai")
	if rec := postForm(t, h, values); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if _, ok := rt.Get().Providers["openai"]; ok {
		t.Fatal("openai provider not removed")
	}
}

func TestSettingsUpdateLLM(t *testing.T) {
	h, rt, _, _ := newSettingsEnv(t)

	values := url.Values{}
	values.Set("action", "update-llm")
	values.Set("max_retries", "7")
	values.Set("daily_token_budget", "100000")
	if rec := postForm(t, h, values); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	got := rt.Get()
	if got.LLM.MaxRetries != 7 || got.LLM.DailyTokenBudget != 100000 {
		t.Fatalf("llm not updated: %+v", got.LLM)
	}
}
