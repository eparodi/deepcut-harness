package cli

import (
	"fmt"
	"io"
	"strings"

	"deepcut-harness/internal/skill"
	"deepcut-harness/internal/store"
)

func skillCmd(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: harness skill create|list|get|remove|search …")
	}
	st, err := openStore()
	if err != nil {
		return err
	}
	defer st.Close()

	switch args[0] {
	case "create":
		return skillCreate(args[1:], st, stdout)
	case "list":
		return skillList(st, stdout)
	case "get":
		return skillGet(args[1:], st, stdout)
	case "remove":
		return skillRemove(args[1:], st, stdout)
	case "search":
		return skillSearch(args[1:], st, stdout)
	default:
		return fmt.Errorf("unknown skill subcommand %q", args[0])
	}
}

func skillCreate(args []string, st store.Store, stdout io.Writer) error {
	pos, flags, err := parseFlags(args)
	if err != nil {
		return err
	}
	if len(pos) < 1 {
		return fmt.Errorf("usage: harness skill create <name> --runtime bash [--description \"…\"] [--category role|stack|process] [--allowed cmd1,cmd2]")
	}
	sk := skill.Skill{
		Name:        pos[0],
		Description: flags["description"],
		Category:    flags["category"],
		Runtime:     flags["runtime"],
	}
	if allowed := flags["allowed"]; allowed != "" {
		for _, c := range strings.Split(allowed, ",") {
			if c = strings.TrimSpace(c); c != "" {
				sk.AllowedCommands = append(sk.AllowedCommands, c)
			}
		}
	}
	created, err := st.CreateSkill(sk)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "created skill %q (%s)\n", created.Name, created.ID)
	return nil
}

func skillList(st store.Store, stdout io.Writer) error {
	skills, err := st.ListSkills()
	if err != nil {
		return err
	}
	if len(skills) == 0 {
		fmt.Fprintln(stdout, "no skills")
		return nil
	}
	for _, s := range skills {
		fmt.Fprintf(stdout, "%s\t%s\t%s\n", s.Name, s.Runtime, s.Category)
	}
	return nil
}

func skillGet(args []string, st store.Store, stdout io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: harness skill get <id|name>")
	}
	s, err := findSkill(st, args[0])
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "id:          %s\n", s.ID)
	fmt.Fprintf(stdout, "name:        %s\n", s.Name)
	fmt.Fprintf(stdout, "description: %s\n", s.Description)
	fmt.Fprintf(stdout, "category:    %s\n", s.Category)
	fmt.Fprintf(stdout, "runtime:     %s\n", s.Runtime)
	fmt.Fprintf(stdout, "allowed:     %s\n", strings.Join(s.AllowedCommands, ", "))
	return nil
}

func skillRemove(args []string, st store.Store, stdout io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: harness skill remove <id|name>")
	}
	s, err := findSkill(st, args[0])
	if err != nil {
		return err
	}
	if err := st.DeleteSkill(s.ID); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "removed skill %q\n", s.Name)
	return nil
}

func skillSearch(args []string, st store.Store, stdout io.Writer) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: harness skill search <query>")
	}
	results, err := st.SearchSkills(strings.Join(args, " "))
	if err != nil {
		return err
	}
	if len(results) == 0 {
		fmt.Fprintln(stdout, "no matching skills")
		return nil
	}
	for _, s := range results {
		fmt.Fprintf(stdout, "%s\t%s\n", s.Name, s.Runtime)
	}
	return nil
}
