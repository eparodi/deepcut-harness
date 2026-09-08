// Command harness is the Harness AI engineering orchestrator: a single
// binary that runs agents (the brain) equipped with skills (the hands)
// against a local workspace. The subcommand dispatch and the default
// dashboard loop live in internal/cli — this file is the thin
// entrypoint that bridges os.Args to it.
package main

import (
	"os"

	"deepcut-harness/internal/cli"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr))
}
