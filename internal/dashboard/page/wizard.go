package page

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"deepcut-harness/internal/id"
	"deepcut-harness/internal/source"
	"deepcut-harness/internal/wizard"
)

// WizardState is the wizard form state for a page (agents or skills).
type WizardState struct {
	Step      wizard.Step
	SessionID string
	Target    wizard.Target
	Created   string
	Error     string
	Providers []string
	Models    []string // the default provider's models (initial model select)
}

// WizardProviders returns the configured provider names, sorted.
func (d Deps) WizardProviders() []string {
	cfg := d.Runtime.Get()
	names := make([]string, 0, len(cfg.Providers))
	for name := range cfg.Providers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// WizardModels returns the models of the first (alphabetically) provider.
func (d Deps) WizardModels() []string {
	names := d.WizardProviders()
	if len(names) == 0 {
		return nil
	}
	return d.Runtime.Get().Providers[names[0]].Models
}

// WizardPost handles a wizard form POST (action=start|answer) for target.
func (d Deps) WizardPost(r *http.Request, target wizard.Target) WizardState {
	state := WizardState{Target: target, Providers: d.WizardProviders(), Models: d.WizardModels()}
	if err := r.ParseForm(); err != nil {
		state.Error = "invalid form"
		return state
	}
	switch r.FormValue("action") {
	case "start":
		return d.wizardStart(r, target, state)
	case "answer":
		return d.wizardAnswer(r, target, state)
	default:
		state.Error = "unknown action"
		return state
	}
}

func (d Deps) wizardStart(r *http.Request, target wizard.Target, state WizardState) WizardState {
	provider := strings.TrimSpace(r.FormValue("provider"))
	if provider == "" {
		provider = firstProvider(state.Providers)
	}
	completer, err := d.LLM(provider)
	if err != nil {
		state.Error = err.Error()
		return state
	}
	sources, err := readSources(d.Source, r.Context(), r.FormValue("source"))
	if err != nil {
		state.Error = err.Error()
		return state
	}
	session := wizard.New(target, completer, provider, strings.TrimSpace(r.FormValue("model")), strings.TrimSpace(r.FormValue("goal")), sources)
	step, err := session.Next(r.Context())
	if err != nil {
		state.Error = err.Error()
		return state
	}
	if step.Action == wizard.ActionDone {
		return d.wizardFinish(target, step, state)
	}
	sid := id.New()
	d.Wizard.Put(sid, session)
	state.Step = step
	state.SessionID = sid
	return state
}

func (d Deps) wizardAnswer(r *http.Request, target wizard.Target, state WizardState) WizardState {
	sid := strings.TrimSpace(r.FormValue("session_id"))
	session, ok := d.Wizard.Get(sid)
	if !ok {
		state.Error = "wizard session not found — start again"
		return state
	}
	step, err := session.Answer(r.Context(), strings.TrimSpace(r.FormValue("answer")))
	if err != nil {
		state.Error = err.Error()
		return state
	}
	if step.Action == wizard.ActionDone {
		d.Wizard.Delete(sid)
		return d.wizardFinish(session.Target(), step, state)
	}
	state.Step = step
	state.SessionID = sid
	state.Target = session.Target()
	return state
}

func (d Deps) wizardFinish(target wizard.Target, step wizard.Step, state WizardState) WizardState {
	if err := step.Validate(); err != nil {
		state.Error = err.Error()
		return state
	}
	switch target {
	case wizard.TargetAgent:
		if step.Agent == nil {
			state.Error = "wizard: agent definition missing"
			return state
		}
		created, err := d.Store.CreateAgent(*step.Agent)
		if err != nil {
			state.Error = err.Error()
			return state
		}
		state.Created = created.Name
		return state
	case wizard.TargetSkill:
		if step.Skill == nil {
			state.Error = "wizard: skill definition missing"
			return state
		}
		created, err := d.Store.CreateSkill(*step.Skill)
		if err != nil {
			state.Error = err.Error()
			return state
		}
		state.Created = created.Name
		return state
	default:
		state.Error = "wizard: unknown target"
		return state
	}
}

func firstProvider(names []string) string {
	if len(names) > 0 {
		return names[0]
	}
	return "deepseek"
}

func readSources(reader source.Source, ctx context.Context, src string) ([]string, error) {
	src = strings.TrimSpace(src)
	if src == "" {
		return nil, nil
	}
	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		content, err := reader.Fetch(ctx, src)
		if err != nil {
			return nil, fmt.Errorf("source: %w", err)
		}
		return []string{content}, nil
	}
	content, err := reader.Read(ctx, src)
	if err != nil {
		return nil, fmt.Errorf("source: %w", err)
	}
	return []string{content}, nil
}
