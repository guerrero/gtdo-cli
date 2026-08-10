package todo

import (
	"errors"
	"testing"
)

func TestSigilPolicyZeroValueAllowsAll(t *testing.T) {
	if err := (SigilPolicy{}).Validate("/tmp/todo.txt", 1, "ship +other @bad"); err != nil {
		t.Fatalf("Validate error = %v, want nil", err)
	}
}

func TestSigilPolicyAllowsListedTokens(t *testing.T) {
	policy := SigilPolicy{AllowedContexts: []string{"@work"}, AllowedProjects: []string{"+gtdo"}}
	if err := policy.Validate("/tmp/todo.txt", 2, "ship +gtdo @work"); err != nil {
		t.Fatalf("Validate error = %v, want nil", err)
	}
}

func TestSigilPolicyRejectsExplicitEmptyCategories(t *testing.T) {
	if err := (SigilPolicy{AllowedContexts: []string{}}).Validate("/tmp/todo.txt", 3, "ship @work"); err == nil {
		t.Fatal("context Validate error = nil, want rejection")
	}
	if err := (SigilPolicy{AllowedProjects: []string{}}).Validate("/tmp/todo.txt", 3, "ship +gtdo"); err == nil {
		t.Fatal("project Validate error = nil, want rejection")
	}
}

func TestSigilPolicyIsCaseSensitive(t *testing.T) {
	policy := SigilPolicy{AllowedContexts: []string{"@work"}}
	if err := policy.Validate("/tmp/todo.txt", 4, "ship @Work"); err == nil {
		t.Fatal("Validate error = nil, want rejection")
	}
}

func TestSigilPolicyRejectsFirstDisallowedToken(t *testing.T) {
	policy := SigilPolicy{
		AllowedContexts: []string{"@work"},
		AllowedProjects: []string{"+gtdo"},
	}
	err := policy.Validate("/tmp/todo.txt", 7, "ship +other @bad +gtdo")
	var violation *SigilValidationError
	if !errors.As(err, &violation) {
		t.Fatalf("Validate error = %v, want *SigilValidationError", err)
	}
	if violation.Kind != "Project" || violation.Token != "+other" || violation.Path != "/tmp/todo.txt" || violation.LineNumber != 7 {
		t.Fatalf("violation = %+v", violation)
	}
	if got := err.Error(); got != `TODO: Project "+other" is not allowed in /tmp/todo.txt at line 7.` {
		t.Fatalf("error = %q", got)
	}
}
