package main

import (
	"fmt"
	"strconv"
)

type Match struct {
	Path       string
	LineNumber int
	Line       []byte
	Match      []int
}

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

func (m *Match) hasMatch() bool {
	return m.LineNumber != 0 && len(m.Line) > 0 && len(m.Match) > 0
}
