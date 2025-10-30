package main

import (
	"bytes"
	"flag"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

func resetTestState(t *testing.T) {
	t.Helper()
	settings = NewSettings()
	statistics = NewStatistics()
	colors = NewColors()
	filenameOnlyFiles = make([]string, 0, 100)
	filesToScan = make([]FileToScan, 0, 100)
}

func TestContainsNullByte(t *testing.T) {
	if containsNullByte([]byte("Here is a string....")) {
		t.Fail()
	}
	if !containsNullByte([]byte("Here is a \x00string....")) {
		t.Fail()
	}
}

func TestPassesFileFilter(t *testing.T) {
	s := NewSettings()
	s.FilenameRegex = regexp.MustCompile("abcd")
	if !s.PassesFileFilter("abcd.txt") {
		t.Fail()
	}
	if s.PassesFileFilter("abc.txt") {
		t.Fail()
	}
	if !s.PassesFileFilter("abcd") {
		t.Fail()
	}
	s.FilenameRegex = regexp.MustCompile(`.*\.txt`)
	if !s.PassesFileFilter("abcd.txt") {
		t.Fail()
	}
	if s.PassesFileFilter("abc.tx") {
		t.Fail()
	}
	if s.PassesFileFilter("abcd") {
		t.Fail()
	}
}

func TestIsHidden(t *testing.T) {
	s := NewSettings()
	if s.IsHidden("/home/ben") {
		t.Fail()
	}
	if s.IsHidden("/home/ben/nothing.txt") {
		t.Fail()
	}
	if !s.IsHidden("/home/.ben/nothing.txt") {
		t.Fail()
	}
	if !s.IsHidden("/home/ben/.nothing.txt") {
		t.Fail()
	}
	if !s.IsHidden("/home/.ben/.nothing.txt") {
		t.Fail()
	}
}

func TestShouldExcludeDirDefaults(t *testing.T) {
	s := NewSettings()
	if !s.ShouldExcludeDir("./.git") {
		t.Fatalf("expected .git to be excluded by default")
	}
	if !s.ShouldExcludeDir("/tmp/project/.svn") {
		t.Fatalf("expected .svn to be excluded by default")
	}
	if s.ShouldExcludeDir("./vendor") {
		t.Fatalf("did not expect vendor to be excluded by default")
	}
}

func TestShouldExcludeDirUserProvided(t *testing.T) {
	s := NewSettings()
	s.AddExcludeDirs("vendor", "./build/")
	if !s.ShouldExcludeDir("/tmp/project/vendor") {
		t.Fatalf("expected vendor directory to be excluded when provided")
	}
	if !s.ShouldExcludeDir("./build") {
		t.Fatalf("expected build directory to be excluded when provided")
	}
	if s.ShouldExcludeDir("/tmp/project/src") {
		t.Fatalf("did not expect src directory to be excluded")
	}
}

func TestGetMatchRegex(t *testing.T) {
	r1 := getMatchRegex(false, false, "HEllo")
	if !r1.MatchString("HEllo") {
		t.Fail()
	}
	if r1.MatchString("hello") {
		t.Fail()
	}
	r2 := getMatchRegex(false, false, "hello")
	//  verify smart case works
	if !r2.MatchString("abcHEllo") {
		t.Fail()
	}
	if !r2.MatchString("abchello") {
		t.Fail()
	}
	if !r2.MatchString("abcHELLO") {
		t.Fail()
	}
	if r2.MatchString("abc") {
		t.Fail()
	}
	r3 := getMatchRegex(true, false, "hello")
	if !r3.MatchString("HEllo") {
		t.Fail()
	}
	if !r3.MatchString("abchello") {
		t.Fail()
	}
	if !r3.MatchString("abcHELLO") {
		t.Fail()
	}
	if r3.MatchString("abc") {
		t.Fail()
	}
	r4 := getMatchRegex(false, true, "hello")
	if r4.MatchString("HEllo") {
		t.Fail()
	}
	if !r4.MatchString("abchello") {
		t.Fail()
	}
	if r4.MatchString("abcHELLO") {
		t.Fail()
	}
	if r4.MatchString("abc") {
		t.Fail()
	}
}

func TestCheckForMatches(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "notes.txt")
	content := []byte("no match here\nTODO: finish tests\n")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	settings.MatchRegex = regexp.MustCompile("TODO")

	matches := checkForMatches(testFile)
	if statistics.LineCount() != 2 {
		t.Fatalf("expected 2 lines scanned, got %d", statistics.LineCount())
	}
	if statistics.MatchCount() != 1 {
		t.Fatalf("expected 1 match recorded, got %d", statistics.MatchCount())
	}

	found := false
	for _, match := range matches {
		if match.hasMatch() {
			found = true
			if match.Path != testFile {
				t.Fatalf("expected match path %q, got %q", testFile, match.Path)
			}
			if match.LineNumber != 2 {
				t.Fatalf("expected match on line 2, got %d", match.LineNumber)
			}
			break
		}
	}
	if !found {
		t.Fatalf("expected at least one match in returned slice")
	}
}

func TestProcessFile(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()

	hiddenDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(hiddenDir, 0755); err != nil {
		t.Fatalf("creating hidden dir: %v", err)
	}
	hiddenInfo, err := os.Stat(hiddenDir)
	if err != nil {
		t.Fatalf("stat hidden dir: %v", err)
	}
	if ret := processFile(hiddenDir, hiddenInfo, nil); ret != filepath.SkipDir {
		t.Fatalf("expected hidden dir to return filepath.SkipDir, got %v", ret)
	}

	visibleFile := filepath.Join(tmpDir, "main.go")
	if err := os.WriteFile(visibleFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("writing temp file: %v", err)
	}
	fileInfo, err := os.Stat(visibleFile)
	if err != nil {
		t.Fatalf("stat temp file: %v", err)
	}
	if ret := processFile(visibleFile, fileInfo, nil); ret != FILE_PROCESSING_COMPLETE {
		t.Fatalf("expected file processing to return nil, got %v", ret)
	}
	if len(filesToScan) != 1 {
		t.Fatalf("expected one file queued, got %d", len(filesToScan))
	}
	if filesToScan[0].Path != visibleFile {
		t.Fatalf("expected queued path %q, got %q", visibleFile, filesToScan[0].Path)
	}
	if statistics.FilesToScanCount() != 1 {
		t.Fatalf("expected statistics to record 1 file to scan, got %d", statistics.FilesToScanCount())
	}
	if statistics.FileCount() != 1 {
		t.Fatalf("expected statistics to record 1 processed file, got %d", statistics.FileCount())
	}
}

type stubFileInfo struct {
	size int64
	mode os.FileMode
}

func (s stubFileInfo) Name() string       { return "stub" }
func (s stubFileInfo) Size() int64        { return s.size }
func (s stubFileInfo) Mode() os.FileMode  { return s.mode }
func (s stubFileInfo) ModTime() time.Time { return time.Time{} }
func (s stubFileInfo) IsDir() bool        { return s.mode.IsDir() }
func (s stubFileInfo) Sys() interface{}   { return nil }

func TestScannerBufferLimits(t *testing.T) {
	cases := []struct {
		name        string
		info        os.FileInfo
		setup       func(t *testing.T)
		wantInitial int
		wantMax     int
	}{
		{
			name: "small file respects minimum cap",
			info: stubFileInfo{size: 1024},
			setup: func(t *testing.T) {
				resetTestState(t)
			},
			wantInitial: scannerMinimumInitialCap,
			wantMax:     scannerMinimumInitialCap,
		},
		{
			name: "large file uses file size",
			info: stubFileInfo{size: 10 * 1024 * 1024},
			setup: func(t *testing.T) {
				resetTestState(t)
			},
			wantInitial: scannerDefaultInitialCap,
			wantMax:     10 * 1024 * 1024,
		},
		{
			name: "huge file clamps to max token cap",
			info: stubFileInfo{size: int64(scannerMaxTokenCap) + (32 * 1024 * 1024)},
			setup: func(t *testing.T) {
				resetTestState(t)
			},
			wantInitial: scannerDefaultInitialCap,
			wantMax:     scannerMaxTokenCap,
		},
		{
			name: "no stat info with no max line length uses max token cap",
			info: nil,
			setup: func(t *testing.T) {
				resetTestState(t)
				settings.NoMaxLineLength = true
			},
			wantInitial: scannerDefaultInitialCap,
			wantMax:     scannerMaxTokenCap,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup(t)
			t.Cleanup(func() {
				resetTestState(t)
			})
			initial, max := scannerBufferLimits(tc.info)
			if initial != tc.wantInitial {
				t.Fatalf("initial cap = %d, want %d", initial, tc.wantInitial)
			}
			if max != tc.wantMax {
				t.Fatalf("max token = %d, want %d", max, tc.wantMax)
			}
		})
	}
}

func TestIntegrationExcludeFlag(t *testing.T) {
	tmpDir := t.TempDir()
	rootFile := filepath.Join(tmpDir, "main.go")
	vendorFile := filepath.Join(tmpDir, "vendor", "lib.go")
	thirdPartyFile := filepath.Join(tmpDir, "third_party", "lib.js")
	gitFile := filepath.Join(tmpDir, ".git", "config")

	mustWriteFile(t, rootFile, "package main\n// TODO: root implementation\n")
	mustWriteFile(t, vendorFile, "// TODO: vendor implementation\n")
	mustWriteFile(t, thirdPartyFile, "// TODO: third party logic\n")
	mustWriteFile(t, gitFile, "TODO: should be hidden\n")

	stdout, stderr := runFindrefMain(t, []string{"--no-color", "--filename-only", "TODO", tmpDir})
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	lines := splitLines(stdout)
	expectContains(t, lines, rootFile)
	expectContains(t, lines, vendorFile)
	expectContains(t, lines, thirdPartyFile)
	expectNotContains(t, lines, gitFile)

	stdout, stderr = runFindrefMain(t, []string{"--no-color", "--filename-only", "--exclude", "vendor", "--exclude", "third_party", "TODO", tmpDir})
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	lines = splitLines(stdout)
	expectContains(t, lines, rootFile)
	expectNotContains(t, lines, vendorFile)
	expectNotContains(t, lines, thirdPartyFile)
	expectNotContains(t, lines, gitFile)
}

func runFindrefMain(t *testing.T, args []string) (string, string) {
	t.Helper()
	resetTestState(t)

	oldArgs := os.Args
	os.Args = append([]string{"findref"}, args...)
	oldCommandLine := flag.CommandLine
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	oldUsage := flag.Usage

	stdout, stderr := captureOutput(func() {
		main()
	})

	flag.CommandLine = oldCommandLine
	flag.Usage = oldUsage
	os.Args = oldArgs

	return stdout, stderr
}

func captureOutput(fn func()) (string, string) {
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	os.Stdout = wOut
	os.Stderr = wErr

	outChan := make(chan string)
	errChan := make(chan string)

	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rOut)
		outChan <- buf.String()
	}()

	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, rErr)
		errChan <- buf.String()
	}()

	var panicVal interface{}
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicVal = r
			}
		}()
		fn()
	}()

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	stdout := <-outChan
	stderr := <-errChan

	_ = rOut.Close()
	_ = rErr.Close()

	if panicVal != nil {
		panic(panicVal)
	}

	return stdout, stderr
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("creating directory for %q: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing file %q: %v", path, err)
	}
}

func splitLines(output string) []string {
	output = strings.TrimSpace(output)
	if output == "" {
		return []string{}
	}
	return strings.Split(output, "\n")
}

func expectContains(t *testing.T, lines []string, target string) {
	t.Helper()
	for _, line := range lines {
		if line == target {
			return
		}
	}
	t.Fatalf("expected output to contain %q; got %+v", target, lines)
}

func expectNotContains(t *testing.T, lines []string, target string) {
	t.Helper()
	for _, line := range lines {
		if line == target {
			t.Fatalf("expected output not to contain %q; got %+v", target, lines)
		}
	}
}
