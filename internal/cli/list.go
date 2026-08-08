package cli

// The listing actions (plan §6.2-§6.3): list, listall, listpri, listcon,
// and listproj, wired to the _format pipeline in internal/todo with
// todo.sh-exact summaries, priority ranges, and sigil extraction. The
// Colorer is the resolved config itself: *config.Config keeps the
// per-letter pri_a..pri_z TOML colors (Task 5's wiring decision), exactly
// like todo.sh's PRI_<letter> exports with the PRI_X fallback.

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/guerrero/gtdo/internal/config"
	"github.com/guerrero/gtdo/internal/todo"
)

// registerListingActions adds the §6.3 listing actions to root. The
// description texts follow todo.sh's actionsHelp output (minus the
// shell-quoting advice, which does not apply to gtdo's argument passing).
func registerListingActions(root *cobra.Command, cfg *config.Config) {
	add := func(spec actionSpec) { root.AddCommand(newAction(spec, cfg)) }
	add(actionSpec{
		use:     "list [TERM...]",
		aliases: []string{"ls"},
		short:   "Display all tasks in todo.txt.",
		long: "Displays all tasks that contain TERM(s) sorted by priority with line\n" +
			"numbers.  Each task must match all TERM(s) (logical AND); to display\n" +
			"tasks that contain any TERM (logical OR), use 'TERM1\\|TERM2' (quoted).\n" +
			"Hides all tasks that contain TERM(s) preceded by a\n" +
			"minus sign (i.e. -TERM).\n" +
			"TERM(s) are grep-style basic regular expressions.\n" +
			"If no TERM specified, lists entire todo.txt.",
		run: actionList,
	})
	add(actionSpec{
		use:     "listall [TERM...]",
		aliases: []string{"lsa"},
		short:   "Display all the lines in todo.txt AND done.txt.",
		long: "Displays all the lines in todo.txt AND done.txt that contain TERM(s)\n" +
			"sorted by priority with line numbers.  Hides all tasks that\n" +
			"contain TERM(s) preceded by a minus sign (i.e. -TERM).  If no\n" +
			"TERM specified, lists entire todo.txt AND done.txt\n" +
			"concatenated and sorted.",
		run: actionListall,
	})
	add(actionSpec{
		use:     "listcon [TERM...]",
		aliases: []string{"lsc"},
		short:   "List all the task contexts that start with the @ sign in todo.txt.",
		long: "Lists all the task contexts that start with the @ sign in todo.txt.\n" +
			"If TERM specified, considers only tasks that contain TERM(s).",
		run: actionListcon,
	})
	add(actionSpec{
		use:     "listpri [PRIORITIES] [TERM...]",
		aliases: []string{"lsp"},
		short:   "Display all tasks prioritized PRIORITIES.",
		long: "Displays all tasks prioritized PRIORITIES.\n" +
			"PRIORITIES can be a [concatenation of] single (A) or range (A-C).\n" +
			"If no PRIORITIES specified, lists all prioritized tasks.\n" +
			"If TERM specified, lists only prioritized tasks that contain TERM(s).\n" +
			"Hides all tasks that contain TERM(s) preceded by a minus sign\n" +
			"(i.e. -TERM).",
		run: actionListpri,
	})
	add(actionSpec{
		use:     "listproj [TERM...]",
		aliases: []string{"lsprj"},
		short:   "List all the projects (terms that start with a + sign) in todo.txt.",
		long: "Lists all the projects (terms that start with a + sign) in\n" +
			"todo.txt.\n" +
			"If TERM specified, considers only tasks that contain TERM(s).",
		run: actionListproj,
	})
}

// actionList implements `list|ls`: the _format pipeline on todo.txt with
// the summary tail (todo.sh's _list).
func actionList(cmd *cobra.Command, args []string, cfg *config.Config) error {
	s, err := newSession(cmd, cfg)
	if err != nil {
		return err
	}
	return listFile(s, cfg.TodoFile, args, todo.FormatOptions{})
}

// listFile runs the _format pipeline on one file and prints the listing
// and the summary tail, todo.sh's _list: the tasks are read fresh (their
// real line numbers and blanks drive the numbering), formatted, and the
// "--" + counts line is appended when verbose.
func listFile(s *session, path string, terms []string, opts todo.FormatOptions) error {
	tasks, err := s.store.ReadTasks(path)
	if err != nil {
		return err
	}
	opts.Colors = s.cfg // *config.Config implements todo.Colorer (§5.2)
	opts.HideProjects = s.cfg.HideProjects
	opts.HideContexts = s.cfg.HideContexts
	opts.HidePriority = s.cfg.HidePriority
	return printList(s, path, tasks, terms, opts)
}

// printList formats and prints one listing: the pipeline lines followed by
// the summary tail when verbose.
func printList(s *session, path string, tasks []todo.Task, terms []string, opts todo.FormatOptions) error {
	lines, shown, total, err := todo.Format(tasks, terms, opts)
	if err != nil {
		return err
	}
	for _, line := range lines {
		fmt.Fprintln(s.out, line)
	}
	if s.verbose() {
		for _, line := range todo.Summary(todo.Prefix(path), shown, total) {
			fmt.Fprintln(s.out, line)
		}
	}
	return nil
}

// actionListall implements `listall|lsa`: todo.txt and done.txt are
// numbered continuously, done.txt's lines are renumbered 0 through the
// pipeline's post-filter, and the summary reports both files and the
// total (todo.sh:1336-1348).
func actionListall(cmd *cobra.Command, args []string, cfg *config.Config) error {
	s, err := newSession(cmd, cfg)
	if err != nil {
		return err
	}
	// TOTAL and PADDING come from todo.txt's raw line count, blank lines
	// included; done.txt's tasks are renumbered 0.
	total, err := s.store.CountLines(cfg.TodoFile)
	if err != nil {
		return err
	}
	padding := len(strconv.Itoa(total))
	tasks, err := s.store.ReadAllTasks()
	if err != nil {
		return err
	}
	opts := todo.FormatOptions{
		Width:        padding,
		Colors:       s.cfg,
		HideProjects: s.cfg.HideProjects,
		HideContexts: s.cfg.HideContexts,
		HidePriority: s.cfg.HidePriority,
		PostFilter:   todo.ListallPostFilter(total, padding),
	}
	lines, _, _, err := todo.Format(tasks, args, opts)
	if err != nil {
		return err
	}
	for _, line := range lines {
		fmt.Fprintln(s.out, line)
	}
	if s.verbose() {
		// The per-file counts re-run the pipeline on each file alone,
		// numbered with width 1 (todo.sh:1341-1342).
		taskNum, err := filteredCount(s, cfg.TodoFile, args)
		if err != nil {
			return err
		}
		doneNum, err := filteredCount(s, cfg.DoneFile, args)
		if err != nil {
			return err
		}
		doneTotal, err := s.store.CountLines(cfg.DoneFile)
		if err != nil {
			return err
		}
		fmt.Fprintln(s.out, "--")
		fmt.Fprintf(s.out, "%s: %d of %d tasks shown\n", todo.Prefix(cfg.TodoFile), taskNum, total)
		fmt.Fprintf(s.out, "%s: %d of %d tasks shown\n", todo.Prefix(cfg.DoneFile), doneNum, doneTotal)
		fmt.Fprintf(s.out, "total %d of %d tasks shown\n", taskNum+doneNum, total+doneTotal)
	}
	return nil
}

// filteredCount mirrors the TASKNUM/DONENUM computations of listall's
// summary: the term-filtered line count of one file, numbered with width 1
// (todo.sh's `_format "$FILE" 1 "$@" | sed -n '$ ='`).
func filteredCount(s *session, path string, terms []string) (int, error) {
	tasks, err := s.store.ReadTasks(path)
	if err != nil {
		return 0, err
	}
	_, shown, _, err := todo.Format(tasks, terms, todo.FormatOptions{Width: 1})
	return shown, err
}

// actionListpri implements `listpri|lsp`: the first argument is a
// PRIORITIES range when it matches todo.sh's detection grep — a single
// letter, a letter-dash-letter, or an uppercase sequence of letters and
// dashes — and becomes the character class of the priority post-filter;
// otherwise the class is A-Z and the argument stays a filter term
// (todo.sh:1381-1385). Only tasks with a priority in the class are shown.
func actionListpri(cmd *cobra.Command, args []string, cfg *config.Config) error {
	s, err := newSession(cmd, cfg)
	if err != nil {
		return err
	}
	pri, rest := "A-Z", args
	if len(args) > 0 {
		if p, ok := todo.ListpriArg(args[0]); ok {
			pri, rest = p, args[1:]
		}
	}
	class, err := todo.CompilePriorityClass(pri)
	if err != nil {
		// The post-filter grep fails to compile: the diagnostic goes to
		// stderr, no task is shown, and the summary still prints
		// (todo.sh:1384, _format's eval of the broken filter).
		fmt.Fprintln(s.errw, err)
		return listBroken(s, cfg.TodoFile)
	}
	return listFile(s, cfg.TodoFile, rest, todo.FormatOptions{
		PostFilter: priorityPostFilter(class),
	})
}

// priorityPostFilter returns the listpri post-filter: todo.sh's
// `grep '^ *[0-9]\+ ([${pri}]) '` over the numbered lines.
func priorityPostFilter(class todo.PriorityClass) func([]string) ([]string, error) {
	return func(lines []string) ([]string, error) {
		kept := lines[:0]
		for _, line := range lines {
			rest := strings.TrimLeft(line, " ")
			i := 0
			for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
				i++
			}
			if i == 0 || i >= len(rest) || rest[i] != ' ' {
				continue
			}
			rest = rest[i+1:]
			if len(rest) >= 4 && rest[0] == '(' && rest[2] == ')' && rest[3] == ' ' &&
				class(rest[1]) {
				kept = append(kept, line)
			}
		}
		return kept, nil
	}
}

// listBroken prints the summary tail for a failed priority class: 0 of
// TOTALTASKS tasks shown, where TOTALTASKS is the pipeline's count of
// listable lines. The failed grep matched nothing, so "shown" is 0
// regardless of the terms.
func listBroken(s *session, path string) error {
	tasks, err := s.store.ReadTasks(path)
	if err != nil {
		return err
	}
	if !s.verbose() {
		return nil
	}
	for _, line := range todo.Summary(todo.Prefix(path), 0, todo.ListableCount(tasks)) {
		fmt.Fprintln(s.out, line)
	}
	return nil
}

// actionListcon implements `listcon|lsc`: the unique @-words of the source
// file(s), sorted byte-wise, no numbering or summary (todo.sh:1370-1372).
func actionListcon(cmd *cobra.Command, args []string, cfg *config.Config) error {
	return listSigils(cmd, cfg, '@', args)
}

// actionListproj implements `listproj|lsprj`: like listcon for +-words.
func actionListproj(cmd *cobra.Command, args []string, cfg *config.Config) error {
	return listSigils(cmd, cfg, '+', args)
}

// listSigils runs listWordsWithSigil (todo.sh:1053): the terms filter the
// raw lines of the source file(s), then the sigil words are extracted,
// deduplicated, and sorted.
func listSigils(cmd *cobra.Command, cfg *config.Config, sigil byte, terms []string) error {
	s, err := newSession(cmd, cfg)
	if err != nil {
		return err
	}
	var lines []string
	for _, path := range sourceVarFiles(cfg) {
		text, err := os.ReadFile(path)
		if err != nil {
			// cat's diagnostic; the pipeline continues with the other files.
			fmt.Fprintf(s.errw, "cat: %s: %s\n", path, catError(err))
			continue
		}
		lines = append(lines, strings.Split(strings.TrimSuffix(string(text), "\n"), "\n")...)
	}
	words, err := todo.SigilWords(lines, sigil, terms)
	if err != nil {
		return err
	}
	for _, w := range words {
		fmt.Fprintln(s.out, w)
	}
	return nil
}

// sourceVarFiles resolves the files listcon/listproj read from (§6.5):
// todo.sh eval's TODOTXT_SOURCEVAR into FILE and cats "${FILE[@]}", so the
// value is either one path or an array literal of paths. $TODO_FILE,
// $DONE_FILE, $REPORT_FILE, and $HOME expand to the configured values.
func sourceVarFiles(cfg *config.Config) []string {
	v := cfg.SourceVar
	if v == "" {
		return []string{cfg.TodoFile}
	}
	expand := func(s string) string {
		s = strings.ReplaceAll(s, "$TODO_FILE", cfg.TodoFile)
		s = strings.ReplaceAll(s, "$DONE_FILE", cfg.DoneFile)
		s = strings.ReplaceAll(s, "$REPORT_FILE", cfg.ReportFile)
		s = strings.ReplaceAll(s, "$HOME", os.Getenv("HOME"))
		return s
	}
	v = expand(v)
	if strings.HasPrefix(v, "(") && strings.HasSuffix(v, ")") {
		var out []string
		for _, w := range strings.Fields(v[1 : len(v)-1]) {
			out = append(out, strings.Trim(w, `"`))
		}
		return out
	}
	return []string{strings.Trim(v, `"`)}
}

// catError renders a read failure the way GNU cat's diagnostic does; the
// pipeline swallows the error and continues, like `cat a b` with b missing.
func catError(err error) string {
	var pe *os.PathError
	if errors.As(err, &pe) {
		if os.IsNotExist(pe.Err) {
			return "No such file or directory"
		}
		if info, statErr := os.Stat(pe.Path); statErr == nil && info.IsDir() {
			return "Is a directory"
		}
	}
	return err.Error()
}
