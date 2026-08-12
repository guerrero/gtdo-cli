package cli

// Guided add composition and the deterministic line-oriented input adapter.
// The line adapter deliberately has no prompt or terminal dependencies: when
// stdin is piped, each phase consumes exactly the lines assigned to it.

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/guerrero/gtdo/internal/todo"
)

// addInput is the common boundary between guided flow composition and its
// terminal or scripted input implementations.
type addInput interface {
	PromptTask(addCandidates) (string, error)
	PromptPriority(addCandidates) (string, error)
	PromptMetadata(addCandidates) ([]string, error)
	Select(guidedPhase, []string) ([]string, error)
}

// runGuided gathers the base task and each enabled optional phase in the
// stable metadata, project, context order before composing one final line.
func runGuided(input addInput, candidates addCandidates, opts addOptions) (string, error) {
	base, err := input.PromptTask(candidates)
	if err != nil {
		return "", err
	}

	var metadata, projects, contexts []string
	if opts.phaseEnabled(phaseMetadata) {
		metadata, err = input.PromptMetadata(candidates)
		if err != nil {
			return "", err
		}
	}
	if opts.phaseEnabled(phaseProject) {
		projects, err = input.Select(phaseProject, candidates.Projects)
		if err != nil {
			return "", err
		}
	}
	if opts.phaseEnabled(phaseContext) {
		contexts, err = input.Select(phaseContext, candidates.Contexts)
		if err != nil {
			return "", err
		}
	}

	return composeGuidedTask(base, metadata, projects, contexts), nil
}

// composeGuidedTask appends non-empty metadata, project, and context groups
// with one separator at each boundary. Existing exact sigil words in base are
// retained and are not appended a second time.
func composeGuidedTask(base string, metadata, projects, contexts []string) string {
	text := base
	for _, pair := range metadata {
		text = appendGuidedToken(text, pair)
	}

	projectSet := make(map[string]struct{})
	for _, project := range (todo.Task{Text: base}).Projects() {
		projectSet[project] = struct{}{}
	}
	for _, project := range sortedGuidedTokens(projects) {
		if project == "" {
			continue
		}
		if _, exists := projectSet[project]; exists {
			continue
		}
		projectSet[project] = struct{}{}
		text = appendGuidedToken(text, project)
	}

	contextSet := make(map[string]struct{})
	for _, context := range (todo.Task{Text: base}).Contexts() {
		contextSet[context] = struct{}{}
	}
	for _, context := range sortedGuidedTokens(contexts) {
		if context == "" {
			continue
		}
		if _, exists := contextSet[context]; exists {
			continue
		}
		contextSet[context] = struct{}{}
		text = appendGuidedToken(text, context)
	}

	return text
}

func appendGuidedToken(text, token string) string {
	if token == "" {
		return text
	}
	if text == "" {
		return token
	}
	return text + " " + token
}

// lineAddInput implements guided's non-TTY protocol:
//
//   - one task line;
//   - metadata key:value lines terminated by an empty line;
//   - one project line and one context line of space-separated selections.
//
// It receives a buffered reader from the session so successive phases never
// discard bytes prefetched by an earlier read.
type lineAddInput struct {
	reader *bufio.Reader
}

var _ addInput = lineAddInput{}

func (l lineAddInput) PromptTask(addCandidates) (string, error) {
	line, err := l.readLineErr()
	if err != nil {
		return "", err
	}
	return line, nil
}

// PromptPriority consumes one priority line. The shared validator accepts
// an empty line (skip) or a single ASCII letter.
func (l lineAddInput) PromptPriority(addCandidates) (string, error) {
	line, err := l.readLineErr()
	if err != nil {
		return "", err
	}
	return parseGuidedPriority(line)
}

func (l lineAddInput) PromptMetadata(addCandidates) ([]string, error) {
	var metadata []string
	for {
		line, err := l.readLineErr()
		if err != nil {
			if line == "" {
				return nil, err
			}
			return nil, err
		}
		if line == "" {
			return metadata, nil
		}
		if err := validateMetadataLine(line); err != nil {
			return nil, err
		}
		metadata = append(metadata, line)
	}
}

func (l lineAddInput) Select(_ guidedPhase, options []string) ([]string, error) {
	line, err := l.readLineErr()
	if err != nil {
		return nil, err
	}
	if line == "" {
		return nil, nil
	}

	allowed := make(map[string]struct{}, len(options))
	for _, option := range options {
		allowed[option] = struct{}{}
	}
	selected := make([]string, 0)
	seen := make(map[string]struct{})
	for _, token := range strings.Fields(line) {
		if _, ok := allowed[token]; !ok {
			continue
		}
		if _, duplicate := seen[token]; duplicate {
			continue
		}
		seen[token] = struct{}{}
		selected = append(selected, token)
	}
	sort.Strings(selected)
	return selected, nil
}

func sortedGuidedTokens(tokens []string) []string {
	if len(tokens) == 0 {
		return nil
	}
	sorted := append([]string(nil), tokens...)
	sort.Strings(sorted)
	return sorted
}

func (l lineAddInput) readLine() (string, error) {
	if l.reader == nil {
		return "", io.EOF
	}
	return readBufferedLine(l.reader)
}

func (l lineAddInput) readLineErr() (string, error) {
	return l.readLine()
}

func readBufferedLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	return strings.TrimSuffix(line, "\n"), err
}

func validateMetadataLine(line string) error {
	colon := strings.IndexByte(line, ':')
	if colon <= 0 || colon == len(line)-1 {
		return fmt.Errorf("invalid metadata %q: expected key:value", line)
	}
	key, value := line[:colon], line[colon+1:]
	for _, char := range key {
		if !isASCIIAlphaNumeric(char) {
			return fmt.Errorf("invalid metadata %q: expected key:value", line)
		}
	}
	if strings.ContainsAny(value, " \t") {
		return fmt.Errorf("invalid metadata %q: expected key:value", line)
	}
	return nil
}

// parseGuidedPriority validates a guided priority line: empty, or a single
// ASCII letter. The letter is returned as typed; composition uppercases it.
func parseGuidedPriority(line string) (string, error) {
	if line == "" {
		return "", nil
	}
	if len(line) != 1 || !isASCIILetter(line[0]) {
		return "", fmt.Errorf("invalid priority %q: expected a single letter A-Z", line)
	}
	return line, nil
}

func isASCIILetter(char byte) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')
}

func isASCIIAlphaNumeric(char rune) bool {
	return (char >= 'a' && char <= 'z') ||
		(char >= 'A' && char <= 'Z') ||
		(char >= '0' && char <= '9')
}
