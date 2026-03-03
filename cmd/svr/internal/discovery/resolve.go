package discovery

import (
	"path/filepath"
	"strings"
)

// NormalizeServiceNames extracts bare service names from paths that may
// result from shell tab-completion (e.g. "app/servora/service/" → "servora").
func NormalizeServiceNames(args []string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		name := NormalizeServiceName(arg)
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

// NormalizeServiceName extracts a bare service name from a single path argument.
func NormalizeServiceName(arg string) string {
	name := filepath.Clean(arg)

	name = strings.TrimPrefix(name, "app/")
	name = strings.TrimPrefix(name, "app\\")
	name = strings.TrimSuffix(name, "/service")
	name = strings.TrimSuffix(name, "\\service")

	if i := strings.IndexAny(name, "/\\"); i >= 0 {
		name = name[:i]
	}

	return name
}
