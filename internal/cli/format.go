package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/guerrero/gtdo/internal/config"
	"github.com/guerrero/gtdo/internal/exitcode"
	"github.com/guerrero/gtdo/internal/todo"
)

func registerFormatAction(root *cobra.Command, cfg *config.Config) {
	root.AddCommand(newAction(actionSpec{
		use:   "format [FILE]",
		short: "Rewrite task files using the configured format.",
		long:  "Rewrites todo.txt and done.txt using the task_format configuration.\nIf FILE is specified, rewrites only that file.",
		run:   actionFormat,
	}, cfg))
}

func resolveFormatTargets(args []string, cfg *config.Config) ([]string, error) {
	if len(args) > 1 {
		return nil, fmt.Errorf("usage: gtdo format [FILE]")
	}
	var targets []string
	if len(args) == 0 {
		targets = []string{cfg.TodoFile, cfg.DoneFile}
	} else if filepath.IsAbs(args[0]) {
		targets = []string{args[0]}
	} else {
		path, err := filepath.Abs(filepath.Join(cfg.Dir, args[0]))
		if err != nil {
			return nil, err
		}
		targets = []string{path}
	}

	seen := make(map[string]struct{}, len(targets))
	unique := make([]string, 0, len(targets))
	for _, path := range targets {
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		unique = append(unique, path)
	}
	return unique, nil
}

func actionFormat(cmd *cobra.Command, args []string, cfg *config.Config) error {
	targets, err := resolveFormatTargets(args, cfg)
	if err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		return exitcode.Wrap(exitcode.Generic, exitcode.ErrFailure)
	}
	format, err := todo.ParseTaskFormat(cfg.TaskFormat)
	if err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		return exitcode.Wrap(exitcode.Generic, exitcode.ErrFailure)
	}
	if len(args) == 1 {
		if msg := formatTargetError(targets[0]); msg != "" {
			fmt.Fprintln(cmd.ErrOrStderr(), msg)
			return exitcode.Wrap(exitcode.Generic, exitcode.ErrFailure)
		}
	}

	s, err := newSession(cmd, cfg)
	if err != nil {
		return err
	}
	for _, path := range targets {
		if msg := formatTargetError(path); msg != "" {
			return s.die(msg)
		}
	}

	type formattedFile struct {
		path string
		data []byte
	}
	files := make([]formattedFile, 0, len(targets))
	for _, path := range targets {
		data, err := todo.ReformatFile(path, format)
		if err != nil {
			return s.die(err.Error())
		}
		files = append(files, formattedFile{path: path, data: data})
	}
	for _, file := range files {
		if err := os.WriteFile(file.path, file.data, 0o644); err != nil {
			return s.die(err.Error())
		}
	}
	if s.verbose() {
		for _, file := range files {
			fmt.Fprintf(s.out, "TODO: %s formatted.\n", file.path)
		}
	}
	return nil
}

func formatTargetError(path string) string {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Sprintf("TODO: File %s does not exist.", path)
	}
	if err != nil {
		return err.Error()
	}
	if !info.Mode().IsRegular() {
		return fmt.Sprintf("TODO: File %s is not a regular file.", path)
	}
	return ""
}
