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
// assert on the mutated config and .env after a POST. It also returns the
// process CSRF token so POSTs pass the middleware.
func newSettingsEnv(t *testing.T) (http.Handler, *config.Runtime, *config.Editor, string, string) {
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
	srv := New(cfg, nil, st, rt, editor)
	return srv.srv.Handler, rt, editor, dir, srv.CSRFToken()
}

func postForm(t *testing.T, h http.Handler, csrf string, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	values.Set("csrf", csrf)
	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestSettingsAddProvider(t *testing.T) {
	h, rt, _, dir, csrf := newSettingsEnv(t)

	values := url.Values{}
	values.Set("action", "add-provider")
	values.Set("name", "openrouter")
	values.Set("base_url", "https://openrouter.ai/api/v1")
	values.Set("api_key_env", "OPENROUTER_API_KEY")
	values.Set("key", "sk-test-123")
	rec := postForm(t, h, csrf, values)
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
	h, rt, _, _, csrf := newSettingsEnv(t)

	values := url.Values{}
	values.Set("action", "remove-provider")
	values.Set("name", "openai")
	if rec := postForm(t, h, csrf, values); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if _, ok := rt.Get().Providers["openai"]; ok {
		t.Fatal("openai provider not removed")
	}
}

func TestSettingsUpdateLLM(t *testing.T) {
	h, rt, _, _, csrf := newSettingsEnv(t)

	values := url.Values{}
	values.Set("action", "update-llm")
	values.Set("max_retries", "7")
	values.Set("daily_token_budget", "100000")
	if rec := postForm(t, h, csrf, values); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	got := rt.Get()
	if got.LLM.MaxRetries != 7 || got.LLM.DailyTokenBudget != 100000 {
		t.Fatalf("llm not updated: %+v", got.LLM)
	}
}

func TestSettingsUpdateLLMFlags(t *testing.T) {
	h, rt, _, _, csrf := newSettingsEnv(t)

	values := url.Values{}
	values.Set("action", "update-llm")
	values.Set("ladder_allow_repair", "true")
	values.Set("ladder_allow_reask", "true")
	values.Set("budget_warn_fraction", "0.5")
	if rec := postForm(t, h, csrf, values); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	got := rt.Get()
	if !got.LLM.AllowRepair || !got.LLM.AllowReask {
		t.Fatalf("flags not set: %+v", got.LLM)
	}
	if got.LLM.BudgetWarnFraction != 0.5 {
		t.Fatalf("budget warn fraction = %v, want 0.5", got.LLM.BudgetWarnFraction)
	}
}

func TestSettingsUpdateLLMFlagsOffWhenUnchecked(t *testing.T) {
	h, rt, _, _, csrf := newSettingsEnv(t)

	// Default() starts AllowRepair/AllowReask true; submitting without the
	// checkbox params must persist them as false (unchecked == absent).
	values := url.Values{}
	values.Set("action", "update-llm")
	values.Set("max_retries", "3")
	if rec := postForm(t, h, csrf, values); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	got := rt.Get()
	if got.LLM.AllowRepair || got.LLM.AllowReask {
		t.Fatalf("flags should be false when unchecked: %+v", got.LLM)
	}
}

// TestSettingsAddProviderDoesNotLeakOnError pins the deep-copy fix: a
// failed save (key without api_key_env) must not mutate the live runtime.
func TestSettingsAddProviderDoesNotLeakOnError(t *testing.T) {
	h, rt, _, _, csrf := newSettingsEnv(t)

	values := url.Values{}
	values.Set("action", "add-provider")
	values.Set("name", "leaky")
	values.Set("base_url", "https://example.com/v1")
	values.Set("key", "sk-test-123") // key set, but no api_key_env → error
	if rec := postForm(t, h, csrf, values); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if _, ok := rt.Get().Providers["leaky"]; ok {
		t.Fatal("provider leaked into runtime on error path")
	}
}

func TestSettingsAddProviderInvalidEnvName(t *testing.T) {
	h, rt, _, _, csrf := newSettingsEnv(t)

	values := url.Values{}
	values.Set("action", "add-provider")
	values.Set("name", "bad")
	values.Set("base_url", "https://example.com/v1")
	values.Set("api_key_env", "NOT VALID")
	if rec := postForm(t, h, csrf, values); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if _, ok := rt.Get().Providers["bad"]; ok {
		t.Fatal("provider with invalid env name was added")
	}
}

func TestSettingsAddProviderInvalidBaseURL(t *testing.T) {
	h, rt, _, _, csrf := newSettingsEnv(t)

	values := url.Values{}
	values.Set("action", "add-provider")
	values.Set("name", "bad")
	values.Set("base_url", "not-a-url")
	if rec := postForm(t, h, csrf, values); rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	if _, ok := rt.Get().Providers["bad"]; ok {
		t.Fatal("provider with invalid base_url was added")
	}
}

func TestSettingsUpdateLLMRejectsNonNumeric(t *testing.T) {
	h, rt, _, _, csrf := newSettingsEnv(t)

	values := url.Values{}
	values.Set("action", "update-llm")
	values.Set("max_retries", "abc")
	rec := postForm(t, h, csrf, values)
	if !strings.Contains(rec.Body.String(), "max_retries: must be an integer") {
		t.Fatalf("expected numeric error, got body:\n%s", rec.Body.String())
	}
	if rt.Get().LLM.MaxRetries != 2 {
		t.Fatalf("runtime mutated on invalid input: %+v", rt.Get().LLM)
	}
}

// TestCSRFProtection pins the middleware: a state-changing POST without a
// valid token is rejected; a valid token passes.
func TestCSRFProtection(t *testing.T) {
	h, _, _, _, csrf := newSettingsEnv(t)

	// Missing token → 403.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader("action=update-llm"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("missing token status = %d, want 403", rec.Code)
	}

	// Wrong token → 403.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader("action=update-llm&csrf=wrong"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("wrong token status = %d, want 403", rec.Code)
	}

	// Correct token → passes CSRF (200; update-llm with empty fields is valid).
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader("action=update-llm&csrf="+url.QueryEscape(csrf)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("valid token status = %d, want 200", rec.Code)
	}
}
