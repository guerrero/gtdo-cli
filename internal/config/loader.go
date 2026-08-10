package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const systemConfigPath = "/etc/gtdo/config.json"

func findConfigFile(opts Options, home, systemPath string) string {
	candidates := []string{opts.ConfigPath}
	if value, ok := envString("GTDO_CONFIG"); ok {
		candidates = append(candidates, expandHome(value, home))
	}
	candidates = append(candidates,
		filepath.Join(home, ".config", "gtdo", "config.json"),
		systemPath,
	)
	for _, path := range candidates {
		if path != "" && exists(path) {
			return path
		}
	}
	return ""
}

func load(opts Options, home, systemPath string) (Config, error) {
	path := findConfigFile(opts, home, systemPath)
	file := defaultFileConfig()
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return Config{}, fmt.Errorf("config: read %s: %w", path, err)
		}
		if err := decodeFileConfig(data, &file); err != nil {
			return Config{}, fmt.Errorf("config: parse %s: %w", path, err)
		}
	}
	cfg := resolve(opts, file, home)
	cfg.ConfigPath = path
	return cfg, nil
}
