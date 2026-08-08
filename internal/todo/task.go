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

import (
	"fmt"
	"strconv"
)

// Task is the parsed form of one line of a todo.txt file: the raw line text
// and its real 1-based file line number, plus the parsed views provided by
// the methods in parse.go (priority, date, done flag, contexts, projects;
// plan §6.2.4).
type Task struct {
	// LineNumber is the 1-based position of the line in its source file.
	LineNumber int

	// Text is the raw line text, exactly as it appears in the file.
	Text string
}

// NumberWidth returns the width needed to right-align line numbers for a
// file with total lines, mirroring todo.sh's getPadding: the digit count of
// the total (§6.2.1). An empty file still pads to one digit, like
// `printf %s ${#LINES}` with LINES=0.
func NumberWidth(total int) int {
	return len(strconv.Itoa(total))
}

// NumberedLine formats the task as "NUM TEXT" with NUM right-aligned in a
// width-wide field (§6.2.1) — the shape the _format pipeline feeds the
// filters, the sort, and the colorizer (Task 4).
func (t Task) NumberedLine(width int) string {
	return fmt.Sprintf("%*d %s", width, t.LineNumber, t.Text)
}
