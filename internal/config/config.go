// Package config resolves gtdo's configuration: CLI flags (pre-parsed),
// environment variables, TOML file, and defaults, in that order.
package config

// Options carries the global flag results produced by cli.Preparse.
type Options struct {
	ConfigPath     string
	Force          bool
	ForceSet       bool
	Plain          bool
	PlainSet       bool
	Preserve       bool
	PreserveSet    bool
	AutoArchive    bool
	AutoArchiveSet bool
	DateOnAdd      bool
	DateOnAddSet   bool
	VerboseCount   int
	HideProjects   bool
	HideContexts   bool
	HidePriority   bool
	Version        bool
}
