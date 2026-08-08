// Package todo implements gtdo's domain core: the Task model for one line of
// a todo.txt file, parsing, filtering, sorting, the mutating actions, and the
// _format listing pipeline (plan §4, Tasks 3–4). It is pure file-format
// logic; the cli package wires it into commands and config supplies the
// knobs.
//
// Behavior mirrors todo.sh byte for byte wherever it touches task text, file
// state, and output; the txtar session tests in internal/cli pin the
// observable results.
package todo

// This file will define the Task type: the parsed form of one todo.txt line —
// raw text, real file line number (1-based), priority, date, done flag, and
// the contexts and projects found in the text (plan §6.2.4, Task 3).
