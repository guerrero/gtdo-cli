package todo

import (
	"reflect"
	"testing"
)

// Task cases pin the parsed views of one todo.txt line. Expected behaviors
// were cross-checked against todo.sh and the todo.txt test suite (t1250
// listpri, t1300 ls, t1310 listcon, t1320 listproj, t1400 prepend).

func TestTaskDone(t *testing.T) {
	cases := []struct {
		text string
		want bool
	}{
		{"x task", true},
		{"x 2011-03-02 task", true},
		{"x ", true}, // todo.sh treats a bare "x " marker as done
		{"x", false},
		{"xx task", false},
		{"X task", false}, // the marker is case-sensitive
		{" task", false},
		{"", false},
	}
	for _, c := range cases {
		if got := (Task{Text: c.text}).Done(); got != c.want {
			t.Errorf("Done(%q) = %v, want %v", c.text, got, c.want)
		}
	}
}

func TestTaskPriority(t *testing.T) {
	cases := []struct {
		text string
		want byte
		ok   bool
	}{
		{"(A) task", 'A', true},
		{"(Z) task", 'Z', true},
		{"(A) (B) task", 'A', true},
		{"(a) task", 0, false},  // lowercase is not a priority (t1250)
		{"(m)others", 0, false}, // "(m)" mid-word is not a priority
		{"(1) task", 0, false},
		{"(AA) task", 0, false},
		{"(A)task", 0, false}, // requires the trailing space
		{"task (A)", 0, false},
		{"x (A) task", 0, false}, // priority only at the line start
		{"", 0, false},
	}
	for _, c := range cases {
		got, ok := (Task{Text: c.text}).Priority()
		if got != c.want || ok != c.ok {
			t.Errorf("Priority(%q) = (%q, %v), want (%q, %v)", c.text, got, ok, c.want, c.ok)
		}
	}
}

func TestTaskDate(t *testing.T) {
	cases := []struct {
		text string
		want string
	}{
		{"2011-03-02 task", "2011-03-02"},
		{"(A) 2009-02-13 new task", "2009-02-13"}, // creation date after priority (t1400)
		{"20090213T044000.12Z 2026-08-08 task", "2026-08-08"},
		{"x (A) 20090213T044000.12Z 2026-08-08 task", "2026-08-08"},
		{"x 2011-03-02 task", "2011-03-02"}, // completion date after "x " (t1310)
		{"x (A) 2011-03-02 task", "2011-03-02"},
		{"1999-01-31 task", "1999-01-31"},
		{"2099-12-31 task", "2099-12-31"},
		{"2000-02-30 task", "2000-02-30"}, // month/day ranges are not validated
		{"2011-13-02 task", "2011-13-02"}, // the regex is loose, like todo.sh's
		{"9999-01-01 task", ""},           // year must start with 19 or 20
		{"2999-01-01 task", ""},           //
		{"2011-3-02 task", ""},            // month and day need two digits
		{"2011-03-2 task", ""},            //
		{"task 2011-03-02", ""},           // not at the start of the task
		{"(A) task", ""},                  //
		{"x task", ""},                    //
		{"", ""},                          //
	}
	for _, c := range cases {
		if got := (Task{Text: c.text}).Date(); got != c.want {
			t.Errorf("Date(%q) = %q, want %q", c.text, got, c.want)
		}
	}
}

func TestTaskUUID(t *testing.T) {
	cases := []struct {
		text string
		want string
	}{
		{"20090213T044000.12Z task", "20090213T044000.12Z"},
		{"(A) 20090213T044000.12Z task", "20090213T044000.12Z"},
		{"x (A) 20090213T044000.12Z 2026-08-08 task", "20090213T044000.12Z"},
		{"20090213T044000.1Z task", ""},
		{"task 20090213T044000.12Z", ""},
		{"2026-08-08 20090213T044000.12Z task", ""},
	}
	for _, c := range cases {
		if got := (Task{Text: c.text}).UUID(); got != c.want {
			t.Errorf("UUID(%q) = %q, want %q", c.text, got, c.want)
		}
	}
}

func TestTaskContexts(t *testing.T) {
	cases := []struct {
		text string
		want []string
	}{
		{"@con01 -- Some context 1 task", []string{"@con01"}},
		{"(A) @con01 +prj01 -- task", []string{"@con01"}},
		{"@1", []string{"@1"}},
		{"@c2", []string{"@c2"}},
		{"@con05@con06", []string{"@con05@con06"}}, // one word, one context (t1310)
		{"@a @b @a", []string{"@a", "@b", "@a"}},   // no dedupe: that is listcon's job
		{"@_", []string{"@_"}},                     // trailing underscore counts
		{"@", nil},
		{"@,", nil},                    // must end in [A-Za-z0-9_] (§6.2.4)
		{"@home)", nil},                //
		{"foo@bar", nil},               // the word must start with the sigil
		{"ginatrapani@gmail.com", nil}, // e-mail addresses are not contexts (t1310)
		{"(@school", nil},              // parenthesized sigils are not contexts (t1310)
		{"w:@OtherContributors", nil},
		{"a\t@tab", []string{"@tab"}}, // tabs separate words like spaces
		{"", nil},
	}
	for _, c := range cases {
		if got := (Task{Text: c.text}).Contexts(); !reflect.DeepEqual(got, c.want) {
			t.Errorf("Contexts(%q) = %v, want %v", c.text, got, c.want)
		}
	}
}

func TestTaskProjects(t *testing.T) {
	cases := []struct {
		text string
		want []string
	}{
		{"+prj01 -- Some project 1 task", []string{"+prj01"}},
		{"(A) @con01 +prj01 -- task", []string{"+prj01"}},
		{"+1", []string{"+1"}},
		{"+prj05+prj06", []string{"+prj05+prj06"}}, // one word, one project (t1320)
		{"+a +b +a", []string{"+a", "+b", "+a"}},
		{"+_", []string{"+_"}},
		{"+", nil},
		{"+,", nil},
		{"+foo.", nil},
		{"ginatrapani+todo@gmail.com", nil}, // embedded + is not a project (t1320)
		{"a\t+b", []string{"+b"}},
		{"", nil},
	}
	for _, c := range cases {
		if got := (Task{Text: c.text}).Projects(); !reflect.DeepEqual(got, c.want) {
			t.Errorf("Projects(%q) = %v, want %v", c.text, got, c.want)
		}
	}
}

func TestTaskNumberedLine(t *testing.T) {
	cases := []struct {
		task  Task
		width int
		want  string
	}{
		{Task{LineNumber: 1, Text: "foo"}, 1, "1 foo"},
		{Task{LineNumber: 1, Text: "foo"}, 2, " 1 foo"},
		{Task{LineNumber: 20, Text: "bar"}, 2, "20 bar"},
		{Task{LineNumber: 1, Text: "(A) @con01"}, 3, "  1 (A) @con01"},
	}
	for _, c := range cases {
		if got := c.task.NumberedLine(c.width); got != c.want {
			t.Errorf("NumberedLine(%+v, %d) = %q, want %q", c.task, c.width, got, c.want)
		}
	}
}

func TestNumberWidth(t *testing.T) {
	cases := []struct {
		total int
		want  int
	}{
		{0, 1},   // an empty file still pads to one digit (${#LINES} of "0")
		{3, 1},   //
		{20, 2},  //
		{99, 2},  //
		{100, 3}, //
		{9999, 4},
	}
	for _, c := range cases {
		if got := NumberWidth(c.total); got != c.want {
			t.Errorf("NumberWidth(%d) = %d, want %d", c.total, got, c.want)
		}
	}
}
