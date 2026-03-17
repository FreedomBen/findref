package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"
)

// mcpOut is the real stdout, used exclusively for JSON-RPC responses.
// During MCP mode, os.Stdout is redirected to os.Stderr so that any
// stray fmt.Print calls from the search internals don't corrupt the
// JSON-RPC channel.
var mcpOut *os.File

// ---------------------------------------------------------------------------
// JSON-RPC 2.0 types
// ---------------------------------------------------------------------------

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ---------------------------------------------------------------------------
// MCP protocol types
// ---------------------------------------------------------------------------

type mcpServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type mcpCapabilities struct {
	Tools *struct{} `json:"tools,omitempty"`
}

type mcpInitializeResult struct {
	ProtocolVersion string          `json:"protocolVersion"`
	Capabilities    mcpCapabilities `json:"capabilities"`
	ServerInfo      mcpServerInfo   `json:"serverInfo"`
}

type mcpTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

type mcpToolsListResult struct {
	Tools []mcpTool `json:"tools"`
}

type mcpToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type mcpContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type mcpToolResult struct {
	Content []mcpContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// ---------------------------------------------------------------------------
// Search tool arguments & structured result
// ---------------------------------------------------------------------------

type searchArgs struct {
	Pattern        string   `json:"pattern"`
	Directory      string   `json:"directory"`
	FilePattern    string   `json:"file_pattern"`
	Exclude        []string `json:"exclude"`
	ExcludePattern []string `json:"exclude_pattern"`
	IgnoreCase     bool     `json:"ignore_case"`
	MatchCase      bool     `json:"match_case"`
	IncludeHidden  bool     `json:"include_hidden"`
	All            bool     `json:"all"`
	FilenameOnly   bool     `json:"filename_only"`
	MaxLineLength  *int     `json:"max_line_length"`
}

type searchResultEntry struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	Text       string `json:"text"`
	MatchStart int    `json:"match_start"`
	MatchEnd   int    `json:"match_end"`
}

// ---------------------------------------------------------------------------
// MCP server main loop
// ---------------------------------------------------------------------------

func mcpServe() {
	// Capture the real stdout for JSON-RPC output, then redirect
	// os.Stdout → os.Stderr so that any internal fmt.Print calls
	// (debug messages, match printing, errors) go to stderr instead
	// of corrupting the JSON-RPC channel.
	mcpOut = os.Stdout
	os.Stdout = os.Stderr

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var req jsonRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			fmt.Fprintf(os.Stderr, "findref-mcp: invalid JSON: %v\n", err)
			continue
		}

		// Notifications have no id and get no response.
		if req.ID == nil {
			continue
		}

		resp := jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
		}

		switch req.Method {
		case "initialize":
			resp.Result = handleInitialize()
		case "tools/list":
			resp.Result = handleToolsList()
		case "tools/call":
			result, err := handleToolsCall(req.Params)
			if err != nil {
				resp.Error = &jsonRPCError{Code: -32603, Message: err.Error()}
			} else {
				resp.Result = result
			}
		default:
			resp.Error = &jsonRPCError{
				Code:    -32601,
				Message: fmt.Sprintf("method not found: %s", req.Method),
			}
		}

		out, _ := json.Marshal(resp)
		fmt.Fprintf(mcpOut, "%s\n", out)
	}
}

// ---------------------------------------------------------------------------
// MCP method handlers
// ---------------------------------------------------------------------------

func handleInitialize() mcpInitializeResult {
	return mcpInitializeResult{
		ProtocolVersion: "2024-11-05",
		Capabilities:    mcpCapabilities{Tools: &struct{}{}},
		ServerInfo:      mcpServerInfo{Name: "findref", Version: Version},
	}
}

func handleToolsList() mcpToolsListResult {
	searchSchema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"pattern": {
				"type": "string",
				"description": "RE2 regular expression to match against lines in files. Uses smart-case by default: all-lowercase pattern is case-insensitive, any uppercase forces case-sensitive."
			},
			"directory": {
				"type": "string",
				"description": "Starting directory to search (default: current working directory)."
			},
			"file_pattern": {
				"type": "string",
				"description": "RE2 regex to filter which files to scan (matched against the file path). Example: '\\.go$' for Go files, '\\.(js|ts)$' for JavaScript/TypeScript."
			},
			"exclude": {
				"type": "array",
				"items": {"type": "string"},
				"description": "Directory or file names to exclude (exact basename match). These are added on top of the defaults."
			},
			"exclude_pattern": {
				"type": "array",
				"items": {"type": "string"},
				"description": "RE2 regex patterns; paths matching any pattern are excluded."
			},
			"ignore_case": {
				"type": "boolean",
				"description": "Force case-insensitive matching (overrides smart-case). Default false."
			},
			"match_case": {
				"type": "boolean",
				"description": "Force case-sensitive matching (overrides smart-case). Default false."
			},
			"include_hidden": {
				"type": "boolean",
				"description": "Include hidden files and directories (names starting with '.'). Default false."
			},
			"all": {
				"type": "boolean",
				"description": "Aggressive mode: implies ignore_case and include_hidden, disables default excludes. Default false."
			},
			"filename_only": {
				"type": "boolean",
				"description": "Return only unique, sorted filenames containing matches instead of individual match details. Default false."
			},
			"max_line_length": {
				"type": "integer",
				"description": "Maximum line length in characters before clipping (default 2000)."
			}
		},
		"required": ["pattern"]
	}`)

	defaultExcludesSchema := json.RawMessage(`{
		"type": "object",
		"properties": {},
		"additionalProperties": false
	}`)

	return mcpToolsListResult{
		Tools: []mcpTool{
			{
				Name:        "search",
				Description: "Search for text patterns in files using RE2 regular expressions. Recursively scans directories, automatically skipping binary files, VCS metadata, lock files, and common build artifacts by default.",
				InputSchema: searchSchema,
			},
			{
				Name:        "list_default_excludes",
				Description: "List the directories and files excluded from search by default (VCS dirs, lock files, build artifacts). Useful for understanding what is filtered before running a search.",
				InputSchema: defaultExcludesSchema,
			},
		},
	}
}

func handleToolsCall(params json.RawMessage) (*mcpToolResult, error) {
	var call mcpToolCallParams
	if err := json.Unmarshal(params, &call); err != nil {
		return nil, fmt.Errorf("invalid tool call params: %w", err)
	}

	switch call.Name {
	case "search":
		return handleSearch(call.Arguments)
	case "list_default_excludes":
		return handleListDefaultExcludes()
	default:
		return nil, fmt.Errorf("unknown tool: %s", call.Name)
	}
}

func handleListDefaultExcludes() (*mcpToolResult, error) {
	result, _ := json.Marshal(defaultExcludeDirs)
	return &mcpToolResult{
		Content: []mcpContent{{Type: "text", Text: string(result)}},
	}, nil
}

func handleSearch(argsJSON json.RawMessage) (*mcpToolResult, error) {
	var args searchArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return &mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: fmt.Sprintf("invalid arguments: %v", err)}},
			IsError: true,
		}, nil
	}

	if args.Pattern == "" {
		return &mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: "pattern is required"}},
			IsError: true,
		}, nil
	}

	// Reset global state for this search invocation.
	settings = NewSettings()
	statistics = NewStatistics()
	colors.ZeroColors()
	filenameOnlyFiles = make([]string, 0, 100)
	filesToScan = make([]FileToScan, 0, 100)

	// Apply arguments to settings.
	allEnabled := args.All
	settings.IncludeHidden = args.IncludeHidden || allEnabled
	settings.FilenameOnly = args.FilenameOnly
	settings.UseDefaultExcludes = !allEnabled

	if args.MaxLineLength != nil {
		settings.MaxLineLength = *args.MaxLineLength
	}
	if len(args.Exclude) > 0 {
		settings.AddExcludes(args.Exclude...)
	}
	if len(args.ExcludePattern) > 0 {
		if err := settings.AddExcludePatterns(args.ExcludePattern...); err != nil {
			return &mcpToolResult{
				Content: []mcpContent{{Type: "text", Text: fmt.Sprintf("invalid exclude_pattern: %v", err)}},
				IsError: true,
			}, nil
		}
	}

	ignoreCase := args.IgnoreCase || allEnabled
	matchRegex, err := getMatchRegex(ignoreCase, args.MatchCase, args.Pattern)
	if err != nil {
		return &mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: fmt.Sprintf("invalid pattern: %v", err)}},
			IsError: true,
		}, nil
	}
	settings.MatchRegex = matchRegex

	if args.FilePattern != "" {
		filenameRegex, err := regexp.Compile(args.FilePattern)
		if err != nil {
			return &mcpToolResult{
				Content: []mcpContent{{Type: "text", Text: fmt.Sprintf("invalid file_pattern: %v", err)}},
				IsError: true,
			}, nil
		}
		settings.FilenameRegex = filenameRegex
	}

	rootDir := "."
	if args.Directory != "" {
		rootDir = args.Directory
	}

	statistics.startTime = time.Now()

	// Walk the directory tree to collect eligible files.
	filepath.Walk(rootDir, processFile)

	// Fan out to worker goroutines.
	runtime.GOMAXPROCS(runtime.NumCPU())
	jobs := make(chan string, len(filesToScan))
	results := make(chan []Match, 100)

	numWorkers := runtime.NumCPU()
	for w := 0; w < numWorkers; w++ {
		go worker(w, jobs, results)
	}
	for _, val := range filesToScan {
		jobs <- val.Path
	}
	close(jobs)

	// Collect match results.
	var allMatches []searchResultEntry
	for r := 0; r < len(filesToScan); r++ {
		batch := <-results
		for _, m := range batch {
			if m.hasMatch() {
				allMatches = append(allMatches, searchResultEntry{
					File:       m.Path,
					Line:       m.LineNumber,
					Text:       string(m.Line),
					MatchStart: m.Match[0],
					MatchEnd:   m.Match[1],
				})
			}
		}
	}

	// Filename-only mode: return sorted unique filenames.
	if settings.FilenameOnly {
		filenames := uniq(filenameOnlyFiles)
		sort.Strings(filenames)
		resultJSON, _ := json.Marshal(filenames)
		return &mcpToolResult{
			Content: []mcpContent{{Type: "text", Text: string(resultJSON)}},
		}, nil
	}

	// Normal mode: return structured match data.
	if allMatches == nil {
		allMatches = []searchResultEntry{}
	}
	output := struct {
		Matches      []searchResultEntry `json:"matches"`
		TotalFiles   int                 `json:"total_files_scanned"`
		TotalLines   int                 `json:"total_lines_scanned"`
		TotalMatches int                 `json:"total_matches"`
	}{
		Matches:      allMatches,
		TotalFiles:   statistics.FileCount(),
		TotalLines:   statistics.LineCount(),
		TotalMatches: len(allMatches),
	}
	resultJSON, _ := json.Marshal(output)
	return &mcpToolResult{
		Content: []mcpContent{{Type: "text", Text: string(resultJSON)}},
	}, nil
}
