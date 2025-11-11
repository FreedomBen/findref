package main

import (
	"path/filepath"
	"regexp"
	"strings"
)

type excludeEntry struct {
	value     string
	matchBase bool
}

type Settings struct {
	Debug           bool
	TrackStats      bool
	FilenameOnly    bool
	IncludeHidden   bool
	MaxLineLength   int
	NoMaxLineLength bool
	MatchRegex      *regexp.Regexp
	FilenameRegex   *regexp.Regexp
	HiddenFileRegex *regexp.Regexp
	excludes        []excludeEntry
}

func NewSettings() *Settings {
	s := &Settings{
		Debug:           false,
		TrackStats:      false,
		FilenameOnly:    false,
		IncludeHidden:   false,
		MaxLineLength:   2000,
		NoMaxLineLength: false,
		MatchRegex:      nil,
		FilenameRegex:   regexp.MustCompile(".*"),
		HiddenFileRegex: regexp.MustCompile(`(^|\/)\.`),
		excludes:        []excludeEntry{},
	}
	s.AddExcludeDirs(".git", ".svn", ".hg", ".bzr", "CVS")
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
		s.excludes = append(s.excludes, entry)
	}
}

func (s *Settings) ExcludeDirs() []string {
	return s.Excludes()
}

func (s *Settings) Excludes() []string {
	retval := make([]string, len(s.excludes))
	for idx, entry := range s.excludes {
		retval[idx] = entry.value
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
	if len(s.excludes) == 0 {
		return false
	}
	cleanedPath := filepath.Clean(path)
	pathBase := filepath.Base(cleanedPath)
	for _, entry := range s.excludes {
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
