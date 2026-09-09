package cli

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"deepcut-harness/internal/agent"
	"deepcut-harness/internal/skill"
	"deepcut-harness/internal/store"
)

func agentCmd(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: harness agent create|list|get|remove|attach|detach|search …")
	}
	st, err := openStore()
	if err != nil {
		return err
	}
	defer st.Close()

	switch args[0] {
	case "create":
		return agentCreate(args[1:], st, stdout)
	case "list":
		return agentList(st, stdout)
	case "get":
		return agentGet(args[1:], st, stdout)
	case "remove":
		return agentRemove(args[1:], st, stdout)
	case "attach":
		return agentAttach(args[1:], st, stdout)
	case "detach":
		return agentDetach(args[1:], st, stdout)
	case "search":
		return agentSearch(args[1:], st, stdout)
	default:
		return fmt.Errorf("unknown agent subcommand %q", args[0])
	}
}

func agentCreate(args []string, st store.Store, stdout io.Writer) error {
	pos, flags, err := parseFlags(args)
	if err != nil {
		return err
	}
	if len(pos) < 1 {
		return fmt.Errorf("usage: harness agent create <name> --prompt \"…\" [--provider deepseek] [--model deepseek-v4-flash] [--temperature 0.2]")
	}
	a := agent.Agent{
		Name:     pos[0],
		Prompt:   flags["prompt"],
		Provider: flags["provider"],
		Model:    flags["model"],
	}
	if a.Provider == "" {
		a.Provider = "deepseek"
	}
	if a.Model == "" {
		a.Model = "deepseek-v4-flash"
	}
	if ts := flags["temperature"]; ts != "" {
		t, err := strconv.ParseFloat(ts, 64)
		if err != nil {
			return fmt.Errorf("invalid --temperature %q", ts)
		}
		a.Temperature = t
	}
	created, err := st.CreateAgent(a)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "created agent %q (%s)\n", created.Name, created.ID)
	return nil
}

func agentList(st store.Store, stdout io.Writer) error {
	agents, err := st.ListAgents()
	if err != nil {
		return err
	}
	if len(agents) == 0 {
		fmt.Fprintln(stdout, "no agents")
		return nil
	}
	for _, a := range agents {
		fmt.Fprintf(stdout, "%s\t%s/%s\t%d skill(s)\n", a.Name, a.Provider, a.Model, len(a.SkillIDs))
	}
	return nil
}

func agentGet(args []string, st store.Store, stdout io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: harness agent get <id|name>")
	}
	a, err := findAgent(st, args[0])
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "id:          %s\n", a.ID)
	fmt.Fprintf(stdout, "name:        %s\n", a.Name)
	fmt.Fprintf(stdout, "prompt:      %s\n", a.Prompt)
	fmt.Fprintf(stdout, "provider:    %s\n", a.Provider)
	fmt.Fprintf(stdout, "model:       %s\n", a.Model)
	fmt.Fprintf(stdout, "temperature: %g\n", a.Temperature)
	fmt.Fprintf(stdout, "skills:      %s\n", strings.Join(a.SkillIDs, ", "))
	return nil
}

func agentRemove(args []string, st store.Store, stdout io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: harness agent remove <id|name>")
	}
	a, err := findAgent(st, args[0])
	if err != nil {
		return err
	}
	if err := st.DeleteAgent(a.ID); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "removed agent %q\n", a.Name)
	return nil
}

func agentAttach(args []string, st store.Store, stdout io.Writer) error {
	return agentSetSkill(args, st, stdout, true)
}

func agentDetach(args []string, st store.Store, stdout io.Writer) error {
	return agentSetSkill(args, st, stdout, false)
}

func agentSetSkill(args []string, st store.Store, stdout io.Writer, attach bool) error {
	if len(args) < 2 {
		verb := "detach"
		if attach {
			verb = "attach"
		}
		return fmt.Errorf("usage: harness agent %s <agent> <skill>", verb)
	}
	a, err := findAgent(st, args[0])
	if err != nil {
		return err
	}
	sk, err := findSkill(st, args[1])
	if err != nil {
		return err
	}
	if attach {
		if containsStr(a.SkillIDs, sk.ID) {
			fmt.Fprintf(stdout, "%q already attached to %q\n", sk.Name, a.Name)
			return nil
		}
		a.SkillIDs = append(a.SkillIDs, sk.ID)
	} else {
		a.SkillIDs = removeStr(a.SkillIDs, sk.ID)
	}
	if _, err := st.UpdateAgent(a); err != nil {
		return err
	}
	verb := "detached"
	if attach {
		verb = "attached"
	}
	fmt.Fprintf(stdout, "%s skill %q to agent %q\n", verb, sk.Name, a.Name)
	return nil
}

func agentSearch(args []string, st store.Store, stdout io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: harness agent search <query>")
	}
	results, err := st.SearchAgents(strings.Join(args, " "))
	if err != nil {
		return err
	}
	if len(results) == 0 {
		fmt.Fprintln(stdout, "no matching agents")
		return nil
	}
	for _, a := range results {
		fmt.Fprintf(stdout, "%s\t%s/%s\n", a.Name, a.Provider, a.Model)
	}
	return nil
}

// findAgent resolves an id-or-name to an Agent.
func findAgent(st store.Store, idOrName string) (agent.Agent, error) {
	a, err := st.GetAgent(idOrName)
	if err == nil {
		return a, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return agent.Agent{}, err
	}
	list, err := st.ListAgents()
	if err != nil {
		return agent.Agent{}, err
	}
	for _, x := range list {
		if x.Name == idOrName {
			return x, nil
		}
	}
	return agent.Agent{}, fmt.Errorf("agent %q not found", idOrName)
}

func findSkill(st store.Store, idOrName string) (skill.Skill, error) {
	sk, err := st.GetSkill(idOrName)
	if err == nil {
		return sk, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return skill.Skill{}, err
	}
	list, err := st.ListSkills()
	if err != nil {
		return skill.Skill{}, err
	}
	for _, x := range list {
		if x.Name == idOrName {
			return x, nil
		}
	}
	return skill.Skill{}, fmt.Errorf("skill %q not found", idOrName)
}

func containsStr(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func removeStr(xs []string, s string) []string {
	out := xs[:0]
	for _, x := range xs {
		if x != s {
			out = append(out, x)
		}
	}
	return out
}
