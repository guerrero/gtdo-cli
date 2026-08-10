package todo

import "testing"

func TestTaskPrefix(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  taskPrefix
	}{
		{name: "task", input: "task", want: taskPrefix{rest: "task"}},
		{name: "done", input: "x task", want: taskPrefix{done: true, rest: "task"}},
		{name: "priority", input: "(A) task", want: taskPrefix{priority: "(A) ", rest: "task"}},
		{
			name:  "all metadata",
			input: "x (A) 20090213T044000.12Z 2026-08-08 task",
			want: taskPrefix{
				done:     true,
				priority: "(A) ",
				uuid:     "20090213T044000.12Z",
				date:     "2026-08-08",
				rest:     "task",
			},
		},
		{
			name:  "legacy date",
			input: "(A) 2026-08-08 task",
			want:  taskPrefix{priority: "(A) ", date: "2026-08-08", rest: "task"},
		},
		{
			name:  "malformed uuid remains text",
			input: "20090213T044000.1Z task",
			want:  taskPrefix{rest: "20090213T044000.1Z task"},
		},
		{
			name:  "uuid after body is not metadata",
			input: "task 20090213T044000.12Z",
			want:  taskPrefix{rest: "task 20090213T044000.12Z"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseTaskPrefix(tt.input); got != tt.want {
				t.Errorf("parseTaskPrefix(%q) = %#v, want %#v", tt.input, got, tt.want)
			}
		})
	}
}

func TestTaskPrefixRender(t *testing.T) {
	tests := []struct {
		name string
		p    taskPrefix
		rest string
		want string
	}{
		{name: "task", p: taskPrefix{}, rest: "task", want: "task"},
		{name: "done", p: taskPrefix{done: true}, rest: "task", want: "x task"},
		{name: "priority", p: taskPrefix{priority: "(A) "}, rest: "task", want: "(A) task"},
		{
			name: "all metadata",
			p: taskPrefix{
				done:     true,
				priority: "(A) ",
				uuid:     "20090213T044000.12Z",
				date:     "2026-08-08",
			},
			rest: "task",
			want: "x (A) 20090213T044000.12Z 2026-08-08 task",
		},
		{name: "empty rest", p: taskPrefix{uuid: "20090213T044000.12Z"}, want: "20090213T044000.12Z "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.render(tt.rest); got != tt.want {
				t.Errorf("taskPrefix.render(%q) = %q, want %q", tt.rest, got, tt.want)
			}
		})
	}
}
