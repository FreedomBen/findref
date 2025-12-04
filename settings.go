package main

import (
	"path/filepath"
	"regexp"
	"strings"
)

var defaultExcludeDirs = []string{
	".git",
	".svn",
	".hg",
	".bzr",
	"CVS",
	"vendor",
	"node_modules",
	"build",
	"dist",
	"out",
	"coverage",
	"package-lock.json",
	"yarn.lock",
	"pnpm-lock.yaml",
	"bun.lockb",
	"composer.lock",
	"Gemfile.lock",
	"mix.lock",
	"Cargo.lock",
	"Pipfile.lock",
	"poetry.lock",
	"Podfile.lock",
	"go.sum",
	"gradle.lockfile",
}

var defaultExcludeEntries = buildExcludeEntries(defaultExcludeDirs...)

type excludeEntry struct {
	value     string
	matchBase bool
}

type Settings struct {
	Debug              bool
	TrackStats         bool
	FilenameOnly       bool
	IncludeHidden      bool
	MaxLineLength      int
	NoMaxLineLength    bool
	MatchRegex         *regexp.Regexp
	FilenameRegex      *regexp.Regexp
	HiddenFileRegex    *regexp.Regexp
	UseDefaultExcludes bool
	excludes           []excludeEntry
}

func NewSettings() *Settings {
	s := &Settings{
		Debug:              false,
		TrackStats:         false,
		FilenameOnly:       false,
		IncludeHidden:      false,
		MaxLineLength:      2000,
		NoMaxLineLength:    false,
		MatchRegex:         nil,
		FilenameRegex:      regexp.MustCompile(".*"),
		HiddenFileRegex:    regexp.MustCompile(`(^|\/)\.`),
		excludes:           []excludeEntry{},
		UseDefaultExcludes: true,
	}
	return s
}

func (s *Settings) PassesFileFilter(path string) bool {
	return s.FilenameRegex.MatchString(path)
}

func (s *Settings) IsHidden(path string) bool {
	// Ignore hidden files unless the IncludeHidden flag is set
	return path != "." && !s.IncludeHidden && s.HiddenFileRegex.MatchString(path)
}

func (s *Settings) AddExcludeDirs(dirs ...string) {
	s.AddExcludes(dirs...)
}

func (s *Settings) AddExcludes(paths ...string) {
	s.excludes = append(s.excludes, buildExcludeEntries(paths...)...)
}

func (s *Settings) ExcludeDirs() []string {
	return s.Excludes()
}

func (s *Settings) Excludes() []string {
	total := len(s.excludes)
	if s.UseDefaultExcludes {
		total += len(defaultExcludeDirs)
	}
	retval := make([]string, 0, total)
	for _, entry := range s.excludes {
		retval = append(retval, entry.value)
	}
	if s.UseDefaultExcludes {
		retval = append(retval, defaultExcludeDirs...)
	}
	return retval
}

func (s *Settings) ShouldExcludeDir(path string) bool {
	return s.shouldExclude(path)
}

func (s *Settings) ShouldExcludeFile(path string) bool {
	return s.shouldExclude(path)
}

func (s *Settings) shouldExclude(path string) bool {
	cleanedPath := filepath.Clean(path)
	pathBase := filepath.Base(cleanedPath)
	if s.matchesExclude(cleanedPath, pathBase, s.excludes) {
		return true
	}
	if s.UseDefaultExcludes && s.matchesExclude(cleanedPath, pathBase, defaultExcludeEntries) {
		return true
	}
	return false
}

func (s *Settings) matchesExclude(cleanedPath string, pathBase string, entries []excludeEntry) bool {
	for _, entry := range entries {
		if cleanedPath == entry.value {
			return true
		}
		if entry.matchBase && pathBase == entry.value {
			return true
		}
		if !entry.matchBase && !filepath.IsAbs(entry.value) {
			if strings.HasSuffix(cleanedPath, string(filepath.Separator)+entry.value) {
				return true
			}
		}
	}
	return false
}

func buildExcludeEntries(paths ...string) []excludeEntry {
	entries := make([]excludeEntry, 0, len(paths))
	for _, path := range paths {
		trimmed := strings.TrimSpace(path)
		if trimmed == "" {
			continue
		}
		cleaned := filepath.Clean(trimmed)
		if cleaned == "." {
			continue
		}
		entry := excludeEntry{
			value:     cleaned,
			matchBase: filepath.Base(cleaned) == cleaned,
		}
		entries = append(entries, entry)
	}
	return entries
}
