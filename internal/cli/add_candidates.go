package cli

// Candidate collection for interactive add reads the configured task files
// without mutating them. It deliberately keeps this source independent from
// the later prompt and completion layers so those callers share one sorted,
// deduplicated view of existing contexts, projects, and metadata.

import (
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/guerrero/gtdo/internal/config"
	"github.com/guerrero/gtdo/internal/todo"
)

// metadataCandidate groups the values observed for one metadata key.
type metadataCandidate struct {
	Key    string
	Values []string
}

// addCandidates contains the sorted words available to interactive add.
type addCandidates struct {
	Contexts []string
	Projects []string
	Metadata []metadataCandidate
}

var addMetadataWordRe = regexp.MustCompile(`^[A-Za-z0-9]+:[^ \t]+$`)

// collectAddCandidates reads the task and done files configured for the
// session. The report file is intentionally excluded: it contains history,
// not current task vocabulary.
func collectAddCandidates(cfg *config.Config) addCandidates {
	if cfg == nil {
		return addCandidates{}
	}
	return collectAddCandidatesFromPaths([]string{cfg.TodoFile, cfg.DoneFile})
}

// collectAddCandidatesFromPaths returns the sorted union of candidate words
// from each distinct path. Files are read best-effort because completion must
// remain usable when one configured file is missing or unreadable.
func collectAddCandidatesFromPaths(paths []string) addCandidates {
	contexts := make(map[string]struct{})
	projects := make(map[string]struct{})
	metadata := make(map[string]map[string]struct{})
	seenPaths := make(map[string]struct{}, len(paths))

	for _, path := range paths {
		if _, seen := seenPaths[path]; seen {
			continue
		}
		seenPaths[path] = struct{}{}

		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			if words, err := todo.SigilWords([]string{line}, '@', nil); err == nil {
				for _, word := range words {
					contexts[word] = struct{}{}
				}
			}
			if words, err := todo.SigilWords([]string{line}, '+', nil); err == nil {
				for _, word := range words {
					projects[word] = struct{}{}
				}
			}

			for _, word := range strings.FieldsFunc(line, func(r rune) bool {
				return r == ' ' || r == '\t'
			}) {
				if !addMetadataWordRe.MatchString(word) {
					continue
				}
				colon := strings.IndexByte(word, ':')
				key, value := word[:colon], word[colon+1:]
				values := metadata[key]
				if values == nil {
					values = make(map[string]struct{})
					metadata[key] = values
				}
				values[value] = struct{}{}
			}
		}
	}

	return addCandidates{
		Contexts: sortedCandidateSet(contexts),
		Projects: sortedCandidateSet(projects),
		Metadata: sortedMetadataCandidates(metadata),
	}
}

func sortedCandidateSet(values map[string]struct{}) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func sortedMetadataCandidates(values map[string]map[string]struct{}) []metadataCandidate {
	if len(values) == 0 {
		return nil
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]metadataCandidate, 0, len(keys))
	for _, key := range keys {
		out = append(out, metadataCandidate{Key: key, Values: sortedCandidateSet(values[key])})
	}
	return out
}
