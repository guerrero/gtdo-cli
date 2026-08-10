package todo

import (
	"strings"
	"testing"
)

const defaultTaskFormat = "[checked][priority][uuid][content][keywords][project][context]"

func mustTaskFormat(t *testing.T, spec string) TaskFormat {
	t.Helper()
	f, err := ParseTaskFormat(spec)
	if err != nil {
		t.Fatalf("ParseTaskFormat(%q): %v", spec, err)
	}
	return f
}

func TestTaskFormatDefaultOrder(t *testing.T) {
	f := mustTaskFormat(t, defaultTaskFormat)
	got := f.FormatLine("(B) write report key:one +work @desk 20260808T143045.12Z")
	want := "(B) 20260808T143045.12Z write report key:one +work @desk"
	if got != want {
		t.Errorf("FormatLine = %q, want %q", got, want)
	}
}

func TestTaskFormatCustomOrderAndChecked(t *testing.T) {
	f := mustTaskFormat(t, "[project][content][keywords][context][uuid][priority][checked]")
	got := f.FormatLine("x 2026-08-08 finish key:done +archive @desk 20260808T143045.12Z")
	want := "+archive finish key:done @desk 20260808T143045.12Z x"
	if got != want {
		t.Errorf("FormatLine = %q, want %q", got, want)
	}
}

func TestTaskFormatOmitsMissingFieldsWithoutExtraSpaces(t *testing.T) {
	f := mustTaskFormat(t, defaultTaskFormat)
	if got := f.FormatLine("plain task"); got != "plain task" {
		t.Errorf("FormatLine = %q, want %q", got, "plain task")
	}
	if got := f.FormatLine("x 2026-08-08"); got != "x" {
		t.Errorf("checked-only FormatLine = %q, want %q", got, "x")
	}
}

func TestTaskFormatKeepsRepeatedValuesInSourceOrder(t *testing.T) {
	f := mustTaskFormat(t, defaultTaskFormat)
	line := "task key:first +one @home key:second +two @work 20260808T143045.12Z"
	want := "20260808T143045.12Z task key:first key:second +one +two @home @work"
	if got := f.FormatLine(line); got != want {
		t.Errorf("FormatLine = %q, want %q", got, want)
	}
}

func TestTaskFormatLeavesInvalidUUIDAsContent(t *testing.T) {
	f := mustTaskFormat(t, defaultTaskFormat)
	line := "task 20260808T143045.1Z 20260808T143045.12Z"
	want := "20260808T143045.12Z task 20260808T143045.1Z"
	if got := f.FormatLine(line); got != want {
		t.Errorf("FormatLine = %q, want %q", got, want)
	}
}

func TestParseTaskFormatRejectsMalformedOrUnknownFields(t *testing.T) {
	for _, spec := range []string{"", "[unknown]", "[content", "content", "[content]literal"} {
		t.Run(spec, func(t *testing.T) {
			if _, err := ParseTaskFormat(spec); err == nil || !strings.Contains(err.Error(), "invalid task format") {
				t.Fatalf("ParseTaskFormat(%q) error = %v, want invalid task format", spec, err)
			}
		})
	}
}
