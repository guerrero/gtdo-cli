package config

import (
	"strings"
	"testing"
)

func TestDecodeFileConfig(t *testing.T) {
	cfg := defaultFileConfig()
	err := decodeFileConfig([]byte(`{
      "dir":"~/todo",
      "files":{"todo":"todo.txt","done":"done.txt","report":"report.txt"},
      "behaviour":{"force":true,"preserveLineNumbers":false,"autoArchive":false,"enableUUID":true,"priorityOnAdd":"B","verbose":2,"defaultAction":"list","sourceVar":"done.txt","sentenceDelimiters":".;","allowedContexts":["@work"],"allowedProjects":["+gtdo"]},
      "colors":{"priA":"yellow","priZ":"cyan","colorDone":"light_grey","colorProject":"light_cyan","colorContext":"blue","colorDate":"red","colorNumber":"green","colorMeta":"brown","map":{"yellow":"\\033[1;43m"}}
    }`), &cfg)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Dir != "~/todo" || cfg.Files.Todo != "todo.txt" || cfg.Files.Done != "done.txt" || cfg.Files.Report != "report.txt" {
		t.Fatalf("paths = %#v/%q, want complete JSON paths", cfg.Files, cfg.Dir)
	}
	if !cfg.Behaviour.Force || cfg.Behaviour.PreserveLineNumbers || !cfg.Behaviour.EnableUUID || cfg.Behaviour.Verbose != 2 {
		t.Fatalf("behaviour = %#v, want decoded values", cfg.Behaviour)
	}
	if len(cfg.Behaviour.AllowedContexts) != 1 || cfg.Behaviour.AllowedContexts[0] != "@work" || len(cfg.Behaviour.AllowedProjects) != 1 || cfg.Behaviour.AllowedProjects[0] != "+gtdo" {
		t.Fatalf("allowed sigils = %#v/%#v, want decoded values", cfg.Behaviour.AllowedContexts, cfg.Behaviour.AllowedProjects)
	}
	if cfg.Colors.PriZ != "cyan" || cfg.Colors.Map["yellow"] != `\033[1;43m` {
		t.Fatalf("colors = %#v, want decoded values", cfg.Colors)
	}
}

func TestDecodeFileConfigRejectsInvalidDocuments(t *testing.T) {
	cases := map[string]string{
		"malformed":           `{"dir":`,
		"trailing value":      `{} {}`,
		"unknown top level":   `{"paths":{}}`,
		"unknown nested":      `{"behaviour":{"preserve_line_numbers":true}}`,
		"mis-cased top level": `{"Behaviour":{"verbose":2}}`,
		"mis-cased scalar":    `{"DIR":"~/todo"}`,
		"mis-cased nested":    `{"colors":{"PriA":"yellow"}}`,
		"wrong type":          `{"behaviour":{"verbose":"loud"}}`,
		"null scalar":         `{"dir":null}`,
		"null object":         `{"files":null}`,
		"null map value":      `{"colors":{"map":{"red":null}}}`,
		"incorrect nesting":   `{"files":{"todo":{"file":"todo.txt"}}}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := defaultFileConfig()
			err := decodeFileConfig([]byte(body), &cfg)
			if err == nil {
				t.Fatal("decodeFileConfig accepted invalid JSON")
			}
			if name == "unknown nested" && !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("error = %q, want unknown-field diagnostic", err)
			}
		})
	}
}
