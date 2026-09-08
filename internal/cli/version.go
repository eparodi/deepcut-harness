package cli

import (
	"fmt"
	"io"
)

// version is stamped at build time via -ldflags "-X deepcut-harness/internal/cli.version=<v>";
// the default is the development marker.
var version = "dev"

func versionCmd(args []string, stdout io.Writer) error {
	fmt.Fprintf(stdout, "harness %s\n", version)
	return nil
}
