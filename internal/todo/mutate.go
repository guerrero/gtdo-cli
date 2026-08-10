package todo

// The mutating actions (plan §6.3): add/addm/addto, append, prepend, pri,
// depri, do, del, replace, move, archive, and report, with todo.sh-exact
// file-state effects. The CLI layer owns prompts, usage errors, and
// message formatting; these operations take their inputs as arguments and
// return the pieces the messages need (line numbers, old/new texts).
//
// The exact todo.sh texts are mirrored, with two deliberate simplifications
// that are invisible in the observable behavior: text inserted by
// append/prepend/replace needs no sed escaping (todo.sh escapes `\`, `|`,
// `&` only so that sed inserts them literally — Go concatenation is
// already literal), and no `.bak` files are left behind.

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// AddResult is one add/addm/addto outcome: the new line number and the
// final task text (after cleaning, priority uppercasing, timestamp-ID, and
// priority_on_add prefixes) — the pieces the CLI prints as
// "N text" and "PREFIX: N added.".
type AddResult struct {
	LineNumber int
	Text       string
}

// Add appends text as a new task to TodoFile and returns its line number
// and final text (§6.3 add).
func (s *Store) Add(text, priorityOnAdd string, now time.Time) (int, string, error) {
	allocator, err := s.idAllocatorFor(s.TodoFile, now)
	if err != nil {
		return 0, "", err
	}
	res, err := s.addTo(s.TodoFile, text, priorityOnAdd, allocator)
	if err != nil {
		return 0, "", err
	}
	return res.LineNumber, res.Text, nil
}

// Addm splits text on newlines and adds each non-empty line as its own
// task, like `IFS=$'\n'; for line in $input` (t2000 'actual multiline
// add').
func (s *Store) Addm(text, priorityOnAdd string, now time.Time) ([]AddResult, error) {
	allocator, err := s.idAllocatorFor(s.TodoFile, now)
	if err != nil {
		return nil, err
	}
	var results []AddResult
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		res, err := s.addTo(s.TodoFile, line, priorityOnAdd, allocator)
		if err != nil {
			return results, err
		}
		results = append(results, res)
	}
	return results, nil
}

// Addto appends text to a file inside Dir; the destination must already
// exist (§6.3 addto, t1020). dest is resolved relative to Dir.
func (s *Store) Addto(dest, text, priorityOnAdd string, now time.Time) (int, string, error) {
	path := filepath.Join(s.Dir, dest)
	if !isRegular(path) {
		// The exact todo.sh die text, contract (§6.3 addto).
		return 0, "", fmt.Errorf("TODO: Destination file %s does not exist.", path) //nolint:revive,staticcheck
	}
	allocator, err := s.idAllocatorFor(path, now)
	if err != nil {
		return 0, "", err
	}
	res, err := s.addTo(path, text, priorityOnAdd, allocator)
	if err != nil {
		return 0, "", err
	}
	return res.LineNumber, res.Text, nil
}

// addTo implements todo.sh's _addto: clean, uppercase the priority, prepend
// priority_on_add, optionally insert a timestamp ID, fix a missing end of
// line, append, and return the new line number.
func (s *Store) addTo(file, text, priorityOnAdd string, allocator *idAllocator) (AddResult, error) {
	input := cleanInput(text)
	input = uppercasePriority(input)
	if priorityOnAdd != "" && !priorityOnAddRe.MatchString(input) {
		input = "(" + priorityOnAdd + ") " + input
	}
	if allocator != nil {
		prefix := parseTaskPrefix(input)
		if prefix.uuid == "" {
			// Generated IDs follow the optional done marker and priority, before
			// any legacy date or ordinary task text.
			prefixLen := 0
			if prefix.done {
				prefixLen += len(donePrefix)
			}
			prefixLen += len(prefix.priority)
			input = input[:prefixLen] + allocator.nextID() + " " + input[prefixLen:]
		} else {
			// Explicit IDs are retained and participate in this batch's
			// collision set so later generated IDs cannot duplicate them.
			allocator.reserve(prefix.uuid)
		}
	}
	lines, _, err := readLines(file)
	if err != nil {
		return AddResult{}, err
	}
	// fixMissingEndOfLine is implicit: writing with a final newline gives
	// an unterminated file its terminator before the appended task.
	if err := writeLines(file, append(lines, input), true); err != nil {
		return AddResult{}, err
	}
	return AddResult{LineNumber: len(lines) + 1, Text: input}, nil
}

// idAllocatorFor snapshots the destination's existing identifiers once for
// the current add operation. Disabled stores avoid the extra read entirely so
// their legacy error and append behavior is unchanged.
func (s *Store) idAllocatorFor(file string, now time.Time) (*idAllocator, error) {
	if !s.EnableUUID {
		return nil, nil
	}
	lines, _, err := readLines(file)
	if err != nil {
		return nil, err
	}
	return newIDAllocator(lines, now), nil
}

// cleanInput maps CR and LF to spaces, todo.sh's cleaninput: tasks always
// comprise a single line. It does not trim other whitespace — "  padded  "
// stays padded (verified against the real todo.sh).
func cleanInput(text string) string {
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	return text
}

// uppercasePriority upper-cases a lowercase priority at the start of the
// text (todo.sh's sed chain s/^[(]a[)]/(A)/ ... s/^[(]z[)]/(Z)/).
func uppercasePriority(text string) string {
	return lowerPriRe.ReplaceAllStringFunc(text, func(s string) string {
		return "(" + strings.ToUpper(s[1:2]) + ")"
	})
}

// Append appends text to the item-th task of TodoFile (§6.3 append). A
// space is inserted unless the text starts with a sentence delimiter from
// SentenceDelimiters; CR/LF are squashed; `\`, `|`, `&` land literally
// (todo.sh escapes them only for its sed substitution, see the package
// comment).
func (s *Store) Append(item int, text string) (string, error) {
	task, err := s.taskAt(s.TodoFile, item)
	if err != nil {
		return "", err
	}
	appendSpace := " "
	if len(text) > 0 && strings.IndexByte(s.SentenceDelimiters, text[0]) >= 0 {
		appendSpace = ""
	}
	prefix := parseMutationPrefix(task.Text)
	newText := prefix.render(prefix.rest + appendSpace + cleanInput(text))
	if err := s.replaceTask(s.TodoFile, item, func(string) string { return newText }); err != nil {
		// The exact todo.sh die text, contract (§6.3 append).
		return "", fmt.Errorf("TODO: Error appending task %d.", item) //nolint:revive,staticcheck
	}
	return newText, nil
}

// Prepend inserts text at the start of the item-th task, preserving an
// existing priority and date prefix; no space is inserted before the text
// (§6.3 prepend, t1400).
func (s *Store) Prepend(item int, text string) (string, error) {
	_, newText, err := s.replaceOrPrepend(item, text, false)
	return newText, err
}

// Replace swaps the item-th task's text for text, keeping the existing
// priority and date unless the replacement text carries its own (§6.3
// replace, t1100).
func (s *Store) Replace(item int, text string) (string, string, error) {
	return s.replaceOrPrepend(item, text, true)
}

// replaceOrPrepend is todo.sh's replaceOrPrepend: extract the existing
// priority and date prefix, for replace let the input's own prefix win and
// strip it from the input, then rebuild the line.
func (s *Store) replaceOrPrepend(item int, text string, isReplace bool) (string, string, error) {
	task, err := s.taskAt(s.TodoFile, item)
	if err != nil {
		return "", "", err
	}
	prefix := parseMutationPrefix(task.Text)
	inputPrefix := parseMutationPrefix(text)
	// Before UUID metadata existed, replace/prepend did not recognize a
	// leading done marker. Keep that byte behavior for legacy lines while
	// retaining the canonical done+UUID prefix for migrated tasks.
	if prefix.done && prefix.uuid == "" {
		prefix = taskPrefix{rest: task.Text}
	}
	priority, prepdate := prefix.priority, prefix.date
	input := cleanInput(text)
	if isReplace {
		// A done marker at the start of replacement input was body text to
		// todo.sh, so leave it untouched. Likewise, when an old task has no
		// UUID, a caller's canonical-looking ID remains literal replacement
		// text; mutations never generate or promote metadata on that line.
		if inputPrefix.done || (prefix.uuid == "" && inputPrefix.uuid != "") {
			if inputPrefix.priority != "" && !inputPrefix.done {
				priority = inputPrefix.priority
				input = strings.TrimPrefix(input, inputPrefix.priority)
			}
		} else {
			if inputPrefix.date != "" {
				prepdate = inputPrefix.date
			}
			if inputPrefix.priority != "" {
				priority = inputPrefix.priority
			}
			// Replacement metadata is stripped from the new body. An existing
			// identifier always wins, and the stripped body is cleaned after
			// parsing so CR/LF never creates an embedded task line.
			input = cleanInput(inputPrefix.rest)
		}
	}
	prefix.priority = priority
	prefix.date = prepdate
	// Temporarily remove the existing prefix, apply the change, and
	// re-insert it (for prepend the input goes in front of the rest).
	newBody := input
	if !isReplace {
		newBody += " " + prefix.rest
	}
	newText := prefix.render(newBody)
	if err := s.replaceTask(s.TodoFile, item, func(string) string { return newText }); err != nil {
		return "", "", err
	}
	return task.Text, newText, nil
}

// PriResult is one pri outcome.
type PriResult struct {
	LineNumber int
	NewText    string
	OldPri     byte // 0 when the task had no priority
	NewPri     byte
}

// Pri sets the priority of the item-th task to newPri, replacing an
// existing one and keeping any date (§6.3 pri, t1200). When the task
// already carries newPri the file is left untouched and OldPri == NewPri,
// which the CLI reports as "already prioritized".
func (s *Store) Pri(item int, newPri byte) (PriResult, error) {
	task, err := s.taskAt(s.TodoFile, item)
	if err != nil {
		return PriResult{}, err
	}
	prefix := parseMutationPrefix(task.Text)
	var oldPri byte
	if prefix.priority != "" {
		oldPri = prefix.priority[1] // `${todo:1:1}` — any character in the parens
	}
	newText := task.Text
	if oldPri != newPri {
		prefix.priority = "(" + string(newPri) + ") "
		newText = prefix.render(prefix.rest)
		if err := s.replaceTask(s.TodoFile, item, func(string) string { return newText }); err != nil {
			return PriResult{}, err
		}
	}
	return PriResult{LineNumber: item, NewText: newText, OldPri: oldPri, NewPri: newPri}, nil
}

// DepriResult is one depri outcome.
type DepriResult struct {
	LineNumber  int
	NewText     string
	Prioritized bool // false when the task carried no priority
}

// Depri removes the priority from each item in order; items without one
// are reported with Prioritized false and left untouched (§6.3 depri,
// t1700). The results cover the items processed before any error, like
// todo.sh's sequential sed calls.
func (s *Store) Depri(items []int) ([]DepriResult, error) {
	var results []DepriResult
	for _, item := range items {
		task, err := s.taskAt(s.TodoFile, item)
		if err != nil {
			return results, err
		}
		prefix := parseMutationPrefix(task.Text)
		if prefix.priority == "" {
			results = append(results, DepriResult{LineNumber: item, NewText: task.Text})
			continue
		}
		prefix.priority = ""
		newText := prefix.render(prefix.rest)
		if err := s.replaceTask(s.TodoFile, item, func(string) string { return newText }); err != nil {
			return results, err
		}
		results = append(results, DepriResult{LineNumber: item, NewText: newText, Prioritized: true})
	}
	return results, nil
}

// DoResult is one do outcome.
type DoResult struct {
	LineNumber  int
	NewText     string
	AlreadyDone bool // the task already carried the "x " marker
}

// Do marks each item done by prepending "x YYYY-MM-DD " (§6.3 do, t1500).
// An existing priority is removed first — `sed "${item}s/^(.) //"` — the
// plan's "preserving priority" note is wrong, verified against the real
// todo.sh. Auto-archive is the caller's job (it runs Archive afterwards).
func (s *Store) Do(items []int, now time.Time) ([]DoResult, error) {
	var results []DoResult
	for _, item := range items {
		task, err := s.taskAt(s.TodoFile, item)
		if err != nil {
			return results, err
		}
		if strings.HasPrefix(task.Text, "x ") {
			results = append(results, DoResult{LineNumber: item, NewText: task.Text, AlreadyDone: true})
			continue
		}
		prefix := parseMutationPrefix(task.Text)
		oldDate := prefix.date
		prefix.done = true
		prefix.priority = ""
		prefix.date = now.Format("2006-01-02")
		body := prefix.rest
		if oldDate != "" {
			// A leading date is creation metadata in an open task. The done
			// action adds a completion date while leaving that old date in the
			// body, matching todo.sh's historical output.
			body = oldDate + " " + body
		}
		newText := prefix.render(body)
		if err := s.replaceTask(s.TodoFile, item, func(string) string { return newText }); err != nil {
			return results, err
		}
		results = append(results, DoResult{LineNumber: item, NewText: newText})
	}
	return results, nil
}

// Del removes the item-th task (§6.3 del, t1800): with PreserveLineNumbers
// the line is blanked, otherwise it is deleted together with every other
// blank line (sed -e '/./!d'). Returns the removed text.
func (s *Store) Del(item int) (string, error) {
	task, err := s.taskAt(s.TodoFile, item)
	if err != nil {
		return "", err
	}
	lines, finalNL, err := readLines(s.TodoFile)
	if err != nil {
		return "", err
	}
	lines[item-1] = ""
	if s.PreserveLineNumbers {
		if err := writeLines(s.TodoFile, lines, finalNL); err != nil {
			return "", err
		}
		return task.Text, nil
	}
	// Compact the blank lines; when the last line goes, the file ends with
	// a newline (sed emits the previous line's terminator).
	lastDeleted := item == len(lines) || lines[len(lines)-1] == ""
	kept := lines[:0]
	for _, line := range lines {
		if line != "" {
			kept = append(kept, line)
		}
	}
	if lastDeleted {
		finalNL = true
	}
	if err := writeLines(s.TodoFile, kept, finalNL); err != nil {
		return "", err
	}
	return task.Text, nil
}

// DelTerm removes term from the item-th task's text by applying todo.sh's
// five sed rules in order (§6.3 del with TERM, t1800). The term is a
// basic regex, translated like the filter terms. When the text does not
// change, the error carries todo.sh's not-found message.
func (s *Store) DelTerm(item int, term string) (string, string, error) {
	task, err := s.taskAt(s.TodoFile, item)
	if err != nil {
		return "", "", err
	}
	prefix := parseMutationPrefix(task.Text)
	newBody := delTermLine(prefix.rest, term)
	newText := prefix.render(newBody)
	if newBody == prefix.rest {
		// The exact todo.sh die text, contract (§6.3 del).
		return task.Text, task.Text, fmt.Errorf("TODO: '%s' not found; no removal done.", term) //nolint:revive,staticcheck
	}
	if err := s.replaceTask(s.TodoFile, item, func(string) string { return newText }); err != nil {
		return "", "", err
	}
	if newBody == "" {
		// The term removed the whole text: todo.sh's getNewtodo dies after
		// the seds have run, leaving the blanked line in the file (verified
		// live: del 1 foo on "foo\nbar\n" → "\nbar\n", exit 1). The error
		// carries the exact die text for the CLI to print.
		return task.Text, newText, fmt.Errorf("TODO: No updated task %d.", item) //nolint:revive,staticcheck
	}
	return task.Text, newText, nil
}

// Move moves the item-th task from src to dest, blanking (or deleting, per
// PreserveLineNumbers) the source line and appending the task to dest
// after fixing dest's end of line (§6.3 move, t1850). destNum is the
// task's new line number in dest.
func (s *Store) Move(item int, dest, src string) (string, int, error) {
	if !isRegular(src) {
		// The exact todo.sh die texts, contract (§6.3 move).
		return "", 0, fmt.Errorf("TODO: Source file %s does not exist.", src) //nolint:revive,staticcheck
	}
	if !isRegular(dest) {
		return "", 0, fmt.Errorf("TODO: Destination file %s does not exist.", dest) //nolint:revive,staticcheck
	}
	task, err := s.taskAt(src, item)
	if err != nil {
		return "", 0, err
	}
	lines, finalNL, err := readLines(src)
	if err != nil {
		return "", 0, err
	}
	lines[item-1] = ""
	if s.PreserveLineNumbers {
		if err := writeLines(src, lines, finalNL); err != nil {
			return "", 0, err
		}
	} else {
		lastDeleted := item == len(lines) || lines[len(lines)-1] == ""
		kept := lines[:0]
		for _, line := range lines {
			if line != "" {
				kept = append(kept, line)
			}
		}
		if lastDeleted {
			finalNL = true
		}
		if err := writeLines(src, kept, finalNL); err != nil {
			return "", 0, err
		}
	}
	// fixMissingEndOfLine, then `echo "$todo" >> dest`.
	destLines, _, err := readLines(dest)
	if err != nil {
		return "", 0, err
	}
	destLines = append(destLines, task.Text)
	if err := writeLines(dest, destLines, true); err != nil {
		return "", 0, err
	}
	return task.Text, len(destLines), nil
}

// Archive moves the "x " lines of TodoFile to DoneFile and removes blank
// lines (§6.3 archive, t1900). It returns the archived lines (their
// pre-move text); the caller prints them and the archived or
// not-found message.
func (s *Store) Archive() ([]string, error) {
	lines, finalNL, err := readLines(s.TodoFile)
	if err != nil {
		return nil, err
	}
	// Defragment first: sed -e '/./!d' runs even when nothing is done.
	lastBlank := len(lines) > 0 && lines[len(lines)-1] == ""
	kept := lines[:0]
	for _, line := range lines {
		if line != "" {
			kept = append(kept, line)
		}
	}
	if lastBlank {
		finalNL = true
	}
	if err := writeLines(s.TodoFile, kept, finalNL); err != nil {
		return nil, err
	}
	// grep "^x " — collect the lines, then append them to done.txt.
	var archived []string
	for _, line := range kept {
		if strings.HasPrefix(line, "x ") {
			archived = append(archived, line)
		}
	}
	if len(archived) == 0 {
		return nil, nil
	}
	if err := appendTo(s.DoneFile, archived); err != nil {
		return nil, err
	}
	// sed -i.bak '/^x /d'
	lastDone := len(kept) > 0 && strings.HasPrefix(kept[len(kept)-1], "x ")
	rest := kept[:0]
	for _, line := range kept {
		if !strings.HasPrefix(line, "x ") {
			rest = append(rest, line)
		}
	}
	if lastDone {
		finalNL = true
	}
	if err := writeLines(s.TodoFile, rest, finalNL); err != nil {
		return nil, err
	}
	return archived, nil
}

// Report reads the open/done counts and appends or reuses the report line
// (§6.3 report, t1950): "YYYY-MM-DDTHH:MM:SS OPEN DONE". updated is false
// when the last report line already carries the current counts. Archiving
// first is the caller's job, mirroring todo.sh's recursive archive
// invocation.
func (s *Store) Report(now time.Time) (line string, updated bool, err error) {
	total, err := s.CountLines(s.TodoFile)
	if err != nil {
		return "", false, err
	}
	tdone, err := s.CountLines(s.DoneFile)
	if err != nil {
		return "", false, err
	}
	newData := fmt.Sprintf("%d %d", total, tdone)
	var lastReport string
	if lines, _, err := readLines(s.ReportFile); err != nil {
		return "", false, err
	} else if len(lines) > 0 {
		lastReport = lines[len(lines)-1] // sed -ne '$p'
	}
	lastData := ""
	if i := strings.IndexByte(lastReport, ' '); i >= 0 {
		lastData = lastReport[i+1:] // ${LASTREPORT#* }
	}
	if lastData == newData {
		return lastReport, false, nil
	}
	line = now.Format("2006-01-02T15:04:05") + " " + newData
	if err := appendTo(s.ReportFile, []string{line}); err != nil {
		return "", false, err
	}
	return line, true, nil
}

// TaskAt is taskAt exported for the CLI layer: del and move need the task
// text for their confirmation prompts, and getTodo's die text is the
// parity contract for out-of-range items.
func (s *Store) TaskAt(path string, item int) (Task, error) {
	return s.taskAt(path, item)
}

// taskAt returns the item-th line of path, todo.sh's getTodo: an
// out-of-range or blank line is an error carrying the file's prefix
// (TODO/DONE), the exact text the CLI prints.
func (s *Store) taskAt(path string, item int) (Task, error) {
	tasks, err := s.ReadTasks(path)
	if err != nil {
		return Task{}, err
	}
	if item < 1 || item > len(tasks) || tasks[item-1].Text == "" {
		// The exact todo.sh die text, contract (§6.3 getTodo).
		return Task{}, fmt.Errorf("%s: No task %d.", Prefix(path), item) //nolint:revive,staticcheck
	}
	return tasks[item-1], nil
}

// replaceTask rewrites line item of path with fn(line) (sed -i semantics:
// the file's end-of-line state is preserved).
func (s *Store) replaceTask(path string, item int, fn func(string) string) error {
	lines, finalNL, err := readLines(path)
	if err != nil {
		return err
	}
	lines[item-1] = fn(lines[item-1])
	return writeLines(path, lines, finalNL)
}

// parseMutationPrefix is the mutation-facing wrapper around the shared
// prefix parser. Legacy mutation expressions accepted any one-character
// priority (including lowercase) and two-to-four-digit years; retain those
// cases while still keeping canonical IDs isolated from body edits.
func parseMutationPrefix(text string) taskPrefix {
	p := parseTaskPrefix(text)
	offset := 0
	if p.done {
		offset = len(donePrefix)
	}
	if p.priority == "" {
		if m := priAnyRe.FindString(text[offset:]); m != "" {
			p.priority = m
			parsed := parseTaskPrefix(text[offset+len(m):])
			p.uuid, p.date, p.rest = parsed.uuid, parsed.date, parsed.rest
		}
	}
	if p.date == "" {
		if m := legacyMutationDateRe.FindStringSubmatch(p.rest); m != nil {
			p.date = m[1]
			p.rest = p.rest[len(m[0]):]
		}
	}
	// A completed line without a canonical UUID is legacy todo.sh text:
	// priority/date expressions apply only at byte zero, so the `x ` marker
	// and everything after it must remain body text for mutation parity.
	if p.done && p.uuid == "" {
		return taskPrefix{rest: text}
	}
	return p
}

// delTermLine applies the five sed expressions of `del NR TERM` in order.
// Each is built as a sed basic regex with the term embedded and translated
// like the filter terms. Note the spacing idioms: `  *` is one or more
// spaces (space + space*), so rules 3 and 4 collapse a term with
// surrounding spaces into a single space.
func delTermLine(line, term string) string {
	t := translateBRE(term)
	line = regexp.MustCompile(`^(\(.\) )? *`+t+` *`).ReplaceAllString(line, "${1}")
	line = regexp.MustCompile(` *`+t+` *$`).ReplaceAllString(line, "")
	line = regexp.MustCompile(` +`+t+` *`).ReplaceAllString(line, " ")
	line = regexp.MustCompile(` *`+t+` +`).ReplaceAllString(line, " ")
	line = regexp.MustCompile(t).ReplaceAllString(line, "")
	return line
}

var (
	// lowerPriRe matches a lowercase priority at the start of the text.
	lowerPriRe = regexp.MustCompile(`^\([a-z]\)`)

	// priorityOnAddRe is grep '^([A-Z])': an existing priority suppresses
	// the priority_on_add prefix (no trailing space required).
	priorityOnAddRe = regexp.MustCompile(`^\([A-Z]\)`)

	// priAnyRe matches any "(X) " at the start of the text — sed's
	// `^(.) `, where the parens are literals and X any character. It is
	// what pri/depri/do strip.
	priAnyRe = regexp.MustCompile(`^\(.\) `)

	// legacyMutationDateRe preserves replace/prepend's historical acceptance
	// of two-to-four-digit years while the shared parser remains strict for
	// Task.Date and canonical rendering.
	legacyMutationDateRe = regexp.MustCompile(`^([0-9]{2,4}-[0-9]{2}-[0-9]{2})(?: |$)`)
)
