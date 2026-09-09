package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"deepcut-harness/internal/config"
	"deepcut-harness/internal/llm/openai"
	"deepcut-harness/internal/source"
	"deepcut-harness/internal/store"
	"deepcut-harness/internal/wizard"
)

// wizardCmd runs the LLM-driven creation wizard for agents and skills.
func wizardCmd(args []string, stdin io.Reader, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: harness wizard agent|skill …")
	}
	cfg, err := config.Load("config.json")
	if err != nil {
		return err
	}
	reg := openai.NewRegistry(cfg)
	st, err := store.Open(cfg.Store.Driver, cfg.Store.DSN)
	if err != nil {
		return err
	}
	defer st.Close()

	switch args[0] {
	case "agent":
		return wizardRun(wizard.TargetAgent, args[1:], reg, st, stdin, stdout)
	case "skill":
		return wizardRun(wizard.TargetSkill, args[1:], reg, st, stdin, stdout)
	default:
		return fmt.Errorf("unknown wizard target %q", args[0])
	}
}

// wizardRun drives the interactive ask/done loop for one target and
// persists the resulting definition.
func wizardRun(target wizard.Target, args []string, reg *openai.Registry, st store.Store, stdin io.Reader, stdout io.Writer) error {
	_, flags, err := parseFlags(args)
	if err != nil {
		return err
	}

	provider := flags["provider"]
	if provider == "" {
		provider = defaultProvider(reg)
	}
	completer, err := reg.JSONCompleter(provider)
	if err != nil {
		return err
	}

	reader := source.Source{}
	srcContent, err := reader.Load(context.Background(), flags["source"])
	if err != nil {
		return fmt.Errorf("source: %w", err)
	}
	var sources []string
	if srcContent != "" {
		sources = []string{srcContent}
	}

	in := bufio.NewReader(stdin)
	goal := strings.TrimSpace(flags["goal"])
	if goal == "" {
		fmt.Fprint(stdout, "What do you want to create? ")
		line, rerr := in.ReadString('\n')
		if rerr != nil && rerr != io.EOF {
			return rerr
		}
		goal = strings.TrimSpace(line)
	}

	session := wizard.New(target, completer, provider, flags["model"], goal, sources)
	ctx := context.Background()

	step, err := session.Next(ctx)
	for {
		if err != nil {
			return err
		}
		switch step.Action {
		case wizard.ActionAsk:
			fmt.Fprintf(stdout, "%s\n> ", step.Question)
			line, rerr := in.ReadString('\n')
			if rerr != nil && rerr != io.EOF {
				return rerr
			}
			step, err = session.Answer(ctx, strings.TrimSpace(line))
		case wizard.ActionDone:
			if verr := step.Validate(); verr != nil {
				return verr
			}
			return persistWizard(target, step, st, stdout)
		default:
			return fmt.Errorf("wizard: unknown action %q", step.Action)
		}
	}
}

// defaultProvider returns the alphabetically-first configured provider.
func defaultProvider(reg *openai.Registry) string {
	names := reg.Names()
	if len(names) > 0 {
		return names[0]
	}
	return "deepseek"
}

// persistWizard validates (already done by the caller) and stores the
// wizard's definition.
func persistWizard(target wizard.Target, step wizard.Step, st store.Store, stdout io.Writer) error {
	switch target {
	case wizard.TargetAgent:
		if step.Agent == nil {
			return fmt.Errorf("wizard: agent definition missing")
		}
		created, err := st.CreateAgent(*step.Agent)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "created agent %q (%s)\n", created.Name, created.ID)
		return nil
	case wizard.TargetSkill:
		if step.Skill == nil {
			return fmt.Errorf("wizard: skill definition missing")
		}
		created, err := st.CreateSkill(*step.Skill)
		if err != nil {
			return err
		}
		fmt.Fprintf(stdout, "created skill %q (%s)\n", created.Name, created.ID)
		return nil
	default:
		return fmt.Errorf("wizard: unknown target %q", target)
	}
}
