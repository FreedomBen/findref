package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// getMatchRegex: smart-case, ignore-case, match-case
// ---------------------------------------------------------------------------

func TestSmartCaseAllLowerIsCaseInsensitive(t *testing.T) {
	r := mustGetMatchRegex(t, false, false, "hello")
	for _, input := range []string{"hello", "Hello", "HELLO", "hElLo"} {
		if !r.MatchString(input) {
			t.Errorf("smart-case all-lower regex should match %q", input)
		}
	}
}

func TestSmartCaseWithUpperIsCaseSensitive(t *testing.T) {
	r := mustGetMatchRegex(t, false, false, "Hello")
	if !r.MatchString("Hello") {
		t.Error("should match exact case")
	}
	if r.MatchString("hello") {
		t.Error("smart-case with uppercase should NOT match all-lower")
	}
	if r.MatchString("HELLO") {
		t.Error("smart-case with uppercase should NOT match all-upper")
	}
}

func TestIgnoreCaseOverridesSmartCase(t *testing.T) {
	// Even though the regex has uppercase, ignore-case should still be insensitive
	r := mustGetMatchRegex(t, true, false, "Hello")
	for _, input := range []string{"hello", "Hello", "HELLO", "hElLo"} {
		if !r.MatchString(input) {
			t.Errorf("ignore-case should match %q", input)
		}
	}
}

func TestMatchCaseForcesExact(t *testing.T) {
	// All lowercase with match-case set should be case-sensitive
	r := mustGetMatchRegex(t, false, true, "hello")
	if !r.MatchString("hello") {
		t.Error("match-case should match exact")
	}
	if r.MatchString("Hello") {
		t.Error("match-case should NOT match different case")
	}
	if r.MatchString("HELLO") {
		t.Error("match-case should NOT match all-upper")
	}
}

func TestSmartCaseWithRegexMetachars(t *testing.T) {
	// Regex with special chars but all lowercase letters should be case-insensitive
	r := mustGetMatchRegex(t, false, false, `func\s+\w+`)
	if !r.MatchString("func main") {
		t.Error("should match lowercase")
	}
	if !r.MatchString("FUNC MAIN") {
		t.Error("smart-case all-lower should match uppercase input")
	}
	if !r.MatchString("Func Main") {
		t.Error("smart-case all-lower should match mixed case input")
	}
}

func TestSmartCaseWithUpperInRegex(t *testing.T) {
	// Regex containing uppercase letter → case-sensitive
	r := mustGetMatchRegex(t, false, false, `Func\w+`)
	if !r.MatchString("FuncMain") {
		t.Error("should match exact case")
	}
	if r.MatchString("funcmain") {
		t.Error("should NOT match lowercase")
	}
}

func TestGetMatchRegexSpecialPatterns(t *testing.T) {
	cases := []struct {
		name    string
		pattern string
		matches []string
		noMatch []string
	}{
		{
			name:    "literal dot",
			pattern: `fmt\.Println`,
			matches: []string{"fmt.Println"},
			noMatch: []string{"fmtXPrintln", "FMT.PRINTLN"}, // uppercase P triggers case-sensitive smart-case
		},
		{
			name:    "alternation",
			pattern: `(foo|bar)`,
			matches: []string{"foo", "bar", "FOO", "BAR", "foobar"},
			noMatch: []string{"baz", "qux"},
		},
		{
			name:    "anchored start",
			pattern: `^package`,
			matches: []string{"package main", "Package main"},
			noMatch: []string{" package main", "// package main"},
		},
		{
			name:    "anchored end",
			pattern: `\)$`,
			matches: []string{"func()", "end)"},
			noMatch: []string{"func() {", "end) //"},
		},
		{
			name:    "character class",
			pattern: `[0-9]+\.[0-9]+`,
			matches: []string{"v1.0", "version 3.14"},
			noMatch: []string{"v1", "nodot"},
		},
		{
			name:    "word boundary emulation",
			pattern: `\btodo\b`,
			matches: []string{"TODO: fix", "a todo item", "FIXME todo end"},
			noMatch: []string{"todolist", "mytodos"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := mustGetMatchRegex(t, false, false, tc.pattern)
			for _, s := range tc.matches {
				if !r.MatchString(s) {
					t.Errorf("expected pattern %q to match %q", tc.pattern, s)
				}
			}
			for _, s := range tc.noMatch {
				if r.MatchString(s) {
					t.Errorf("expected pattern %q NOT to match %q", tc.pattern, s)
				}
			}
		})
	}
}

func TestGetMatchRegexInvalidPatterns(t *testing.T) {
	invalids := []string{"(", "[", "(?P<name)", "*", "+", "?"}
	for _, p := range invalids {
		_, err := getMatchRegex(false, false, p)
		if err == nil {
			t.Errorf("expected error for invalid regex %q", p)
		}
	}
}

// ---------------------------------------------------------------------------
// checkForMatches: line-by-line matching behavior
// ---------------------------------------------------------------------------

func TestCheckForMatchesSingleMatch(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "test.txt")
	mustWriteFile(t, f, "line one\nline two has TODO\nline three\n")
	settings.MatchRegex = regexp.MustCompile("TODO")

	matches := checkForMatches(f)

	var found []Match
	for _, m := range matches {
		if m.hasMatch() {
			found = append(found, m)
		}
	}
	if len(found) != 1 {
		t.Fatalf("expected 1 match, got %d", len(found))
	}
	if found[0].LineNumber != 2 {
		t.Errorf("expected match on line 2, got %d", found[0].LineNumber)
	}
	if string(found[0].Line) != "line two has TODO" {
		t.Errorf("unexpected line content: %q", string(found[0].Line))
	}
	// Verify match indices point to "TODO" within the line
	matchStr := string(found[0].Line[found[0].Match[0]:found[0].Match[1]])
	if matchStr != "TODO" {
		t.Errorf("expected match text %q, got %q", "TODO", matchStr)
	}
}

func TestCheckForMatchesMultipleMatches(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "test.txt")
	mustWriteFile(t, f, "TODO first\nnothing\nTODO second\nTODO third\n")
	settings.MatchRegex = regexp.MustCompile("TODO")

	matches := checkForMatches(f)

	var found []Match
	for _, m := range matches {
		if m.hasMatch() {
			found = append(found, m)
		}
	}
	if len(found) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(found))
	}
	expectedLines := []int{1, 3, 4}
	for i, m := range found {
		if m.LineNumber != expectedLines[i] {
			t.Errorf("match %d: expected line %d, got %d", i, expectedLines[i], m.LineNumber)
		}
	}
}

func TestCheckForMatchesNoMatches(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "test.txt")
	mustWriteFile(t, f, "nothing here\njust text\n")
	settings.MatchRegex = regexp.MustCompile("NOTFOUND")

	matches := checkForMatches(f)

	for _, m := range matches {
		if m.hasMatch() {
			t.Fatal("expected no matches")
		}
	}
	if statistics.MatchCount() != 0 {
		t.Errorf("expected 0 match count, got %d", statistics.MatchCount())
	}
	if statistics.LineCount() != 2 {
		t.Errorf("expected 2 lines scanned, got %d", statistics.LineCount())
	}
}

func TestCheckForMatchesBinaryFileSkipped(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "binary.bin")
	// Write content with a null byte on the first line
	mustWriteFile(t, f, "TODO\x00binary\n")
	settings.MatchRegex = regexp.MustCompile("TODO")

	matches := checkForMatches(f)

	for _, m := range matches {
		if m.hasMatch() {
			t.Fatal("should not match in binary file")
		}
	}
	if statistics.SkippedNullCount() != 1 {
		t.Errorf("expected 1 skipped null file, got %d", statistics.SkippedNullCount())
	}
}

func TestCheckForMatchesBinaryDetectedMidFile(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "mixed.bin")
	// First line is text, second line has null byte
	mustWriteFile(t, f, "TODO match this\nother \x00 binary stuff\nTODO after binary\n")
	settings.MatchRegex = regexp.MustCompile("TODO")

	matches := checkForMatches(f)

	// It should find the match on line 1, then hit binary on line 2 and bail
	var found []Match
	for _, m := range matches {
		if m.hasMatch() {
			found = append(found, m)
		}
	}
	if len(found) != 1 {
		t.Fatalf("expected 1 match before binary detection, got %d", len(found))
	}
	if found[0].LineNumber != 1 {
		t.Errorf("expected match on line 1, got %d", found[0].LineNumber)
	}
}

func TestCheckForMatchesEmptyFile(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "empty.txt")
	mustWriteFile(t, f, "")
	settings.MatchRegex = regexp.MustCompile("anything")

	matches := checkForMatches(f)

	for _, m := range matches {
		if m.hasMatch() {
			t.Fatal("expected no matches in empty file")
		}
	}
}

func TestCheckForMatchesRegexOnPartialLine(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "test.txt")
	mustWriteFile(t, f, "the quick brown fox\njumps over the lazy dog\n")
	settings.MatchRegex = regexp.MustCompile(`qu\w+`)

	matches := checkForMatches(f)

	var found []Match
	for _, m := range matches {
		if m.hasMatch() {
			found = append(found, m)
		}
	}
	if len(found) != 1 {
		t.Fatalf("expected 1 match, got %d", len(found))
	}
	matchStr := string(found[0].Line[found[0].Match[0]:found[0].Match[1]])
	if matchStr != "quick" {
		t.Errorf("expected match text %q, got %q", "quick", matchStr)
	}
}

func TestCheckForMatchesCaseInsensitive(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "test.txt")
	mustWriteFile(t, f, "Hello World\nhello world\nHELLO WORLD\n")
	settings.MatchRegex = mustGetMatchRegex(t, false, false, "hello")

	matches := checkForMatches(f)

	var found []Match
	for _, m := range matches {
		if m.hasMatch() {
			found = append(found, m)
		}
	}
	if len(found) != 3 {
		t.Fatalf("expected 3 matches with smart-case lowercase, got %d", len(found))
	}
}

func TestCheckForMatchesCaseSensitive(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "test.txt")
	mustWriteFile(t, f, "Hello World\nhello world\nHELLO WORLD\n")
	settings.MatchRegex = mustGetMatchRegex(t, false, true, "hello")

	matches := checkForMatches(f)

	var found []Match
	for _, m := range matches {
		if m.hasMatch() {
			found = append(found, m)
		}
	}
	if len(found) != 1 {
		t.Fatalf("expected 1 match with match-case, got %d", len(found))
	}
	if found[0].LineNumber != 2 {
		t.Errorf("expected match on line 2, got %d", found[0].LineNumber)
	}
}

func TestCheckForMatchesFilenameOnly(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "test.txt")
	mustWriteFile(t, f, "TODO first\nTODO second\n")
	settings.MatchRegex = regexp.MustCompile("TODO")
	settings.FilenameOnly = true
	filenameOnlyFiles = make([]string, 0, 100)

	checkForMatches(f)

	if len(filenameOnlyFiles) != 2 {
		t.Fatalf("expected 2 filename entries (one per match), got %d", len(filenameOnlyFiles))
	}
	for _, name := range filenameOnlyFiles {
		if name != f {
			t.Errorf("expected filename %q, got %q", f, name)
		}
	}
}

func TestCheckForMatchesMatchIndex(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "test.txt")
	mustWriteFile(t, f, "prefix_TARGET_suffix\n")
	settings.MatchRegex = regexp.MustCompile("TARGET")

	matches := checkForMatches(f)

	var found []Match
	for _, m := range matches {
		if m.hasMatch() {
			found = append(found, m)
		}
	}
	if len(found) != 1 {
		t.Fatalf("expected 1 match, got %d", len(found))
	}
	m := found[0]
	if m.Match[0] != 7 {
		t.Errorf("expected match start at 7, got %d", m.Match[0])
	}
	if m.Match[1] != 13 {
		t.Errorf("expected match end at 13, got %d", m.Match[1])
	}
}

func TestCheckForMatchesNonexistentFile(t *testing.T) {
	resetTestState(t)
	settings.MatchRegex = regexp.MustCompile("anything")

	matches := checkForMatches("/nonexistent/path/file.txt")

	// Should return a "no match" entry (LineNumber == 0)
	for _, m := range matches {
		if m.hasMatch() {
			t.Fatal("should not have real matches for nonexistent file")
		}
	}
}

// ---------------------------------------------------------------------------
// containsNullByte
// ---------------------------------------------------------------------------

func TestContainsNullByteVariants(t *testing.T) {
	cases := []struct {
		name   string
		input  []byte
		expect bool
	}{
		{"empty", []byte{}, false},
		{"normal text", []byte("hello world"), false},
		{"null at start", []byte("\x00hello"), true},
		{"null at end", []byte("hello\x00"), true},
		{"null in middle", []byte("hel\x00lo"), true},
		{"only null", []byte{0}, true},
		{"multiple nulls", []byte{0, 0, 0}, true},
		{"high bytes no null", []byte{0xFF, 0xFE, 0x01}, false},
		{"unicode text", []byte("héllo wörld"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := containsNullByte(tc.input)
			if got != tc.expect {
				t.Errorf("containsNullByte(%q) = %v, want %v", tc.input, got, tc.expect)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Match.hasMatch
// ---------------------------------------------------------------------------

func TestHasMatch(t *testing.T) {
	cases := []struct {
		name   string
		match  Match
		expect bool
	}{
		{"valid match", Match{"f.go", 1, []byte("line"), []int{0, 4}, 2000}, true},
		{"zero line number", Match{"f.go", 0, []byte("line"), []int{0, 4}, 2000}, false},
		{"empty line", Match{"f.go", 1, []byte{}, []int{0, 4}, 2000}, false},
		{"nil match indices", Match{"f.go", 1, []byte("line"), nil, 2000}, false},
		{"empty match indices", Match{"f.go", 1, []byte("line"), []int{}, 2000}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.match.hasMatch()
			if got != tc.expect {
				t.Errorf("hasMatch() = %v, want %v", got, tc.expect)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Settings: exclusion logic
// ---------------------------------------------------------------------------

func TestExcludeBareName(t *testing.T) {
	s := NewSettings()
	s.UseDefaultExcludes = false
	s.AddExcludes("tmp")

	// Bare name matches at any depth
	if !s.ShouldExcludeDir("tmp") {
		t.Error("should exclude bare 'tmp'")
	}
	if !s.ShouldExcludeDir("project/tmp") {
		t.Error("should exclude 'project/tmp'")
	}
	if !s.ShouldExcludeDir("/abs/path/tmp") {
		t.Error("should exclude '/abs/path/tmp'")
	}
	// Should not match partial
	if s.ShouldExcludeDir("tmpdir") {
		t.Error("should NOT exclude 'tmpdir'")
	}
	if s.ShouldExcludeDir("my_tmp") {
		t.Error("should NOT exclude 'my_tmp'")
	}
}

func TestExcludePathWithSeparator(t *testing.T) {
	s := NewSettings()
	s.UseDefaultExcludes = false
	s.AddExcludes("src/generated")

	if !s.ShouldExcludeDir("project/src/generated") {
		t.Error("should exclude 'project/src/generated' as suffix match")
	}
	if !s.ShouldExcludeDir("src/generated") {
		t.Error("should exclude exact 'src/generated'")
	}
	if s.ShouldExcludeDir("other/generated") {
		t.Error("should NOT exclude 'other/generated'")
	}
}

func TestExcludeFileBareName(t *testing.T) {
	s := NewSettings()
	s.UseDefaultExcludes = false
	s.AddExcludes("README.md")

	if !s.ShouldExcludeFile("README.md") {
		t.Error("should exclude bare 'README.md'")
	}
	if !s.ShouldExcludeFile("docs/README.md") {
		t.Error("should exclude 'docs/README.md'")
	}
	if !s.ShouldExcludeFile("/abs/docs/README.md") {
		t.Error("should exclude '/abs/docs/README.md'")
	}
	if s.ShouldExcludeFile("README.txt") {
		t.Error("should NOT exclude 'README.txt'")
	}
}

func TestExcludePatternsRegex(t *testing.T) {
	s := NewSettings()
	s.UseDefaultExcludes = false
	if err := s.AddExcludePatterns(`_test\.go$`, `\.min\.js$`, `^vendor/`); err != nil {
		t.Fatal(err)
	}

	if !s.ShouldExcludeFile("src/foo_test.go") {
		t.Error("should exclude test file")
	}
	if !s.ShouldExcludeFile("dist/bundle.min.js") {
		t.Error("should exclude minified JS")
	}
	if !s.ShouldExcludeFile("vendor/lib.go") {
		t.Error("should exclude vendor path")
	}
	if s.ShouldExcludeFile("src/foo.go") {
		t.Error("should NOT exclude regular go file")
	}
	if s.ShouldExcludeFile("src/app.js") {
		t.Error("should NOT exclude regular JS")
	}
}

func TestExcludeEmptyAndWhitespace(t *testing.T) {
	s := NewSettings()
	s.UseDefaultExcludes = false
	s.AddExcludes("", "  ", "  valid  ")

	// Empty/whitespace should be ignored, "valid" should work
	if !s.ShouldExcludeDir("project/valid") {
		t.Error("should exclude 'valid' after trimming")
	}
	// Make sure the empty entries didn't break anything
	if s.ShouldExcludeDir("project/src") {
		t.Error("should NOT exclude 'src'")
	}
}

func TestExcludePatternEmpty(t *testing.T) {
	s := NewSettings()
	// Adding empty/whitespace patterns should not error
	if err := s.AddExcludePatterns("", "  "); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.excludePatterns) != 0 {
		t.Errorf("expected 0 patterns, got %d", len(s.excludePatterns))
	}
}

func TestDefaultExcludesAllPresent(t *testing.T) {
	s := NewSettings()
	expected := []string{
		".git", ".svn", ".hg", ".bzr", "CVS",
		"vendor", "node_modules", "build", "dist", "out", "coverage",
		"package-lock.json", "yarn.lock", "pnpm-lock.yaml", "bun.lockb",
		"composer.lock", "Gemfile.lock", "mix.lock", "Cargo.lock",
		"Pipfile.lock", "poetry.lock", "Podfile.lock", "go.sum", "gradle.lockfile",
	}
	for _, name := range expected {
		if !s.ShouldExcludeDir(filepath.Join("project", name)) {
			t.Errorf("expected default exclude for %q", name)
		}
	}
}

func TestDefaultExcludesDoNotExcludeCommonDirs(t *testing.T) {
	s := NewSettings()
	safe := []string{"src", "lib", "cmd", "internal", "pkg", "test", "tests", "app"}
	for _, name := range safe {
		if s.ShouldExcludeDir(filepath.Join("project", name)) {
			t.Errorf("did NOT expect default exclude for %q", name)
		}
	}
}

// ---------------------------------------------------------------------------
// Settings: inclusion logic
// ---------------------------------------------------------------------------

func TestIncludeBareName(t *testing.T) {
	s := NewSettings()
	s.UseDefaultExcludes = false
	s.AddIncludes("main.go")

	if !s.ShouldIncludeFile("main.go") {
		t.Error("should include bare 'main.go'")
	}
	if !s.ShouldIncludeFile("cmd/main.go") {
		t.Error("should include 'cmd/main.go'")
	}
	if !s.ShouldIncludeFile("/abs/path/main.go") {
		t.Error("should include '/abs/path/main.go'")
	}
	if s.ShouldIncludeFile("main_test.go") {
		t.Error("should NOT include 'main_test.go'")
	}
	if s.ShouldIncludeFile("other.go") {
		t.Error("should NOT include 'other.go'")
	}
}

func TestIncludePathWithSeparator(t *testing.T) {
	s := NewSettings()
	s.UseDefaultExcludes = false
	s.AddIncludes("src/main.go")

	if !s.ShouldIncludeFile("project/src/main.go") {
		t.Error("should include 'project/src/main.go' as suffix match")
	}
	if !s.ShouldIncludeFile("src/main.go") {
		t.Error("should include exact 'src/main.go'")
	}
	if s.ShouldIncludeFile("other/main.go") {
		t.Error("should NOT include 'other/main.go'")
	}
}

func TestIncludePatternsRegex(t *testing.T) {
	s := NewSettings()
	s.UseDefaultExcludes = false
	if err := s.AddIncludePatterns(`\.go$`, `\.py$`); err != nil {
		t.Fatal(err)
	}

	if !s.ShouldIncludeFile("src/foo.go") {
		t.Error("should include Go file")
	}
	if !s.ShouldIncludeFile("lib/bar.py") {
		t.Error("should include Python file")
	}
	if s.ShouldIncludeFile("src/app.js") {
		t.Error("should NOT include JS file")
	}
	if s.ShouldIncludeFile("README.md") {
		t.Error("should NOT include markdown file")
	}
}

func TestIncludePatternInvalid(t *testing.T) {
	s := NewSettings()
	err := s.AddIncludePatterns("[invalid")
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestIncludePatternEmpty(t *testing.T) {
	s := NewSettings()
	if err := s.AddIncludePatterns("", "  "); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(s.includePatterns) != 0 {
		t.Errorf("expected 0 patterns, got %d", len(s.includePatterns))
	}
}

func TestNoIncludesMatchesEverything(t *testing.T) {
	s := NewSettings()
	s.UseDefaultExcludes = false

	// With no includes set, everything should be included
	if !s.ShouldIncludeFile("anything.txt") {
		t.Error("with no includes, all files should be included")
	}
	if !s.ShouldIncludeFile("src/foo.go") {
		t.Error("with no includes, all files should be included")
	}
	if s.HasIncludes() {
		t.Error("HasIncludes should be false when no includes are set")
	}
}

func TestIncludeAndExcludeCombined(t *testing.T) {
	s := NewSettings()
	s.UseDefaultExcludes = false
	if err := s.AddIncludePatterns(`\.go$`); err != nil {
		t.Fatal(err)
	}
	s.AddExcludes("vendor")

	// Go file outside vendor: included
	if !s.ShouldIncludeFile("src/foo.go") {
		t.Error("should include Go file in src")
	}
	// Go file in vendor: included by include pattern (exclude for "vendor" applies
	// at the directory level during walk, not at the file level)
	if !s.ShouldIncludeFile("vendor/lib.go") {
		t.Error("ShouldIncludeFile should return true for vendor Go file (exclude is checked separately)")
	}
	// Non-Go file: not included
	if s.ShouldIncludeFile("src/readme.md") {
		t.Error("should NOT include non-Go file")
	}
}

func TestIncludeMultipleNames(t *testing.T) {
	s := NewSettings()
	s.UseDefaultExcludes = false
	s.AddIncludes("Makefile", "Rakefile")

	if !s.ShouldIncludeFile("Makefile") {
		t.Error("should include Makefile")
	}
	if !s.ShouldIncludeFile("project/Rakefile") {
		t.Error("should include Rakefile")
	}
	if s.ShouldIncludeFile("README.md") {
		t.Error("should NOT include README.md")
	}
}

func TestIncludeNameAndPatternCombined(t *testing.T) {
	s := NewSettings()
	s.UseDefaultExcludes = false
	s.AddIncludes("Makefile")
	if err := s.AddIncludePatterns(`\.go$`); err != nil {
		t.Fatal(err)
	}

	// Matches by name
	if !s.ShouldIncludeFile("Makefile") {
		t.Error("should include Makefile by name")
	}
	// Matches by pattern
	if !s.ShouldIncludeFile("src/foo.go") {
		t.Error("should include Go file by pattern")
	}
	// Matches neither
	if s.ShouldIncludeFile("README.md") {
		t.Error("should NOT include README.md")
	}
}

// ---------------------------------------------------------------------------
// Settings: hidden file detection
// ---------------------------------------------------------------------------

func TestIsHiddenVariants(t *testing.T) {
	s := NewSettings()
	cases := []struct {
		path   string
		hidden bool
	}{
		{".", false},          // current dir is never hidden
		{".git", true},       // dotdir at root
		{".gitignore", true}, // dotfile at root
		{"src/.env", true},
		{"src/main.go", false},
		{"/abs/.hidden/file.txt", true},
		{"/abs/visible/file.txt", false},
		{"a/b/c/.secret", true},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			got := s.IsHidden(tc.path)
			if got != tc.hidden {
				t.Errorf("IsHidden(%q) = %v, want %v", tc.path, got, tc.hidden)
			}
		})
	}
}

func TestIsHiddenWithIncludeHidden(t *testing.T) {
	s := NewSettings()
	s.IncludeHidden = true
	hidden := []string{".git", ".env", "src/.hidden"}
	for _, path := range hidden {
		if s.IsHidden(path) {
			t.Errorf("with IncludeHidden=true, IsHidden(%q) should be false", path)
		}
	}
}

// ---------------------------------------------------------------------------
// Settings: filename regex filter
// ---------------------------------------------------------------------------

func TestPassesFileFilterVariants(t *testing.T) {
	s := NewSettings()

	// Default filter matches everything
	if !s.PassesFileFilter("anything.txt") {
		t.Error("default filter should match anything")
	}

	// C++ file filter
	s.FilenameRegex = regexp.MustCompile(`\.(cpp|hpp|h|c)$`)
	passes := []string{"main.cpp", "header.hpp", "util.h", "lib.c", "dir/main.cpp"}
	fails := []string{"main.go", "test.py", "readme.md", "Makefile"}

	for _, f := range passes {
		if !s.PassesFileFilter(f) {
			t.Errorf("expected %q to pass C++ filter", f)
		}
	}
	for _, f := range fails {
		if s.PassesFileFilter(f) {
			t.Errorf("expected %q to fail C++ filter", f)
		}
	}
}

// ---------------------------------------------------------------------------
// processFile: integration of walk + filter + exclusion + hidden
// ---------------------------------------------------------------------------

func TestProcessFileExcludedDir(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()

	// Add a custom exclude
	settings.AddExcludes("skipme")

	dir := filepath.Join(tmpDir, "skipme")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(dir)

	ret := processFile(dir, info, nil)
	if ret != filepath.SkipDir {
		t.Errorf("expected SkipDir for excluded dir, got %v", ret)
	}
}

func TestProcessFileHiddenDir(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()

	dir := filepath.Join(tmpDir, ".hidden")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(dir)

	ret := processFile(dir, info, nil)
	if ret != filepath.SkipDir {
		t.Errorf("expected SkipDir for hidden dir, got %v", ret)
	}
}

func TestProcessFileHiddenDirIncluded(t *testing.T) {
	resetTestState(t)
	settings.IncludeHidden = true
	tmpDir := t.TempDir()

	dir := filepath.Join(tmpDir, ".hidden")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	info, _ := os.Stat(dir)

	ret := processFile(dir, info, nil)
	if ret != FILE_PROCESSING_COMPLETE {
		t.Errorf("expected nil for hidden dir with IncludeHidden, got %v", ret)
	}
}

func TestProcessFileNormalFileQueued(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()

	f := filepath.Join(tmpDir, "code.go")
	mustWriteFile(t, f, "package main\n")
	info, _ := os.Stat(f)

	ret := processFile(f, info, nil)
	if ret != FILE_PROCESSING_COMPLETE {
		t.Errorf("expected nil, got %v", ret)
	}
	if len(filesToScan) != 1 {
		t.Fatalf("expected 1 file queued, got %d", len(filesToScan))
	}
}

func TestProcessFileHiddenFileSkipped(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()

	f := filepath.Join(tmpDir, ".env")
	mustWriteFile(t, f, "SECRET=123\n")
	info, _ := os.Stat(f)

	ret := processFile(f, info, nil)
	if ret != FILE_PROCESSING_COMPLETE {
		t.Errorf("expected nil, got %v", ret)
	}
	if len(filesToScan) != 0 {
		t.Errorf("hidden file should not be queued")
	}
}

func TestProcessFileFilteredByFilenameRegex(t *testing.T) {
	resetTestState(t)
	settings.FilenameRegex = regexp.MustCompile(`\.go$`)
	tmpDir := t.TempDir()

	goFile := filepath.Join(tmpDir, "main.go")
	mustWriteFile(t, goFile, "package main\n")
	goInfo, _ := os.Stat(goFile)

	txtFile := filepath.Join(tmpDir, "readme.txt")
	mustWriteFile(t, txtFile, "hello\n")
	txtInfo, _ := os.Stat(txtFile)

	processFile(goFile, goInfo, nil)
	processFile(txtFile, txtInfo, nil)

	if len(filesToScan) != 1 {
		t.Fatalf("expected 1 file queued (only .go), got %d", len(filesToScan))
	}
	if filesToScan[0].Path != goFile {
		t.Errorf("expected %q queued, got %q", goFile, filesToScan[0].Path)
	}
}

func TestProcessFileWithError(t *testing.T) {
	resetTestState(t)
	ret := processFile("/nonexistent", nil, os.ErrNotExist)
	if ret != FILE_PROCESSING_COMPLETE {
		t.Errorf("expected nil on error, got %v", ret)
	}
}

// ---------------------------------------------------------------------------
// uniq helper
// ---------------------------------------------------------------------------

func TestUniq(t *testing.T) {
	cases := []struct {
		name  string
		input []string
		want  int
	}{
		{"empty", []string{}, 0},
		{"no duplicates", []string{"a", "b", "c"}, 3},
		{"all same", []string{"a", "a", "a"}, 1},
		{"some duplicates", []string{"a", "b", "a", "c", "b"}, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := uniq(tc.input)
			if len(got) != tc.want {
				t.Errorf("uniq(%v) has %d elements, want %d", tc.input, len(got), tc.want)
			}
			// Verify all unique elements are present
			seen := map[string]bool{}
			for _, s := range tc.input {
				seen[s] = true
			}
			for _, s := range got {
				if !seen[s] {
					t.Errorf("unexpected element %q in result", s)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Integration: end-to-end matching via runFindrefMain
// ---------------------------------------------------------------------------

func TestIntegrationBasicMatch(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "main.go"), "package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n")

	stdout, stderr := runFindrefMain(t, []string{"--no-color", "fmt.Println", tmpDir})
	if stderr != "" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
	if !strings.Contains(stdout, "fmt.Println") {
		t.Fatalf("expected stdout to contain match, got %q", stdout)
	}
	if !strings.Contains(stdout, "main.go") {
		t.Fatalf("expected stdout to contain filename, got %q", stdout)
	}
}

func TestIntegrationSmartCaseLower(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "test.txt"), "Hello\nhello\nHELLO\n")

	stdout, _ := runFindrefMain(t, []string{"--no-color", "--filename-only", "hello", tmpDir})
	lines := splitLines(stdout)
	// Smart-case with all-lowercase should match all three lines → file appears
	expectContains(t, lines, filepath.Join(tmpDir, "test.txt"))
}

func TestIntegrationSmartCaseUpper(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "a.txt"), "Hello World\n")
	mustWriteFile(t, filepath.Join(tmpDir, "b.txt"), "hello world\n")

	stdout, _ := runFindrefMain(t, []string{"--no-color", "--filename-only", "Hello", tmpDir})
	lines := splitLines(stdout)
	expectContains(t, lines, filepath.Join(tmpDir, "a.txt"))
	expectNotContains(t, lines, filepath.Join(tmpDir, "b.txt"))
}

func TestIntegrationIgnoreCaseFlag(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "a.txt"), "Hello World\n")
	mustWriteFile(t, filepath.Join(tmpDir, "b.txt"), "hello world\n")

	stdout, _ := runFindrefMain(t, []string{"--no-color", "--filename-only", "-c", "Hello", tmpDir})
	lines := splitLines(stdout)
	expectContains(t, lines, filepath.Join(tmpDir, "a.txt"))
	expectContains(t, lines, filepath.Join(tmpDir, "b.txt"))
}

func TestIntegrationMatchCaseFlag(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "a.txt"), "Hello World\n")
	mustWriteFile(t, filepath.Join(tmpDir, "b.txt"), "hello world\n")
	mustWriteFile(t, filepath.Join(tmpDir, "c.txt"), "HELLO WORLD\n")

	stdout, _ := runFindrefMain(t, []string{"--no-color", "--filename-only", "-m", "hello", tmpDir})
	lines := splitLines(stdout)
	expectNotContains(t, lines, filepath.Join(tmpDir, "a.txt"))
	expectContains(t, lines, filepath.Join(tmpDir, "b.txt"))
	expectNotContains(t, lines, filepath.Join(tmpDir, "c.txt"))
}

func TestIntegrationAllFlag(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, ".hidden.txt"), "TODO hidden\n")
	mustWriteFile(t, filepath.Join(tmpDir, "vendor", "lib.go"), "TODO vendor\n")
	mustWriteFile(t, filepath.Join(tmpDir, "main.go"), "TODO main\n")

	stdout, _ := runFindrefMain(t, []string{"--no-color", "--filename-only", "--all", "TODO", tmpDir})
	lines := splitLines(stdout)
	expectContains(t, lines, filepath.Join(tmpDir, ".hidden.txt"))
	expectContains(t, lines, filepath.Join(tmpDir, "vendor", "lib.go"))
	expectContains(t, lines, filepath.Join(tmpDir, "main.go"))
}

func TestIntegrationFilenameRegex(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "main.go"), "TODO: go file\n")
	mustWriteFile(t, filepath.Join(tmpDir, "main.py"), "TODO: py file\n")
	mustWriteFile(t, filepath.Join(tmpDir, "main.js"), "TODO: js file\n")

	stdout, _ := runFindrefMain(t, []string{"--no-color", "--filename-only", "TODO", tmpDir, `\.go$`})
	lines := splitLines(stdout)
	expectContains(t, lines, filepath.Join(tmpDir, "main.go"))
	expectNotContains(t, lines, filepath.Join(tmpDir, "main.py"))
	expectNotContains(t, lines, filepath.Join(tmpDir, "main.js"))
}

func TestIntegrationHiddenFlag(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, ".env"), "SECRET=TODO\n")
	mustWriteFile(t, filepath.Join(tmpDir, "main.go"), "TODO\n")

	// Without --hidden
	stdout, _ := runFindrefMain(t, []string{"--no-color", "--filename-only", "TODO", tmpDir})
	lines := splitLines(stdout)
	expectContains(t, lines, filepath.Join(tmpDir, "main.go"))
	expectNotContains(t, lines, filepath.Join(tmpDir, ".env"))

	// With --hidden
	stdout, _ = runFindrefMain(t, []string{"--no-color", "--filename-only", "--hidden", "TODO", tmpDir})
	lines = splitLines(stdout)
	expectContains(t, lines, filepath.Join(tmpDir, "main.go"))
	expectContains(t, lines, filepath.Join(tmpDir, ".env"))
}

func TestIntegrationBinaryFileSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "text.txt"), "TODO text\n")
	mustWriteFile(t, filepath.Join(tmpDir, "binary.bin"), "TODO\x00binary\n")

	stdout, _ := runFindrefMain(t, []string{"--no-color", "--filename-only", "TODO", tmpDir})
	lines := splitLines(stdout)
	expectContains(t, lines, filepath.Join(tmpDir, "text.txt"))
	expectNotContains(t, lines, filepath.Join(tmpDir, "binary.bin"))
}

func TestIntegrationMultipleMatchesInFile(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "code.go"),
		"// TODO: first\nfunc main() {}\n// TODO: second\n// TODO: third\n")

	stdout, _ := runFindrefMain(t, []string{"--no-color", "TODO", tmpDir})
	count := strings.Count(stdout, "TODO")
	if count != 3 {
		t.Errorf("expected 3 TODO matches in output, got %d", count)
	}
}

func TestIntegrationNestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "a", "b", "c", "deep.txt"), "TODO deep\n")
	mustWriteFile(t, filepath.Join(tmpDir, "top.txt"), "TODO top\n")

	stdout, _ := runFindrefMain(t, []string{"--no-color", "--filename-only", "TODO", tmpDir})
	lines := splitLines(stdout)
	expectContains(t, lines, filepath.Join(tmpDir, "a", "b", "c", "deep.txt"))
	expectContains(t, lines, filepath.Join(tmpDir, "top.txt"))
}

func TestIntegrationDefaultExcludesApply(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "src", "main.go"), "TODO main\n")
	mustWriteFile(t, filepath.Join(tmpDir, "node_modules", "pkg", "index.js"), "TODO node\n")
	mustWriteFile(t, filepath.Join(tmpDir, ".git", "config"), "TODO git\n")
	mustWriteFile(t, filepath.Join(tmpDir, "build", "output.js"), "TODO build\n")

	stdout, _ := runFindrefMain(t, []string{"--no-color", "--filename-only", "TODO", tmpDir})
	lines := splitLines(stdout)
	expectContains(t, lines, filepath.Join(tmpDir, "src", "main.go"))
	expectNotContains(t, lines, filepath.Join(tmpDir, "node_modules", "pkg", "index.js"))
	expectNotContains(t, lines, filepath.Join(tmpDir, ".git", "config"))
	expectNotContains(t, lines, filepath.Join(tmpDir, "build", "output.js"))
}

func TestIntegrationRegexMatch(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "test.txt"),
		"func main() {}\nfunc helper() {}\nvar x = 1\n")

	stdout, _ := runFindrefMain(t, []string{"--no-color", `^func \w+`, tmpDir})
	if !strings.Contains(stdout, "func main") {
		t.Error("expected to find 'func main'")
	}
	if !strings.Contains(stdout, "func helper") {
		t.Error("expected to find 'func helper'")
	}
	if strings.Contains(stdout, "var x") {
		t.Error("should NOT match 'var x'")
	}
}

func TestIntegrationEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	stdout, stderr := runFindrefMain(t, []string{"--no-color", "TODO", tmpDir})
	if stderr != "" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Fatalf("expected empty output for empty dir, got %q", stdout)
	}
}

func TestIntegrationStatsFlag(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "test.txt"), "TODO line1\nno match\nTODO line3\n")

	stdout, _ := runFindrefMain(t, []string{"--no-color", "--stats", "TODO", tmpDir})
	if !strings.Contains(stdout, "Lines scanned:") {
		t.Error("expected stats output to contain 'Lines scanned:'")
	}
	if !strings.Contains(stdout, "Files scanned:") {
		t.Error("expected stats output to contain 'Files scanned:'")
	}
	if !strings.Contains(stdout, "Matches found:") {
		t.Error("expected stats output to contain 'Matches found:'")
	}
}

func TestIntegrationExcludePatternWithAll(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "main.go"), "TODO main\n")
	mustWriteFile(t, filepath.Join(tmpDir, "main_test.go"), "TODO test\n")
	mustWriteFile(t, filepath.Join(tmpDir, ".hidden.go"), "TODO hidden\n")

	// --all includes hidden and disables defaults, but exclude-pattern should still work
	stdout, _ := runFindrefMain(t, []string{
		"--no-color", "--filename-only", "--all",
		"--exclude-pattern", `_test\.go$`,
		"TODO", tmpDir,
	})
	lines := splitLines(stdout)
	expectContains(t, lines, filepath.Join(tmpDir, "main.go"))
	expectNotContains(t, lines, filepath.Join(tmpDir, "main_test.go"))
	expectContains(t, lines, filepath.Join(tmpDir, ".hidden.go"))
}

func TestIntegrationNoMatchOutput(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "test.txt"), "nothing relevant here\n")

	stdout, stderr := runFindrefMain(t, []string{"--no-color", "ZZZZNOTFOUND", tmpDir})
	if stderr != "" {
		t.Fatalf("unexpected stderr: %q", stderr)
	}
	if strings.TrimSpace(stdout) != "" {
		t.Errorf("expected empty output when no matches, got %q", stdout)
	}
}

func TestIntegrationMultipleExcludes(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "keep", "a.txt"), "TODO keep\n")
	mustWriteFile(t, filepath.Join(tmpDir, "skip1", "b.txt"), "TODO skip1\n")
	mustWriteFile(t, filepath.Join(tmpDir, "skip2", "c.txt"), "TODO skip2\n")

	stdout, _ := runFindrefMain(t, []string{
		"--no-color", "--filename-only",
		"--exclude", "skip1", "--exclude", "skip2",
		"TODO", tmpDir,
	})
	lines := splitLines(stdout)
	expectContains(t, lines, filepath.Join(tmpDir, "keep", "a.txt"))
	expectNotContains(t, lines, filepath.Join(tmpDir, "skip1", "b.txt"))
	expectNotContains(t, lines, filepath.Join(tmpDir, "skip2", "c.txt"))
}

func TestIntegrationMatchOutputFormat(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "code.go")
	mustWriteFile(t, f, "line one\nTODO: fix this\nline three\n")

	stdout, _ := runFindrefMain(t, []string{"--no-color", "TODO", tmpDir})
	// Output format should be: path:linenum:content
	if !strings.Contains(stdout, f+":2:") {
		t.Errorf("expected output to contain %q, got %q", f+":2:", stdout)
	}
}

func TestIntegrationSymlinkHandling(t *testing.T) {
	tmpDir := t.TempDir()
	realFile := filepath.Join(tmpDir, "real.txt")
	mustWriteFile(t, realFile, "TODO in real file\n")
	linkFile := filepath.Join(tmpDir, "link.txt")
	if err := os.Symlink(realFile, linkFile); err != nil {
		t.Skip("symlinks not supported on this system")
	}

	stdout, _ := runFindrefMain(t, []string{"--no-color", "--filename-only", "TODO", tmpDir})
	lines := splitLines(stdout)
	// Both the real file and symlink should show up
	expectContains(t, lines, realFile)
	expectContains(t, lines, linkFile)
}

func TestIntegrationDanglingSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "good.txt"), "TODO good\n")
	linkFile := filepath.Join(tmpDir, "bad_link.txt")
	if err := os.Symlink("/nonexistent/target", linkFile); err != nil {
		t.Skip("symlinks not supported on this system")
	}

	// Should not crash, should find the good file
	stdout, _ := runFindrefMain(t, []string{"--no-color", "--filename-only", "TODO", tmpDir})
	lines := splitLines(stdout)
	expectContains(t, lines, filepath.Join(tmpDir, "good.txt"))
}

func TestIntegrationLargeLineClipping(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "long.txt")
	// Create a line much longer than the default 2000 char limit
	longLine := strings.Repeat("x", 1000) + "TARGET" + strings.Repeat("y", 1500)
	mustWriteFile(t, f, longLine+"\n")

	stdout, _ := runFindrefMain(t, []string{"--no-color", "TARGET", tmpDir})
	// Should still find the match, just with clipped output
	if !strings.Contains(stdout, "TARGET") {
		t.Error("expected to find TARGET in clipped output")
	}
	// The full line should not appear (it would be 2506 chars)
	if strings.Contains(stdout, strings.Repeat("x", 1000)) {
		t.Error("expected long prefix to be clipped")
	}
}

func TestIntegrationUTF8Content(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "utf8.txt"),
		"café résumé\nnaïve über\nTODO: héllo wörld\n")

	stdout, _ := runFindrefMain(t, []string{"--no-color", "--filename-only", "TODO", tmpDir})
	lines := splitLines(stdout)
	expectContains(t, lines, filepath.Join(tmpDir, "utf8.txt"))
}

func TestIntegrationMatchInLastLineNoNewline(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "nonewline.txt")
	// File without trailing newline
	mustWriteFile(t, f, "first line\nTODO last line no newline")

	stdout, _ := runFindrefMain(t, []string{"--no-color", "TODO", tmpDir})
	if !strings.Contains(stdout, "TODO") {
		t.Error("expected to match TODO in last line without trailing newline")
	}
}

func TestIntegrationMatchEmptyLineRegex(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "test.txt"), "hello\n\nworld\n")

	// Match lines containing "hello" or "world"
	stdout, _ := runFindrefMain(t, []string{"--no-color", "--filename-only", "hello|world", tmpDir})
	lines := splitLines(stdout)
	expectContains(t, lines, filepath.Join(tmpDir, "test.txt"))
}

func TestIntegrationHiddenSubdirSkipped(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, ".hidden", "deep", "file.txt"), "TODO hidden deep\n")
	mustWriteFile(t, filepath.Join(tmpDir, "visible.txt"), "TODO visible\n")

	stdout, _ := runFindrefMain(t, []string{"--no-color", "--filename-only", "TODO", tmpDir})
	lines := splitLines(stdout)
	expectContains(t, lines, filepath.Join(tmpDir, "visible.txt"))
	expectNotContains(t, lines, filepath.Join(tmpDir, ".hidden", "deep", "file.txt"))
}

func TestIntegrationExcludePatternMultiple(t *testing.T) {
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "app.go"), "TODO app\n")
	mustWriteFile(t, filepath.Join(tmpDir, "app_test.go"), "TODO test\n")
	mustWriteFile(t, filepath.Join(tmpDir, "bundle.min.js"), "TODO minified\n")
	mustWriteFile(t, filepath.Join(tmpDir, "app.js"), "TODO normal js\n")

	stdout, _ := runFindrefMain(t, []string{
		"--no-color", "--filename-only",
		"-E", `_test\.go$`,
		"-E", `\.min\.js$`,
		"TODO", tmpDir,
	})
	lines := splitLines(stdout)
	expectContains(t, lines, filepath.Join(tmpDir, "app.go"))
	expectContains(t, lines, filepath.Join(tmpDir, "app.js"))
	expectNotContains(t, lines, filepath.Join(tmpDir, "app_test.go"))
	expectNotContains(t, lines, filepath.Join(tmpDir, "bundle.min.js"))
}
