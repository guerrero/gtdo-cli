package cli

import (
	"testing"

	"github.com/guerrero/gtdo/internal/config"
)

func TestPreparse(t *testing.T) {
	cases := []struct {
		name   string
		args   []string
		action string
		rest   []string
		check  func(t *testing.T, o *config.Options)
	}{
		{"no args", nil, "", nil, nil},
		{
			"plain before action",
			[]string{"-p", "list", "foo"},
			"list",
			[]string{"foo"},
			func(t *testing.T, o *config.Options) {
				if !o.Plain || !o.PlainSet {
					t.Error("plain not set")
				}
			},
		},
		{"flag after action is positional", []string{"list", "-p"}, "list", []string{"-p"}, nil},
		{
			"cluster",
			[]string{"-npf", "del", "2"},
			"del",
			[]string{"2"},
			func(t *testing.T, o *config.Options) {
				if o.Preserve || !o.PreserveSet || !o.Force || !o.ForceSet || !o.Plain || !o.PlainSet {
					t.Error("cluster flags not applied")
				}
			},
		},
		{
			"counters",
			[]string{"-vv", "list"},
			"list", nil,
			func(t *testing.T, o *config.Options) {
				if o.VerboseCount != 2 {
					t.Error("want 2 -v")
				}
			},
		},
		{
			"toggles odd",
			[]string{"-@", "-+", "-P", "list"},
			"list", nil,
			func(t *testing.T, o *config.Options) {
				if !o.HideContexts || !o.HideProjects || !o.HidePriority {
					t.Error("odd toggles should hide")
				}
			},
		},
		{
			"toggles even",
			[]string{"-@", "-@", "list"},
			"list", nil,
			func(t *testing.T, o *config.Options) {
				if o.HideContexts {
					t.Error("even toggles should show")
				}
			},
		},
		{
			"d with separate arg",
			[]string{"-d", "cfg.toml", "list"},
			"list", nil,
			func(t *testing.T, o *config.Options) {
				if o.ConfigPath != "cfg.toml" {
					t.Error("config path")
				}
			},
		},
		{
			"d in cluster",
			[]string{"-Pd", "cfg.toml", "list"},
			"list", nil,
			func(t *testing.T, o *config.Options) {
				if o.ConfigPath != "cfg.toml" || !o.HidePriority {
					t.Error("cluster -d")
				}
			},
		},
		{"d missing arg", []string{"-d"}, "", nil, func(_ *testing.T, _ *config.Options) {}},
		{"h short-circuits", []string{"-h", "-d", "x", "bogus"}, "shorthelp", nil, nil},
		{
			"V short-circuits",
			[]string{"-V", "list"},
			"", nil,
			func(t *testing.T, o *config.Options) {
				if !o.Version {
					t.Error("version flag")
				}
			},
		},
		{"unknown flag", []string{"-q", "list"}, "", nil, nil},
		{"removed date flag t", []string{"-t", "list"}, "", nil, nil},
		{"removed date flag T", []string{"-T", "list"}, "", nil, nil},
		{"double dash", []string{"--", "-p"}, "-p", nil, nil},
		{"lone dash is action", []string{"-"}, "-", nil, nil},
		{"x is no-op", []string{"-x", "list"}, "list", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts, action, rest, err := Preparse(tc.args)
			if tc.name == "d missing arg" || tc.name == "unknown flag" || tc.name == "removed date flag t" || tc.name == "removed date flag T" {
				if err == nil {
					t.Fatal("want usage error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if action != tc.action {
				t.Errorf("action = %q, want %q", action, tc.action)
			}
			if len(rest) != len(tc.rest) {
				t.Fatalf("rest = %v, want %v", rest, tc.rest)
			}
			for i := range rest {
				if rest[i] != tc.rest[i] {
					t.Fatalf("rest = %v, want %v", rest, tc.rest)
				}
			}
			if tc.check != nil {
				tc.check(t, opts)
			}
		})
	}
}
