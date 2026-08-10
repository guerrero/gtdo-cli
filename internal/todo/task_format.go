package todo

import (
	"fmt"
	"regexp"
	"strings"
)

type taskFormatField uint8

const (
	taskFormatChecked taskFormatField = iota
	taskFormatPriority
	taskFormatUUID
	taskFormatContent
	taskFormatKeywords
	taskFormatProject
	taskFormatContext
)

// TaskFormat is a validated field ordering for a todo.txt task line.
type TaskFormat struct {
	fields []taskFormatField
}

// ParseTaskFormat validates spec and returns its requested field ordering.
func ParseTaskFormat(spec string) (TaskFormat, error) {
	var fields []taskFormatField
	for i := 0; i < len(spec); {
		if isTaskFormatSpace(spec[i]) {
			i++
			continue
		}
		if spec[i] != '[' {
			return TaskFormat{}, fmt.Errorf("invalid task format: expected field at byte %d", i)
		}
		end := strings.IndexByte(spec[i:], ']')
		if end < 0 {
			return TaskFormat{}, fmt.Errorf("invalid task format: missing closing bracket")
		}
		name := spec[i+1 : i+end]
		field, ok := taskFormatFieldFor(name)
		if !ok {
			return TaskFormat{}, fmt.Errorf("invalid task format: unknown field %q", name)
		}
		fields = append(fields, field)
		i += end + 1
	}
	if len(fields) == 0 {
		return TaskFormat{}, fmt.Errorf("invalid task format: no fields")
	}
	return TaskFormat{fields: fields}, nil
}

func isTaskFormatSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n'
}

func taskFormatFieldFor(name string) (taskFormatField, bool) {
	switch name {
	case "checked":
		return taskFormatChecked, true
	case "priority":
		return taskFormatPriority, true
	case "uuid":
		return taskFormatUUID, true
	case "content":
		return taskFormatContent, true
	case "keywords":
		return taskFormatKeywords, true
	case "project":
		return taskFormatProject, true
	case "context":
		return taskFormatContext, true
	default:
		return 0, false
	}
}

// FormatLine reorders the recognized fields in line according to f.
func (f TaskFormat) FormatLine(line string) string {
	words := strings.Fields(line)
	values := make([][]string, taskFormatContext+1)
	if len(words) > 0 && words[0] == "x" {
		values[taskFormatChecked] = []string{"x"}
		words = words[1:]
		if len(words) > 0 && completionDateWordRe.MatchString(words[0]) {
			words = words[1:]
		}
	}
	if len(words) > 0 && taskFormatPriorityWordRe.MatchString(words[0]) {
		values[taskFormatPriority] = []string{words[0]}
		words = words[1:]
	}
	for _, word := range words {
		switch {
		case taskFormatUUIDWordRe.MatchString(word):
			values[taskFormatUUID] = append(values[taskFormatUUID], word)
		case metaWordRe.MatchString(word):
			values[taskFormatKeywords] = append(values[taskFormatKeywords], word)
		case projectWordRe.MatchString(word):
			values[taskFormatProject] = append(values[taskFormatProject], word)
		case contextWordRe.MatchString(word):
			values[taskFormatContext] = append(values[taskFormatContext], word)
		default:
			values[taskFormatContent] = append(values[taskFormatContent], word)
		}
	}

	parts := make([]string, 0, len(f.fields))
	for _, field := range f.fields {
		if value := strings.Join(values[field], " "); value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, " ")
}

var (
	completionDateWordRe     = regexp.MustCompile(`^(19|20)[0-9]{2}-[0-9]{2}-[0-9]{2}$`)
	taskFormatPriorityWordRe = regexp.MustCompile(`^\([A-Z]\)$`)
	taskFormatUUIDWordRe     = regexp.MustCompile(`^[0-9]{8}T[0-9]{6}\.[0-9]{2}Z$`)
)
