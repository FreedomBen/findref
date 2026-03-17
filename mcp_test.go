package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// handleInitialize
// ---------------------------------------------------------------------------

func TestMCPInitialize(t *testing.T) {
	result := handleInitialize()

	if result.ProtocolVersion != "2024-11-05" {
		t.Errorf("expected protocol version 2024-11-05, got %q", result.ProtocolVersion)
	}
	if result.ServerInfo.Name != "findref" {
		t.Errorf("expected server name findref, got %q", result.ServerInfo.Name)
	}
	if result.ServerInfo.Version != Version {
		t.Errorf("expected server version %q, got %q", Version, result.ServerInfo.Version)
	}
	if result.Capabilities.Tools == nil {
		t.Error("expected tools capability to be non-nil")
	}
}

// ---------------------------------------------------------------------------
// handleToolsList
// ---------------------------------------------------------------------------

func TestMCPToolsList(t *testing.T) {
	result := handleToolsList()

	if len(result.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(result.Tools))
	}

	names := map[string]bool{}
	for _, tool := range result.Tools {
		names[tool.Name] = true
		if tool.Description == "" {
			t.Errorf("tool %q has empty description", tool.Name)
		}
		// Verify inputSchema is valid JSON
		var schema map[string]interface{}
		if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
			t.Errorf("tool %q has invalid inputSchema: %v", tool.Name, err)
		}
	}

	if !names["search"] {
		t.Error("expected search tool")
	}
	if !names["list_default_excludes"] {
		t.Error("expected list_default_excludes tool")
	}
}

func TestMCPSearchToolSchema(t *testing.T) {
	result := handleToolsList()

	var searchTool mcpTool
	for _, tool := range result.Tools {
		if tool.Name == "search" {
			searchTool = tool
			break
		}
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(searchTool.InputSchema, &schema); err != nil {
		t.Fatalf("invalid schema: %v", err)
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties in schema")
	}

	expectedProps := []string{
		"pattern", "directory", "file_pattern", "exclude",
		"exclude_pattern", "ignore_case", "match_case",
		"include_hidden", "all", "filename_only", "max_line_length",
	}
	for _, prop := range expectedProps {
		if _, exists := props[prop]; !exists {
			t.Errorf("expected property %q in search schema", prop)
		}
	}

	required, ok := schema["required"].([]interface{})
	if !ok {
		t.Fatal("expected required array in schema")
	}
	if len(required) != 1 || required[0] != "pattern" {
		t.Errorf("expected required=[\"pattern\"], got %v", required)
	}
}

// ---------------------------------------------------------------------------
// handleToolsCall: dispatch
// ---------------------------------------------------------------------------

func TestMCPToolsCallUnknownTool(t *testing.T) {
	params, _ := json.Marshal(mcpToolCallParams{Name: "nonexistent", Arguments: json.RawMessage(`{}`)})
	_, err := handleToolsCall(params)
	if err == nil {
		t.Fatal("expected error for unknown tool")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Errorf("expected 'unknown tool' error, got %q", err.Error())
	}
}

func TestMCPToolsCallInvalidJSON(t *testing.T) {
	_, err := handleToolsCall(json.RawMessage(`{invalid`))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// ---------------------------------------------------------------------------
// handleListDefaultExcludes
// ---------------------------------------------------------------------------

func TestMCPListDefaultExcludes(t *testing.T) {
	result, err := handleListDefaultExcludes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success result")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	if result.Content[0].Type != "text" {
		t.Errorf("expected content type 'text', got %q", result.Content[0].Type)
	}

	var excludes []string
	if err := json.Unmarshal([]byte(result.Content[0].Text), &excludes); err != nil {
		t.Fatalf("content is not valid JSON array: %v", err)
	}
	if len(excludes) != len(defaultExcludeDirs) {
		t.Errorf("expected %d excludes, got %d", len(defaultExcludeDirs), len(excludes))
	}

	// Spot check a few entries
	excludeSet := map[string]bool{}
	for _, e := range excludes {
		excludeSet[e] = true
	}
	for _, expected := range []string{".git", "node_modules", "vendor", "go.sum"} {
		if !excludeSet[expected] {
			t.Errorf("expected %q in default excludes", expected)
		}
	}
}

// ---------------------------------------------------------------------------
// handleSearch: basic search
// ---------------------------------------------------------------------------

func TestMCPSearchBasic(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "main.go"), "package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n")
	mustWriteFile(t, filepath.Join(tmpDir, "lib.go"), "package main\n\nfunc helper() {}\n")

	args, _ := json.Marshal(searchArgs{
		Pattern:   "^func",
		Directory: tmpDir,
	})
	result, err := handleSearch(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("search returned error: %s", result.Content[0].Text)
	}

	var output struct {
		Matches      []searchResultEntry `json:"matches"`
		TotalFiles   int                 `json:"total_files_scanned"`
		TotalLines   int                 `json:"total_lines_scanned"`
		TotalMatches int                 `json:"total_matches"`
	}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &output); err != nil {
		t.Fatalf("invalid result JSON: %v", err)
	}

	if output.TotalMatches != 2 {
		t.Errorf("expected 2 matches, got %d", output.TotalMatches)
	}
	if output.TotalFiles < 2 {
		t.Errorf("expected at least 2 files scanned, got %d", output.TotalFiles)
	}
	if output.TotalLines < 5 {
		t.Errorf("expected at least 5 lines scanned, got %d", output.TotalLines)
	}

	// Verify match structure
	for _, m := range output.Matches {
		if m.File == "" {
			t.Error("match has empty file path")
		}
		if m.Line == 0 {
			t.Error("match has zero line number")
		}
		if m.Text == "" {
			t.Error("match has empty text")
		}
		if !strings.HasPrefix(m.Text, "func") {
			t.Errorf("expected match to start with 'func', got %q", m.Text)
		}
		if m.MatchEnd <= m.MatchStart {
			t.Errorf("expected match_end > match_start, got %d <= %d", m.MatchEnd, m.MatchStart)
		}
	}
}

// ---------------------------------------------------------------------------
// handleSearch: file_pattern filter
// ---------------------------------------------------------------------------

func TestMCPSearchFilePattern(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "main.go"), "TODO in go\n")
	mustWriteFile(t, filepath.Join(tmpDir, "main.py"), "TODO in py\n")
	mustWriteFile(t, filepath.Join(tmpDir, "main.js"), "TODO in js\n")

	args, _ := json.Marshal(searchArgs{
		Pattern:     "TODO",
		Directory:   tmpDir,
		FilePattern: `\.go$`,
	})
	result, err := handleSearch(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var output struct {
		Matches      []searchResultEntry `json:"matches"`
		TotalMatches int                 `json:"total_matches"`
	}
	json.Unmarshal([]byte(result.Content[0].Text), &output)

	if output.TotalMatches != 1 {
		t.Errorf("expected 1 match (only .go file), got %d", output.TotalMatches)
	}
	if len(output.Matches) > 0 && !strings.HasSuffix(output.Matches[0].File, ".go") {
		t.Errorf("expected match in .go file, got %q", output.Matches[0].File)
	}
}

// ---------------------------------------------------------------------------
// handleSearch: exclude
// ---------------------------------------------------------------------------

func TestMCPSearchExclude(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "src", "main.go"), "TODO src\n")
	mustWriteFile(t, filepath.Join(tmpDir, "skipme", "lib.go"), "TODO skip\n")

	args, _ := json.Marshal(searchArgs{
		Pattern:   "TODO",
		Directory: tmpDir,
		Exclude:   []string{"skipme"},
	})
	result, err := handleSearch(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var output struct {
		Matches []searchResultEntry `json:"matches"`
	}
	json.Unmarshal([]byte(result.Content[0].Text), &output)

	for _, m := range output.Matches {
		if strings.Contains(m.File, "skipme") {
			t.Errorf("expected skipme to be excluded, found match in %q", m.File)
		}
	}
}

// ---------------------------------------------------------------------------
// handleSearch: exclude_pattern
// ---------------------------------------------------------------------------

func TestMCPSearchExcludePattern(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "app.go"), "TODO app\n")
	mustWriteFile(t, filepath.Join(tmpDir, "app_test.go"), "TODO test\n")

	args, _ := json.Marshal(searchArgs{
		Pattern:        "TODO",
		Directory:      tmpDir,
		ExcludePattern: []string{`_test\.go$`},
	})
	result, err := handleSearch(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var output struct {
		Matches      []searchResultEntry `json:"matches"`
		TotalMatches int                 `json:"total_matches"`
	}
	json.Unmarshal([]byte(result.Content[0].Text), &output)

	if output.TotalMatches != 1 {
		t.Errorf("expected 1 match (test file excluded), got %d", output.TotalMatches)
	}
	for _, m := range output.Matches {
		if strings.Contains(m.File, "_test.go") {
			t.Errorf("expected test file to be excluded, found match in %q", m.File)
		}
	}
}

// ---------------------------------------------------------------------------
// handleSearch: filename_only mode
// ---------------------------------------------------------------------------

func TestMCPSearchFilenameOnly(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "a.txt"), "TODO first\nTODO second\n")
	mustWriteFile(t, filepath.Join(tmpDir, "b.txt"), "TODO here too\n")
	mustWriteFile(t, filepath.Join(tmpDir, "c.txt"), "no match\n")

	args, _ := json.Marshal(searchArgs{
		Pattern:      "TODO",
		Directory:    tmpDir,
		FilenameOnly: true,
	})
	result, err := handleSearch(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var filenames []string
	if err := json.Unmarshal([]byte(result.Content[0].Text), &filenames); err != nil {
		t.Fatalf("expected JSON array of filenames: %v", err)
	}
	if len(filenames) != 2 {
		t.Errorf("expected 2 filenames, got %d: %v", len(filenames), filenames)
	}

	// Should be sorted
	for i := 1; i < len(filenames); i++ {
		if filenames[i] < filenames[i-1] {
			t.Error("expected filenames to be sorted")
			break
		}
	}

	// c.txt should not be present
	for _, f := range filenames {
		if strings.HasSuffix(f, "c.txt") {
			t.Error("c.txt should not be in results (no match)")
		}
	}
}

// ---------------------------------------------------------------------------
// handleSearch: case sensitivity
// ---------------------------------------------------------------------------

func TestMCPSearchIgnoreCase(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "test.txt"), "Hello\nhello\nHELLO\n")

	args, _ := json.Marshal(searchArgs{
		Pattern:    "Hello",
		Directory:  tmpDir,
		IgnoreCase: true,
	})
	result, err := handleSearch(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var output struct {
		TotalMatches int `json:"total_matches"`
	}
	json.Unmarshal([]byte(result.Content[0].Text), &output)

	if output.TotalMatches != 3 {
		t.Errorf("expected 3 matches with ignore_case, got %d", output.TotalMatches)
	}
}

func TestMCPSearchMatchCase(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "test.txt"), "Hello\nhello\nHELLO\n")

	args, _ := json.Marshal(searchArgs{
		Pattern:   "hello",
		Directory: tmpDir,
		MatchCase: true,
	})
	result, err := handleSearch(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var output struct {
		TotalMatches int `json:"total_matches"`
	}
	json.Unmarshal([]byte(result.Content[0].Text), &output)

	if output.TotalMatches != 1 {
		t.Errorf("expected 1 match with match_case, got %d", output.TotalMatches)
	}
}

func TestMCPSearchSmartCase(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "test.txt"), "Hello\nhello\nHELLO\n")

	// All lowercase → smart-case → case-insensitive
	args, _ := json.Marshal(searchArgs{
		Pattern:   "hello",
		Directory: tmpDir,
	})
	result, err := handleSearch(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var output struct {
		TotalMatches int `json:"total_matches"`
	}
	json.Unmarshal([]byte(result.Content[0].Text), &output)

	if output.TotalMatches != 3 {
		t.Errorf("expected 3 matches with smart-case lowercase, got %d", output.TotalMatches)
	}

	// Uppercase in pattern → case-sensitive
	args2, _ := json.Marshal(searchArgs{
		Pattern:   "Hello",
		Directory: tmpDir,
	})
	result2, _ := handleSearch(args2)
	var output2 struct {
		TotalMatches int `json:"total_matches"`
	}
	json.Unmarshal([]byte(result2.Content[0].Text), &output2)

	if output2.TotalMatches != 1 {
		t.Errorf("expected 1 match with smart-case uppercase, got %d", output2.TotalMatches)
	}
}

// ---------------------------------------------------------------------------
// handleSearch: include_hidden
// ---------------------------------------------------------------------------

func TestMCPSearchIncludeHidden(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, ".env"), "TODO secret\n")
	mustWriteFile(t, filepath.Join(tmpDir, "main.go"), "TODO main\n")

	// Without include_hidden
	args, _ := json.Marshal(searchArgs{
		Pattern:   "TODO",
		Directory: tmpDir,
	})
	result, _ := handleSearch(args)
	var output struct {
		TotalMatches int `json:"total_matches"`
	}
	json.Unmarshal([]byte(result.Content[0].Text), &output)
	if output.TotalMatches != 1 {
		t.Errorf("expected 1 match without hidden, got %d", output.TotalMatches)
	}

	// With include_hidden
	args2, _ := json.Marshal(searchArgs{
		Pattern:       "TODO",
		Directory:     tmpDir,
		IncludeHidden: true,
	})
	result2, _ := handleSearch(args2)
	var output2 struct {
		TotalMatches int `json:"total_matches"`
	}
	json.Unmarshal([]byte(result2.Content[0].Text), &output2)
	if output2.TotalMatches != 2 {
		t.Errorf("expected 2 matches with hidden, got %d", output2.TotalMatches)
	}
}

// ---------------------------------------------------------------------------
// handleSearch: all mode
// ---------------------------------------------------------------------------

func TestMCPSearchAllMode(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, ".hidden.txt"), "TODO hidden\n")
	mustWriteFile(t, filepath.Join(tmpDir, "vendor", "lib.go"), "TODO vendor\n")
	mustWriteFile(t, filepath.Join(tmpDir, "main.go"), "TODO main\n")

	args, _ := json.Marshal(searchArgs{
		Pattern:   "TODO",
		Directory: tmpDir,
		All:       true,
	})
	result, _ := handleSearch(args)
	var output struct {
		TotalMatches int `json:"total_matches"`
	}
	json.Unmarshal([]byte(result.Content[0].Text), &output)

	if output.TotalMatches != 3 {
		t.Errorf("expected 3 matches with all mode, got %d", output.TotalMatches)
	}
}

// ---------------------------------------------------------------------------
// handleSearch: no matches
// ---------------------------------------------------------------------------

func TestMCPSearchNoMatches(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "test.txt"), "nothing relevant\n")

	args, _ := json.Marshal(searchArgs{
		Pattern:   "ZZZZNOTFOUND",
		Directory: tmpDir,
	})
	result, err := handleSearch(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected success result even with no matches")
	}

	var output struct {
		Matches      []searchResultEntry `json:"matches"`
		TotalMatches int                 `json:"total_matches"`
	}
	json.Unmarshal([]byte(result.Content[0].Text), &output)

	if output.TotalMatches != 0 {
		t.Errorf("expected 0 matches, got %d", output.TotalMatches)
	}
	if len(output.Matches) != 0 {
		t.Errorf("expected empty matches array, got %d items", len(output.Matches))
	}
}

// ---------------------------------------------------------------------------
// handleSearch: error cases
// ---------------------------------------------------------------------------

func TestMCPSearchEmptyPattern(t *testing.T) {
	resetTestState(t)
	args, _ := json.Marshal(searchArgs{Pattern: ""})
	result, err := handleSearch(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for empty pattern")
	}
	if !strings.Contains(result.Content[0].Text, "pattern is required") {
		t.Errorf("expected 'pattern is required', got %q", result.Content[0].Text)
	}
}

func TestMCPSearchInvalidPattern(t *testing.T) {
	resetTestState(t)
	args, _ := json.Marshal(searchArgs{Pattern: "["})
	result, err := handleSearch(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for invalid regex")
	}
	if !strings.Contains(result.Content[0].Text, "invalid pattern") {
		t.Errorf("expected 'invalid pattern' error, got %q", result.Content[0].Text)
	}
}

func TestMCPSearchInvalidFilePattern(t *testing.T) {
	resetTestState(t)
	args, _ := json.Marshal(searchArgs{Pattern: "TODO", FilePattern: "["})
	result, err := handleSearch(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for invalid file_pattern")
	}
	if !strings.Contains(result.Content[0].Text, "invalid file_pattern") {
		t.Errorf("expected 'invalid file_pattern' error, got %q", result.Content[0].Text)
	}
}

func TestMCPSearchInvalidExcludePattern(t *testing.T) {
	resetTestState(t)
	args, _ := json.Marshal(searchArgs{Pattern: "TODO", ExcludePattern: []string{"("}})
	result, err := handleSearch(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for invalid exclude_pattern")
	}
	if !strings.Contains(result.Content[0].Text, "invalid exclude_pattern") {
		t.Errorf("expected 'invalid exclude_pattern' error, got %q", result.Content[0].Text)
	}
}

func TestMCPSearchInvalidArguments(t *testing.T) {
	result, err := handleSearch(json.RawMessage(`{invalid`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for invalid JSON")
	}
}

// ---------------------------------------------------------------------------
// handleSearch: binary files skipped
// ---------------------------------------------------------------------------

func TestMCPSearchBinarySkipped(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "text.txt"), "TODO text\n")
	mustWriteFile(t, filepath.Join(tmpDir, "binary.bin"), "TODO\x00binary\n")

	args, _ := json.Marshal(searchArgs{
		Pattern:   "TODO",
		Directory: tmpDir,
	})
	result, _ := handleSearch(args)
	var output struct {
		Matches []searchResultEntry `json:"matches"`
	}
	json.Unmarshal([]byte(result.Content[0].Text), &output)

	for _, m := range output.Matches {
		if strings.Contains(m.File, "binary") {
			t.Error("expected binary file to be skipped")
		}
	}
}

// ---------------------------------------------------------------------------
// handleSearch: default excludes active
// ---------------------------------------------------------------------------

func TestMCPSearchDefaultExcludes(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "src", "main.go"), "TODO main\n")
	mustWriteFile(t, filepath.Join(tmpDir, "node_modules", "pkg.js"), "TODO node\n")
	mustWriteFile(t, filepath.Join(tmpDir, "vendor", "lib.go"), "TODO vendor\n")

	args, _ := json.Marshal(searchArgs{
		Pattern:   "TODO",
		Directory: tmpDir,
	})
	result, _ := handleSearch(args)
	var output struct {
		Matches []searchResultEntry `json:"matches"`
	}
	json.Unmarshal([]byte(result.Content[0].Text), &output)

	for _, m := range output.Matches {
		if strings.Contains(m.File, "node_modules") {
			t.Error("expected node_modules to be excluded by default")
		}
		if strings.Contains(m.File, "vendor") {
			t.Error("expected vendor to be excluded by default")
		}
	}
	if len(output.Matches) != 1 {
		t.Errorf("expected 1 match (only src/main.go), got %d", len(output.Matches))
	}
}

// ---------------------------------------------------------------------------
// handleSearch: max_line_length
// ---------------------------------------------------------------------------

func TestMCPSearchMaxLineLength(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "short.txt"), "TODO short\n")

	maxLen := 50
	args, _ := json.Marshal(searchArgs{
		Pattern:       "TODO",
		Directory:     tmpDir,
		MaxLineLength: &maxLen,
	})
	result, _ := handleSearch(args)
	var output struct {
		TotalMatches int `json:"total_matches"`
	}
	json.Unmarshal([]byte(result.Content[0].Text), &output)

	if output.TotalMatches != 1 {
		t.Errorf("expected 1 match, got %d", output.TotalMatches)
	}
}

// ---------------------------------------------------------------------------
// handleSearch: multiple sequential searches reset state
// ---------------------------------------------------------------------------

func TestMCPSearchStateReset(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "test.txt"), "alpha\nbeta\n")

	// First search
	args1, _ := json.Marshal(searchArgs{Pattern: "alpha", Directory: tmpDir})
	result1, _ := handleSearch(args1)
	var output1 struct {
		TotalMatches int `json:"total_matches"`
	}
	json.Unmarshal([]byte(result1.Content[0].Text), &output1)
	if output1.TotalMatches != 1 {
		t.Errorf("first search: expected 1 match, got %d", output1.TotalMatches)
	}

	// Second search — should not carry over state from first
	args2, _ := json.Marshal(searchArgs{Pattern: "beta", Directory: tmpDir})
	result2, _ := handleSearch(args2)
	var output2 struct {
		TotalMatches int `json:"total_matches"`
	}
	json.Unmarshal([]byte(result2.Content[0].Text), &output2)
	if output2.TotalMatches != 1 {
		t.Errorf("second search: expected 1 match, got %d", output2.TotalMatches)
	}
}

// ---------------------------------------------------------------------------
// handleSearch: nested directories
// ---------------------------------------------------------------------------

func TestMCPSearchNestedDirs(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "a", "b", "c", "deep.txt"), "TODO deep\n")
	mustWriteFile(t, filepath.Join(tmpDir, "top.txt"), "TODO top\n")

	args, _ := json.Marshal(searchArgs{
		Pattern:      "TODO",
		Directory:    tmpDir,
		FilenameOnly: true,
	})
	result, _ := handleSearch(args)

	var filenames []string
	json.Unmarshal([]byte(result.Content[0].Text), &filenames)

	if len(filenames) != 2 {
		t.Errorf("expected 2 filenames, got %d", len(filenames))
	}
}

// ---------------------------------------------------------------------------
// handleSearch: empty directory
// ---------------------------------------------------------------------------

func TestMCPSearchEmptyDir(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()

	args, _ := json.Marshal(searchArgs{
		Pattern:   "TODO",
		Directory: tmpDir,
	})
	result, _ := handleSearch(args)

	var output struct {
		Matches      []searchResultEntry `json:"matches"`
		TotalMatches int                 `json:"total_matches"`
		TotalFiles   int                 `json:"total_files_scanned"`
	}
	json.Unmarshal([]byte(result.Content[0].Text), &output)

	if output.TotalMatches != 0 {
		t.Errorf("expected 0 matches in empty dir, got %d", output.TotalMatches)
	}
	if output.TotalFiles != 0 {
		t.Errorf("expected 0 files scanned in empty dir, got %d", output.TotalFiles)
	}
	if len(output.Matches) != 0 {
		t.Errorf("expected empty matches array, got %d items", len(output.Matches))
	}
}

// ---------------------------------------------------------------------------
// mcpServe: full JSON-RPC round-trip
// ---------------------------------------------------------------------------

func TestMCPServeRoundTrip(t *testing.T) {
	resetTestState(t)
	tmpDir := t.TempDir()
	mustWriteFile(t, filepath.Join(tmpDir, "test.txt"), "TODO find me\n")

	// Build a series of JSON-RPC messages
	searchArgs, _ := json.Marshal(map[string]interface{}{
		"name":      "search",
		"arguments": map[string]interface{}{"pattern": "TODO", "directory": tmpDir},
	})
	messages := []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":` + string(searchArgs) + `}`,
		`{"jsonrpc":"2.0","id":4,"method":"unknown/method","params":{}}`,
	}
	input := strings.Join(messages, "\n") + "\n"

	// Set up stdin/stdout pipes
	oldStdin := os.Stdin
	oldStdout := os.Stdout

	stdinR, stdinW, _ := os.Pipe()
	stdoutR, stdoutW, _ := os.Pipe()

	os.Stdin = stdinR
	// mcpServe will set mcpOut = os.Stdout, then redirect os.Stdout to os.Stderr
	// So we need to set os.Stdout to our capture pipe before calling mcpServe
	os.Stdout = stdoutW

	// Write input and close
	go func() {
		stdinW.WriteString(input)
		stdinW.Close()
	}()

	// Read output (ReadAll blocks until stdoutW is closed)
	outputCh := make(chan string)
	go func() {
		data, _ := io.ReadAll(stdoutR)
		outputCh <- string(data)
	}()

	mcpServe()

	stdoutW.Close()
	output := <-outputCh
	stdoutR.Close()

	os.Stdin = oldStdin
	os.Stdout = oldStdout

	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have 4 responses (notification has no response)
	if len(lines) != 4 {
		t.Fatalf("expected 4 response lines, got %d: %v", len(lines), lines)
	}

	// Verify initialize response
	var resp1 jsonRPCResponse
	json.Unmarshal([]byte(lines[0]), &resp1)
	if resp1.Error != nil {
		t.Errorf("initialize should not error: %v", resp1.Error)
	}

	// Verify tools/list response
	var resp2 jsonRPCResponse
	json.Unmarshal([]byte(lines[1]), &resp2)
	if resp2.Error != nil {
		t.Errorf("tools/list should not error: %v", resp2.Error)
	}

	// Verify search response
	var resp3 jsonRPCResponse
	json.Unmarshal([]byte(lines[2]), &resp3)
	if resp3.Error != nil {
		t.Errorf("tools/call should not error: %v", resp3.Error)
	}

	// Verify unknown method returns error
	var resp4 jsonRPCResponse
	json.Unmarshal([]byte(lines[3]), &resp4)
	if resp4.Error == nil {
		t.Error("expected error for unknown method")
	}
}
