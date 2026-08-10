package config

// defaultFileConfig returns the §5.2 schema pre-filled with todo.sh's
// defaults (todo.sh lines ~659-712). Paths stay empty on purpose: their
// defaults derive from the resolved dir, like todo.cfg's
// ${TODO_DIR}/todo.txt and friends.
func defaultFileConfig() fileConfig {
	return fileConfig{
		Behavior: behaviorTOML{
			PreserveLineNumbers: true,
			AutoArchive:         true,
			EnableUUID:          false,
			Verbose:             1,
			SentenceDelimiters:  ",.:;",
		},
		Colors: defaultColors(),
	}
}

// defaultColors returns the [colors] defaults: todo.sh's PRI_A..PRI_C, PRI_X
// and COLOR_DONE reference the color map by name; everything else is off.
// Map stays nil: the built-in names are merged in resolveColors so that
// user overrides always win regardless of map iteration order.
func defaultColors() colorsTOML {
	return colorsTOML{
		PriA:      "yellow",
		PriB:      "green",
		PriC:      "light_blue",
		PriX:      "white",
		ColorDone: "light_grey",
	}
}

// builtinColorMap is todo.sh's color map (todo.sh lines ~669-696): NONE, the
// 16 ANSI colors, and DEFAULT, keyed lower-case. "magenta" is accepted as an
// alias for todo.cfg's "purple".
func builtinColorMap() map[string]string {
	return map[string]string{
		"none":         "",
		"black":        "\x1b[0;30m",
		"red":          "\x1b[0;31m",
		"green":        "\x1b[0;32m",
		"brown":        "\x1b[0;33m",
		"blue":         "\x1b[0;34m",
		"purple":       "\x1b[0;35m",
		"cyan":         "\x1b[0;36m",
		"light_grey":   "\x1b[0;37m",
		"dark_grey":    "\x1b[1;30m",
		"light_red":    "\x1b[1;31m",
		"light_green":  "\x1b[1;32m",
		"yellow":       "\x1b[1;33m",
		"light_blue":   "\x1b[1;34m",
		"light_purple": "\x1b[1;35m",
		"light_cyan":   "\x1b[1;36m",
		"white":        "\x1b[1;37m",
		"default":      "\x1b[0m",
		"magenta":      "\x1b[0;35m", // alias for purple
	}
}
