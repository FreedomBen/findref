package main

import (
	"path/filepath"
	"regexp"
	"strings"
)

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
	excludeDirs     []string
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
		excludeDirs:     []string{},
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
	for _, dir := range dirs {
		trimmed := strings.TrimSpace(dir)
		if trimmed == "" {
			continue
		}
		cleaned := filepath.Clean(trimmed)
		if cleaned == "." {
			continue
		}
		s.excludeDirs = append(s.excludeDirs, cleaned)
	}
}

func (s *Settings) ExcludeDirs() []string {
	retval := make([]string, len(s.excludeDirs))
	copy(retval, s.excludeDirs)
	return retval
}

func (s *Settings) ShouldExcludeDir(path string) bool {
	if len(s.excludeDirs) == 0 {
		return false
	}
	cleanedPath := filepath.Clean(path)
	pathBase := filepath.Base(cleanedPath)
	for _, dir := range s.excludeDirs {
		if cleanedPath == dir {
			return true
		}
		if pathBase == filepath.Base(dir) {
			return true
		}
	}
	return false
}
