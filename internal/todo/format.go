package todo

// The _format listing pipeline (plan §6.2): numbering with padding to the
// width of the total line count, term filters, case-insensitive sort by
// task text, per-word coloring, and the hide toggles (-@/-+/-P). Plain
// mode (no Colorer) disables colors. The summary tail (`--` + the counts
// line) is built by Summary; the CLI appends it, like todo.sh's _list.

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Colorer supplies the ANSI codes the colorizer applies: the config roles
// color_done, color_project, color_context, color_date, color_number,
// color_meta, and pri_<letter> with the pri_x fallback. A nil Colorer
// disables colors (plain mode).
type Colorer interface {
	Color(role string) string
	PriorityColor(letter byte) string
}

// FormatOptions parameterizes the pipeline.
type FormatOptions struct {
	// Width is the numbering width; 0 derives it from the highest line
	// number, like getPadding.
	Width int

	// HideProjects, HideContexts, HidePriority implement -+ -@ -P.
	HideProjects bool
	HideContexts bool
	HidePriority bool

	// Colors supplies the ANSI codes; nil means plain mode.
	Colors Colorer

	// PostFilter runs after the term filters and before the sort —
	// todo.sh's post_filter_command, which listall uses to renumber
	// done.txt tasks to 0.
	PostFilter func([]string) ([]string, error)
}

// Format runs the _format pipeline on the tasks: number, filter, sort,
// colorize (§6.2.1-§6.2.4). It returns the output lines (without the
// summary) and the shown/total counts the summary prints.
func Format(tasks []Task, terms []string, opts FormatOptions) (lines []string, shown, total int, err error) {
	width := opts.Width
	if width == 0 {
		width = NumberWidth(autoWidth(tasks))
	}
	// Number the tasks, dropping the lines the numbering sed cannot tell
	// from blanks: blank lines and lines of only digits and spaces
	// (sed -e '/^[ 0-9]\{1,\} *$/d').
	var numbered []string
	for _, task := range tasks {
		if allDigitSpaceRe.MatchString(task.Text) {
			continue
		}
		numbered = append(numbered, task.NumberedLine(width))
	}
	filtered, err := FilterLines(numbered, terms)
	if err != nil {
		return nil, 0, 0, err
	}
	if opts.PostFilter != nil {
		filtered, err = opts.PostFilter(filtered)
		if err != nil {
			return nil, 0, 0, err
		}
	}
	ZeroPadNumbers(filtered)
	SortLines(filtered)
	lines = make([]string, len(filtered))
	for i, line := range filtered {
		lines[i] = hideSigils(awkStep(line, opts), opts)
	}
	return lines, len(filtered), len(numbered), nil
}

// autoWidth returns the highest line number among the tasks (the file's
// line count, blank lines included).
func autoWidth(tasks []Task) int {
	maxLine := 0
	for _, task := range tasks {
		if task.LineNumber > maxLine {
			maxLine = task.LineNumber
		}
	}
	return maxLine
}

// reset is DEFAULT, todo.sh's \033[0m: emitted after every colored word
// and at the end of a colored line.
const reset = "\x1b[0m"

// awkStep mirrors todo.sh's _format awk (§6.2.4): pick the line color from
// the number prefix ("NN x " → color_done; "NN (X) " → pri_<X> with the
// pri_x fallback), drop the "(X) " label under -P, then color the words —
// the number, +projects, @contexts, valid dates, key:value metadata — with
// a reset to DEFAULT plus the line color after each colored word, and a
// final DEFAULT at the end of a colored line.
func awkStep(line string, opts FormatOptions) string {
	c := opts.Colors
	clr := ""
	if doneLineRe.MatchString(line) {
		if c != nil {
			clr = c.Color("color_done")
		}
	} else if m := priLineRe.FindStringSubmatch(line); m != nil {
		if c != nil {
			clr = c.PriorityColor(m[1][0])
		}
		if opts.HidePriority {
			// substr($0, 1, RLENGTH-4) substr($0, RSTART+RLENGTH): drop
			// the "(X) " label, keep the number and its space.
			idx := priLineRe.FindStringIndex(line)
			line = line[:idx[1]-4] + line[idx[1]:]
		}
	}
	if c == nil {
		return line
	}
	var b strings.Builder
	b.WriteString(clr)
	first := true
	for _, word := range splitWords(line) {
		beg := ""
		switch {
		case first && numberWordRe.MatchString(word):
			beg = c.Color("color_number")
		case projectWordRe.MatchString(word) && !opts.HideProjects:
			beg = c.Color("color_project")
		case contextWordRe.MatchString(word) && !opts.HideContexts:
			beg = c.Color("color_context")
		case dateWordRe.MatchString(word):
			beg = c.Color("color_date")
		case metaWordRe.MatchString(word):
			beg = c.Color("color_meta")
		}
		if beg != "" {
			b.WriteString(beg)
			b.WriteString(word)
			b.WriteString(reset)
			b.WriteString(clr)
		} else {
			b.WriteString(word)
		}
		first = false
	}
	if clr != "" {
		b.WriteString(reset)
	}
	return b.String()
}

// splitWords splits on runs of spaces and tabs, keeping the runs as their
// own words — the awk's gsub(/[ \t][ \t]*/, "\n&\n") plus split.
func splitWords(line string) []string {
	var out []string
	start := 0
	for _, m := range wsRunRe.FindAllStringIndex(line, -1) {
		out = append(out, line[start:m[0]], line[m[0]:m[1]])
		start = m[1]
	}
	return append(out, line[start:])
}

// hideSigils applies the -@/-+ substitutions todo.sh runs on the
// colorized lines: a sigil word and its preceding space is removed. -P is
// not here: awkStep's splice is the complete todo.sh behavior, which keeps
// mid-text and done-line "(X) " labels.
func hideSigils(line string, opts FormatOptions) string {
	if opts.HideProjects {
		line = hideProjectsRe.ReplaceAllString(line, "")
	}
	if opts.HideContexts {
		line = hideContextsRe.ReplaceAllString(line, "")
	}
	return line
}

// Summary builds the tail of a listing (todo.sh's _list): the "--"
// separator and the "PREFIX: N of M tasks shown" line (t0001: 0 of 0 is
// still shown).
func Summary(prefix string, shown, total int) []string {
	return []string{"--", fmt.Sprintf("%s: %d of %d tasks shown", prefix, shown, total)}
}

// Prefix is todo.sh's getPrefix: the basename of path without its
// extension, uppercased — TODO, DONE, GARDEN.
func Prefix(path string) string {
	base := filepath.Base(path)
	if i := strings.LastIndexByte(base, '.'); i >= 0 {
		base = base[:i]
	}
	return strings.ToUpper(base)
}

var (
	// allDigitSpaceRe matches task texts the numbering sed drops as
	// blank: nothing but digits and spaces.
	allDigitSpaceRe = regexp.MustCompile(`^[ 0-9]*$`)

	// doneLineRe and priLineRe classify a numbered line for the line
	// color: awk match(/^[0-9]+ x /) and match(/^[0-9]+ \([A-Z]\) /).
	doneLineRe = regexp.MustCompile(`^[0-9]+ x `)
	priLineRe  = regexp.MustCompile(`^[0-9]+ \(([A-Z])\) `)

	// The word classifiers (§6.2.4); projectWordRe/contextWordRe come
	// from parse.go. dateWordRe validates the month and day, unlike the
	// loose date regex of parse.go.
	numberWordRe = regexp.MustCompile(`^[0-9]+$`)
	dateWordRe   = regexp.MustCompile(`^(19|20)[0-9][0-9]-(0[1-9]|1[012])-(0[1-9]|[12][0-9]|3[01])$`)
	metaWordRe   = regexp.MustCompile(`^[A-Za-z0-9]+:[^ ]+$`)

	// The -@/-+ substitutions: [[:space:]][+][[:graph:]]\{1,\} and
	// [[:space:]]@[[:graph:]]\{1,\} (the parens are literal in sed BRE).
	hideProjectsRe = regexp.MustCompile(`[[:space:]][+][[:graph:]]+`)
	hideContextsRe = regexp.MustCompile(`[[:space:]]@[[:graph:]]+`)

	wsRunRe = regexp.MustCompile(`[ \t]+`)
)
