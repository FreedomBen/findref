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

func mustGetMatchRegex(t *testing.T, ignoreCase bool, matchCase bool, usersRegex string) *regexp.Regexp {
	t.Helper()
	r, err := getMatchRegex(ignoreCase, matchCase, usersRegex)
	if err != nil {
		t.Fatalf("unexpected error compiling regex %q: %v", usersRegex, err)
	}
	return r
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

func TestDefaultExcludePaths(t *testing.T) {
	s := NewSettings()
	for _, entry := range defaultExcludeDirs {
		path := filepath.Join("/tmp/project", entry)
		if !s.ShouldExcludeDir(path) {
			t.Fatalf("expected %s to be excluded by default as directory", entry)
		}
		if !s.ShouldExcludeFile(path) {
			t.Fatalf("expected %s to be excluded by default as file", entry)
		}
	}
	if s.ShouldExcludeDir("./src") {
		t.Fatalf("did not expect src to be excluded by default")
	}
	if s.ShouldExcludeFile("./src/file.go") {
		t.Fatalf("did not expect src/file.go to be excluded by default")
	}
}

func TestDisableDefaultExcludes(t *testing.T) {
	s := NewSettings()
	s.UseDefaultExcludes = false
	for _, dir := range defaultExcludeDirs {
		path := filepath.Join("/tmp/project", dir)
		if s.ShouldExcludeDir(path) {
			t.Fatalf("did not expect %s to be excluded when defaults disabled", dir)
		}
	}
	s.AddExcludeDirs("vendor")
	if !s.ShouldExcludeDir(filepath.Join("/tmp/project", "vendor")) {
		t.Fatalf("expected vendor to be excluded when user provided")
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

func TestShouldExcludeFileUserProvided(t *testing.T) {
	s := NewSettings()
	s.AddExcludes("README.md", "src/generated.go")
	if !s.ShouldExcludeFile("/tmp/project/docs/README.md") {
		t.Fatalf("expected README.md files to be excluded when provided")
	}
	if !s.ShouldExcludeFile("/tmp/project/src/generated.go") {
		t.Fatalf("expected generated.go to be excluded when provided with relative path")
	}
	if s.ShouldExcludeFile("/tmp/project/src/generated_test.go") {
		t.Fatalf("did not expect generated_test.go to be excluded")
	}
}

func TestGetMatchRegex(t *testing.T) {
	r1 := mustGetMatchRegex(t, false, false, "HEllo")
	if !r1.MatchString("HEllo") {
		t.Fail()
	}
	if r1.MatchString("hello") {
		t.Fail()
	}
	r2 := mustGetMatchRegex(t, false, false, "hello")
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
	r3 := mustGetMatchRegex(t, true, false, "hello")
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
	r4 := mustGetMatchRegex(t, false, true, "hello")
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

func TestGetMatchRegexInvalid(t *testing.T) {
	_, err := getMatchRegex(false, false, "(")
	if err == nil {
		t.Fatalf("expected invalid regex to return an error")
	}
	if !strings.Contains(err.Error(), "invalid match regex") {
		t.Fatalf("expected error to mention invalid match regex, got %q", err.Error())
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
	expectNotContains(t, lines, vendorFile)
	expectContains(t, lines, thirdPartyFile)
	expectNotContains(t, lines, gitFile)

	stdout, stderr = runFindrefMain(t, []string{"--no-color", "--filename-only", "--all", "TODO", tmpDir})
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	lines = splitLines(stdout)
	expectContains(t, lines, rootFile)
	expectContains(t, lines, vendorFile)
	expectContains(t, lines, thirdPartyFile)
	expectContains(t, lines, gitFile)

	stdout, stderr = runFindrefMain(t, []string{"--no-color", "--filename-only", "--exclude", "vendor", "--exclude", "third_party", "TODO", tmpDir})
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	lines = splitLines(stdout)
	expectContains(t, lines, rootFile)
	expectNotContains(t, lines, vendorFile)
	expectNotContains(t, lines, thirdPartyFile)
	expectNotContains(t, lines, gitFile)

	stdout, stderr = runFindrefMain(t, []string{"--no-color", "--filename-only", "--all", "--exclude", "vendor", "TODO", tmpDir})
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	lines = splitLines(stdout)
	expectContains(t, lines, rootFile)
	expectNotContains(t, lines, vendorFile)
	expectContains(t, lines, thirdPartyFile)
	expectContains(t, lines, gitFile)
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

func runFindrefMainInDir(t *testing.T, args []string, dir string) (string, string) {
	t.Helper()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir to %q: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore working dir: %v", err)
		}
	})

	return runFindrefMain(t, args)
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

func TestFindConfigFilePrefersCwd(t *testing.T) {
	base := t.TempDir()
	workDir := filepath.Join(base, "work")
	homeDir := filepath.Join(base, "home")

	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("mkdir work dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(homeDir, ".config", "findref"), 0o755); err != nil {
		t.Fatalf("mkdir xdg dir: %v", err)
	}

	mustWriteFile(t, filepath.Join(workDir, ".findref.yaml"), "debug: true\n")
	mustWriteFile(t, filepath.Join(homeDir, ".config", "findref", "config.yaml"), "stats: true\n")

	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))

	restore := chdirHelper(t, workDir)
	defer restore()

	path, err := findConfigFile()
	if err != nil {
		t.Fatalf("findConfigFile returned error: %v", err)
	}

	expected := filepath.Join(workDir, ".findref.yaml")
	if path != expected {
		t.Fatalf("expected cwd config %q, got %q", expected, path)
	}
}

func TestFindConfigFileXDGHome(t *testing.T) {
	base := t.TempDir()
	workDir := filepath.Join(base, "work")
	homeDir := filepath.Join(base, "home")

	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("mkdir work dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(homeDir, ".config", "findref"), 0o755); err != nil {
		t.Fatalf("mkdir xdg dir: %v", err)
	}

	configPath := filepath.Join(homeDir, ".config", "findref", "config.yaml")
	mustWriteFile(t, configPath, "stats: true\n")

	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))

	restore := chdirHelper(t, workDir)
	defer restore()

	path, err := findConfigFile()
	if err != nil {
		t.Fatalf("findConfigFile returned error: %v", err)
	}

	if path != configPath {
		t.Fatalf("expected XDG config %q, got %q", configPath, path)
	}
}

func TestFindConfigFileHomeFallback(t *testing.T) {
	base := t.TempDir()
	workDir := filepath.Join(base, "work")
	homeDir := filepath.Join(base, "home")

	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("mkdir work dir: %v", err)
	}

	configPath := filepath.Join(homeDir, ".findref.yaml")
	mustWriteFile(t, configPath, "debug: true\n")

	t.Setenv("HOME", homeDir)
	// leave XDG unset so default resolves inside home/.config which is empty

	restore := chdirHelper(t, workDir)
	defer restore()

	path, err := findConfigFile()
	if err != nil {
		t.Fatalf("findConfigFile returned error: %v", err)
	}

	if path != configPath {
		t.Fatalf("expected home fallback config %q, got %q", configPath, path)
	}
}

func TestConfigFileAppliedToRun(t *testing.T) {
	base := t.TempDir()
	workDir := filepath.Join(base, "work")
	homeDir := filepath.Join(base, "home")

	mustWriteFile(t, filepath.Join(workDir, ".findref.yaml"), "match_regex: todo\nignore_case: true\nfilename_only: true\nexclude:\n  - skipme\n")
	mustWriteFile(t, filepath.Join(workDir, "main.go"), "package main\n// TODO: main\n")
	mustWriteFile(t, filepath.Join(workDir, "skipme", "note.txt"), "todo: hidden\n")

	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))

	stdout, stderr := runFindrefMainInDir(t, []string{"--no-color"}, workDir)
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	lines := splitLines(stdout)
	expectContains(t, lines, "main.go")
	expectNotContains(t, lines, filepath.Join("skipme", "note.txt"))
}

func TestConfigFileCliOverrides(t *testing.T) {
	base := t.TempDir()
	workDir := filepath.Join(base, "work")
	homeDir := filepath.Join(base, "home")

	mustWriteFile(t, filepath.Join(workDir, ".findref.yaml"), "match_regex: TODO\nstart_dir: .\nmax_line_length: 5\n")
	mustWriteFile(t, filepath.Join(workDir, "main.go"), "package main\n// TODO: adjust\n")

	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config"))

	stdout, stderr := runFindrefMainInDir(t, []string{"--no-color", "--max-line-length", "50"}, workDir)
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if stdout == "" {
		t.Fatalf("expected some output due to match")
	}
	if settings.MaxLineLength != 50 {
		t.Fatalf("expected CLI max line length 50 to override config, got %d", settings.MaxLineLength)
	}
	resetTestState(t)
}

func chdirHelper(t *testing.T, target string) func() {
	t.Helper()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(target); err != nil {
		t.Fatalf("chdir to %q: %v", target, err)
	}

	return func() {
		if err := os.Chdir(oldWD); err != nil {
			t.Fatalf("restore working dir: %v", err)
		}
	}
}
