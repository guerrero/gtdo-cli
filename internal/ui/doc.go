// Package ui provides gtdo's terminal output helpers (plan §4, Task 5): the
// 16-name ANSI color map from todo.cfg, line and per-word coloring with
// resets, number padding, and the hide-sigil/priority helpers used by the
// listing pipeline. In plain mode the color values carry no escape sequences
// at all, so the same rendering code serves both modes.
package ui
