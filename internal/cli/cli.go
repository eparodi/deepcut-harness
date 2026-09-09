// Package cli implements the Harness subcommands and the default
// dashboard loop. The root main.go is a thin entrypoint: it delegates
// os.Args[1:] here and exits with the returned code.
package cli

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/joho/godotenv"
)

// command is one Harness subcommand: its name (the first CLI argument),
// a one-line usage description, and the handler. Handlers share the
// signature func(args, stdout) error; commands that need extra wiring
// adapt inside their entry so the table stays uniform. errMsg is the
// exact slog message logged when the command fails.
type command struct {
	name   string
	usage  string
	errMsg string
	run    func(args []string, stdout io.Writer) error
}

// commands is the dispatch table. Order renders the usage text, so it
// is the usage order (new commands append at the end). A new subcommand
// registers here and gets its usage line for free.
var commands = []command{
	{
		name:   "version",
		usage:  "print the build version",
		errMsg: "version failed",
		run:    versionCmd,
	},
	{
		name:   "agent",
		usage:  "manage agents (agent create|list|get|remove|attach|detach|search …)",
		errMsg: "agent command failed",
		run:    agentCmd,
	},
	{
		name:   "skill",
		usage:  "manage skills (skill create|list|get|remove|search …)",
		errMsg: "skill command failed",
		run:    skillCmd,
	},
}

// Main dispatches the Harness subcommands (the os.Args[1:] slice) and
// returns the process exit code: 0 on success, 1 on command or loop
// failure, 2 on an unknown subcommand. The no-args default runs the
// dashboard.
func Main(args []string, stdout, stderr io.Writer) int {
	// Load a gitignored .env (API keys) if present — best-effort; a
	// missing .env is not an error, the providers just have empty keys.
	_ = godotenv.Load()

	if len(args) > 0 {
		for _, c := range commands {
			if c.name == args[0] {
				if err := c.run(args[1:], stdout); err != nil {
					slog.Error(c.errMsg, "err", err)
					return 1
				}
				return 0
			}
		}
		// Unknown subcommand: never fall through into the dashboard loop
		// on a typo — print usage and exit non-zero.
		printUsage(stderr)
		return 2
	}
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		return 1
	}
	return 0
}

// printUsage renders the top-level command usage (printed on exit code
// 2 for unknown first arguments), generated from the dispatch table so
// it can never drift from the registered commands.
func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: harness [subcommand]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  (no subcommand)    run the Harness dashboard")
	for _, c := range commands {
		fmt.Fprintf(w, "  %-10s  %s\n", c.name, c.usage)
	}
}
