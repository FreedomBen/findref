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
