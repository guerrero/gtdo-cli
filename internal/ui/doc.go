// Package ui provides gtdo's terminal color palette (plan §5.2, Task 5): a
// snapshot of todo.cfg's color assignments — the priority colors pri_a..pri_c
// with the pri_x fallback, color_done, the project/context/date/number/meta
// word colors, and default (the reset after each colored word) — with the
// ANSI codes resolved by internal/config, the single source of truth for the
// 16-name color map. The Color palette implements the listing pipeline's
// Colorer hook (internal/todo, §6.2.4): it supplies color strings only; the
// pipeline applies them and renders uncolored when the palette is plain (all
// fields empty, as in plain mode). The pipeline also owns number padding and
// the -@/-+/-P hide toggles, so ui has no duplicate of those.
package ui
