package page

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"deepcut-harness/internal/config"
	"deepcut-harness/internal/llm"
	"deepcut-harness/internal/source"
	"deepcut-harness/internal/store"
	"deepcut-harness/internal/wizard"
)

// fakeCompleter returns scripted JSON responses in order.
type fakeCompleter struct {
	responses []string
	calls     int
}

func (f *fakeCompleter) CompleteJSON(_ context.Context, _ llm.Request, target any) error {
	if f.calls >= len(f.responses) {
		return errors.New("fake: no more responses")
	}
	resp := f.responses[f.calls]
	f.calls++
	return json.Unmarshal([]byte(resp), target)
}

func fakeLLM(responses []string) func(string) (llm.JSONCompleter, error) {
	return func(string) (llm.JSONCompleter, error) {
		return &fakeCompleter{responses: responses}, nil
	}
}

func newWizardDeps(t *testing.T, responses []string) Deps {
	t.Helper()
	st, err := store.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return Deps{
		Store:   st,
		LLM:     fakeLLM(responses),
		Wizard:  wizard.NewManager(),
		Runtime: config.NewRuntime(config.Default()),
		Source:  source.Source{},
	}
}

func postWizard(t *testing.T, d Deps, target wizard.Target, values url.Values) WizardState {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/x", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return d.WizardPost(req, target)
}

func TestWizardPostCreatesAgent(t *testing.T) {
	d := newWizardDeps(t, []string{
		`{"action":"ask","question":"What is the name?"}`,
		`{"action":"done","agent":{"name":"gardener","prompt":"refactor","temperature":0.2}}`,
	})

	start := url.Values{}
	start.Set("action", "start")
	start.Set("goal", "create a gardener")
	start.Set("provider", "deepseek")
	start.Set("model", "deepseek-v4-flash")
	state := postWizard(t, d, wizard.TargetAgent, start)
	if state.Error != "" {
		t.Fatalf("start error: %s", state.Error)
	}
	if state.SessionID == "" || state.Step.Action != wizard.ActionAsk {
		t.Fatalf("start state = %+v, want ask + session", state)
	}

	answer := url.Values{}
	answer.Set("action", "answer")
	answer.Set("session_id", state.SessionID)
	answer.Set("answer", "gardener")
	state2 := postWizard(t, d, wizard.TargetAgent, answer)
	if state2.Error != "" {
		t.Fatalf("answer error: %s", state2.Error)
	}
	if state2.Created == "" {
		t.Fatalf("answer state = %+v, want created", state2)
	}

	agents, err := d.Store.ListAgents()
	if err != nil {
		t.Fatal(err)
	}
	if len(agents) != 1 || agents[0].Name != "gardener" {
		t.Fatalf("agents = %+v", agents)
	}
	if agents[0].Provider != "deepseek" || agents[0].Model != "deepseek-v4-flash" {
		t.Fatalf("provider/model not stamped: %q/%q", agents[0].Provider, agents[0].Model)
	}
}

func TestWizardPostCreatesSkill(t *testing.T) {
	d := newWizardDeps(t, []string{
		`{"action":"done","skill":{"name":"bash","runtime":"bash","category":"stack"}}`,
	})

	start := url.Values{}
	start.Set("action", "start")
	start.Set("goal", "create a bash skill")
	start.Set("provider", "deepseek")
	start.Set("model", "deepseek-v4-flash")
	state := postWizard(t, d, wizard.TargetSkill, start)
	if state.Error != "" {
		t.Fatalf("start error: %s", state.Error)
	}
	if state.Created == "" {
		t.Fatalf("state = %+v, want created", state)
	}

	skills, err := d.Store.ListSkills()
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 || skills[0].Name != "bash" {
		t.Fatalf("skills = %+v", skills)
	}
}

func TestWizardPostUnknownAction(t *testing.T) {
	d := newWizardDeps(t, nil)
	values := url.Values{}
	values.Set("action", "bogus")
	state := postWizard(t, d, wizard.TargetAgent, values)
	if state.Error == "" {
		t.Fatal("want error for unknown action")
	}
}
