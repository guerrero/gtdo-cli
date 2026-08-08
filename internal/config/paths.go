package config

import (
	"os"
	"path/filepath"
	"strings"
)

// expandHome expands a leading "~" or "$HOME" in path. "~user", mid-string
// "~", and other variables are left untouched: only the documented forms
// expand.
func expandHome(path, home string) string {
	switch {
	case path == "~":
		return home
	case strings.HasPrefix(path, "~/"):
		return filepath.Join(home, path[2:])
	case strings.HasPrefix(path, "$HOME"):
		return filepath.Join(home, strings.TrimPrefix(path, "$HOME"))
	}
	return path
}

// exists reports whether path names a regular file; directories do not count
// as config files.
func exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// firstNonEmpty returns the first non-empty string in vs.
func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}
