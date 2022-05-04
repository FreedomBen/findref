package main

import (
	"fmt"
	"strconv"
)

const SideBuffer = 40

type Match struct {
	Path       string
	LineNumber int
	Line       []byte
	Match      []int
	MaxLength  int
}

// Prints the filename and line number, plus text with match in red
// Emulates exactly the behavior of grep
func (m *Match) printMatch() {
	fmt.Printf("%s%s%s%s:%s:%s%s%s%s%s%s\n",
		colors.Purple,
		m.Path,
		colors.Restore,
		colors.Green,
		strconv.Itoa(m.LineNumber),
		colors.Restore,
		string(m.Line[:m.Match[0]]),
		colors.LightRed,
		string(m.Line[m.Match[0]:m.Match[1]]),
		colors.Restore,
		string(m.Line[m.Match[1]:]),
	)
}

// Prints the filename and line number, but if the text on the left or right
// exceeds the size of SideBuffer then replace that part with a yellow ...
func (m *Match) printMatchClip() {
	startStr := "..."
	endStr := "..."
	start := m.Match[0] - SideBuffer
	end := m.Match[1] + SideBuffer

	if start < 0 {
		start = 0
		startStr = ""
	}
	if end > len(m.Line)-1 {
		end = len(m.Line) - 1
		endStr = ""
	}

	fmt.Printf("%s%s%s%s:%s:%s%s%s%s%s%s%s%s%s%s%s%s\n",
		colors.Purple,
		m.Path,
		colors.Restore,
		colors.Green,
		strconv.Itoa(m.LineNumber),
		colors.Restore,
		colors.Yellow,
		startStr,
		colors.Restore,
		string(m.Line[start:m.Match[0]]),
		colors.LightRed,
		string(m.Line[m.Match[0]:m.Match[1]]),
		colors.Restore,
		string(m.Line[m.Match[1]:end]),
		colors.Yellow,
		endStr,
		colors.Restore,
	)
}

// Prints the filename and line number, but replaces text with:
// "<match exceeded maximum length of 2000>"
func (m *Match) printMatchTooLong() {
	fmt.Printf("%s%s%s%s:%s:%s%s%s%s%s%s%s%s\n",
		colors.Purple,
		m.Path,
		colors.Restore,
		colors.Green,
		strconv.Itoa(m.LineNumber),
		colors.Restore,
		colors.Yellow,
		"<match exceeded maximum length of ",
		colors.Restore,
		strconv.Itoa(m.MaxLength),
		colors.Yellow,
		">",
		colors.Restore,
	)
}

func (m *Match) hasMatch() bool {
	return m.LineNumber != 0 && len(m.Line) > 0 && len(m.Match) > 0
}
