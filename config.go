package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// FileConfig represents configuration values loaded from a YAML file.
// All fields are pointers so we can distinguish between "unset" and
// an explicit false/zero value.
type FileConfig struct {
	All             *bool    `yaml:"all"`
	Debug           *bool    `yaml:"debug"`
	Stats           *bool    `yaml:"stats"`
	Hidden          *bool    `yaml:"hidden"`
	Version         *bool    `yaml:"version"`
	NoColor         *bool    `yaml:"no_color"`
	MatchCase       *bool    `yaml:"match_case"`
	IgnoreCase      *bool    `yaml:"ignore_case"`
	FilenameOnly    *bool    `yaml:"filename_only"`
	MaxLineLength   *int     `yaml:"max_line_length"`
	NoMaxLineLength *bool    `yaml:"no_max_line_length"`
	Exclude         []string `yaml:"exclude"`
	MatchRegex      string   `yaml:"match_regex"`
	StartDir        string   `yaml:"start_dir"`
	FilenameRegex   string   `yaml:"filename_regex"`
}

func findConfigFile() (string, error) {
	candidates := []string{}

	cwd, err := os.Getwd()
	if err == nil {
		candidates = append(candidates, filepath.Join(cwd, ".findref.yaml"))
	}

	homeDir, homeErr := os.UserHomeDir()
	if homeErr == nil {
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigHome == "" {
			xdgConfigHome = filepath.Join(homeDir, ".config")
		}

		candidates = appendIfMissing(candidates, filepath.Join(xdgConfigHome, "findref", "config.yaml"))
		candidates = appendIfMissing(candidates, filepath.Join(homeDir, ".findref.yaml"))
	}

	for _, candidate := range candidates {
		info, statErr := os.Stat(candidate)
		if statErr == nil && !info.IsDir() {
			return candidate, nil
		}
		if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
			return "", fmt.Errorf("checking config path %q: %w", candidate, statErr)
		}
	}

	return "", nil
}

func appendIfMissing(paths []string, candidate string) []string {
	for _, existing := range paths {
		if existing == candidate {
			return paths
		}
	}
	return append(paths, candidate)
}

func loadConfigFile() (*FileConfig, string, error) {
	configPath, err := findConfigFile()
	if err != nil || configPath == "" {
		return nil, configPath, err
	}

	content, readErr := os.ReadFile(configPath)
	if readErr != nil {
		return nil, configPath, fmt.Errorf("reading config file %q: %w", configPath, readErr)
	}

	var cfg FileConfig
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, configPath, fmt.Errorf("parsing config file %q: %w", configPath, err)
	}

	return &cfg, configPath, nil
}

func normalizeWriteConfigArg() {
	args := os.Args
	for i := 1; i < len(args); i++ {
		switch {
		case args[i] == "--write-config" || args[i] == "-write-config":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				args[i] = "--write-config=local"
			}
		case strings.HasPrefix(args[i], "--write-config="):
			if trimmed := strings.TrimPrefix(args[i], "--write-config="); trimmed == "" {
				args[i] = "--write-config=local"
			}
		}
	}
	os.Args = args
}

func defaultConfigTemplate() string {
	var b strings.Builder
	b.WriteString(`# Base findref configuration
# Each key mirrors its long-form CLI flag. Command-line arguments still take precedence.

# Required match expression if not provided via CLI.
match_regex: ""

# Starting directory. Set to "." to search the current working directory.
start_dir: "."

# Optional filename filter (regex). Leave blank to search all files.
filename_regex: ""

# Search behavior flags
all: false                # implies --ignore-case and --hidden, disables default excludes
debug: false              # print verbose debug output
stats: false              # track and print basic statistics on exit
hidden: false             # include hidden files and directories
version: false            # print version and exit
no_color: false           # disable colorized output
match_case: false         # force case-sensitive matching (otherwise smart-case)
ignore_case: false        # force case-insensitive matching
filename_only: false      # print only filenames with matches
max_line_length: 2000     # maximum line length before clipping
no_max_line_length: false # disable line length limit entirely

# Additional paths or files to exclude. Defaults mirror the built-in list; remove or add as needed.
exclude:
`)
	for _, entry := range defaultExcludeDirs {
		fmt.Fprintf(&b, "  - %s\n", entry)
	}
	return b.String()
}

func writeDefaultConfig(target string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(target))
	if trimmed == "" {
		trimmed = "local"
	}
	switch trimmed {
	case "local":
		return writeConfigFile(localConfigPath())
	case "global":
		return writeConfigFile(globalConfigPath())
	default:
		return "", fmt.Errorf("invalid --write-config target %q (use 'local' or 'global')", target)
	}
}

func localConfigPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		return filepath.Join(".", ".findref.yaml")
	}
	return filepath.Join(cwd, ".findref.yaml")
}

func globalConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome == "" && homeDir != "" {
		xdgConfigHome = filepath.Join(homeDir, ".config")
	}
	if xdgConfigHome != "" {
		return filepath.Join(xdgConfigHome, "findref", "config.yaml")
	}
	// Final fallback
	return filepath.Join(homeDir, ".findref.yaml")
}

func writeConfigFile(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("unable to determine config path")
	}

	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return "", fmt.Errorf("config file already exists at %s; refusing to overwrite", path)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating config directory %q: %w", dir, err)
	}

	if err := os.WriteFile(path, []byte(defaultConfigTemplate()), 0o644); err != nil {
		return "", fmt.Errorf("writing config file %q: %w", path, err)
	}

	return path, nil
}

func configArgs(cfg *FileConfig) []string {
	if cfg == nil {
		return []string{}
	}

	args := []string{}

	addBool := func(val *bool, flagName string) {
		if val != nil && *val {
			args = append(args, flagName)
		}
	}

	addBool(cfg.All, "--all")
	addBool(cfg.Debug, "--debug")
	addBool(cfg.Stats, "--stats")
	addBool(cfg.Hidden, "--hidden")
	addBool(cfg.Version, "--version")
	addBool(cfg.NoColor, "--no-color")
	addBool(cfg.MatchCase, "--match-case")
	addBool(cfg.IgnoreCase, "--ignore-case")
	addBool(cfg.FilenameOnly, "--filename-only")
	addBool(cfg.NoMaxLineLength, "--no-max-line-length")

	if cfg.MaxLineLength != nil {
		args = append(args, "--max-line-length", strconv.Itoa(*cfg.MaxLineLength))
	}

	for _, ex := range cfg.Exclude {
		trimmed := strings.TrimSpace(ex)
		if trimmed != "" {
			args = append(args, "--exclude", trimmed)
		}
	}

	return args
}

func mergeArgsWithConfig(configArgs []string) {
	if len(configArgs) == 0 {
		return
	}

	merged := make([]string, 0, len(os.Args)+len(configArgs))
	merged = append(merged, os.Args[0])
	merged = append(merged, configArgs...)
	merged = append(merged, os.Args[1:]...)
	os.Args = merged
}
