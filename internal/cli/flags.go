package cli

import (
	"fmt"
	"strings"
)

// parseFlags splits args into positional args and --key value flags.
// Flags may appear anywhere and always consume the following value.
func parseFlags(args []string) (positional []string, flags map[string]string, err error) {
	flags = map[string]string{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if strings.HasPrefix(a, "--") {
			key := strings.TrimPrefix(a, "--")
			if key == "" {
				return nil, nil, fmt.Errorf("invalid flag %q", a)
			}
			if i+1 >= len(args) {
				return nil, nil, fmt.Errorf("flag --%s requires a value", key)
			}
			flags[key] = args[i+1]
			i++
			continue
		}
		positional = append(positional, a)
	}
	return positional, flags, nil
}
