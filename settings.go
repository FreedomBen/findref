package main

import (
	"regexp"
)

type Settings struct {
	Debug           bool
	TrackStats      bool
	FilenameOnly    bool
	IncludeHidden   bool
	MatchRegex      *regexp.Regexp
	FilenameRegex   *regexp.Regexp
	HiddenFileRegex *regexp.Regexp
}

func NewSettings() *Settings {
	return &Settings{
		Debug:           false,
		TrackStats:      false,
		FilenameOnly:    false,
		IncludeHidden:   false,
		MatchRegex:      nil,
		FilenameRegex:   regexp.MustCompile(".*"),
		HiddenFileRegex: regexp.MustCompile(`(^|\/)\.`),
	}
}

func (s *Settings) PassesFileFilter(path string) bool {
	return s.FilenameRegex.MatchString(path)
}

func (s *Settings) IsHidden(path string) bool {
	// Ignore hidden files unless the IncludeHidden flag is set
	return path != "." && !s.IncludeHidden && s.HiddenFileRegex.MatchString(path)
}
