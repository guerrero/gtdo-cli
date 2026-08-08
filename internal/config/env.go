package config

import (
	"os"
	"strconv"
	"strings"
)

// envString returns the variable's value when it is set to something
// non-empty. Like todo.sh's ${VAR:-default}, an empty variable counts as
// unset.
func envString(name string) (string, bool) {
	if v := os.Getenv(name); v != "" {
		return v, true
	}
	return "", false
}

// envOr returns the variable's value, or fallback when unset or empty.
func envOr(name, fallback string) string {
	if v, ok := envString(name); ok {
		return v
	}
	return fallback
}

// envBool parses a boolean variable. Only "0", "false", "no", and "off"
// (any case) disable; any other non-empty value enables, mirroring todo.sh,
// where any value other than "0" means true.
func envBool(name string) (bool, bool) {
	v, ok := envString(name)
	if !ok {
		return false, false
	}
	switch strings.ToLower(v) {
	case "0", "false", "no", "off":
		return false, true
	}
	return true, true
}

// envInt parses an integer variable. An unparseable value is treated as
// unset rather than an error.
func envInt(name string) (int, bool) {
	v, ok := envString(name)
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return n, true
}

// pickBool layers a CLI flag over the environment over the fallback value.
func pickBool(set bool, flag bool, envName string, fallback bool) bool {
	if set {
		return flag
	}
	if v, ok := envBool(envName); ok {
		return v
	}
	return fallback
}

// pickString layers the environment over the fallback (no string flags).
func pickString(envName, fallback string) string {
	if v, ok := envString(envName); ok {
		return v
	}
	return fallback
}
