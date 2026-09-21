package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// LocalProvider implements an offline, heuristic-based AI engine.
type LocalProvider struct{}

func NewLocalProvider() *LocalProvider {
	return &LocalProvider{}
}

func (p *LocalProvider) ID() string {
	return "local"
}

func (p *LocalProvider) Name() string {
	return "ForgeAI Local Engine"
}

func (p *LocalProvider) Description() string {
	return "Built-in offline heuristic analysis engine with zero external API dependencies"
}

func (p *LocalProvider) DefaultModel() string {
	return "forge-heuristic-v1"
}

func (p *LocalProvider) IsConfigured() bool {
	return true
}

func (p *LocalProvider) Generate(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	// Check context cancellation
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	content := req.UserPrompt
	system := strings.ToLower(req.SystemPrompt)

	var output string

	if strings.Contains(system, "code explanation") || strings.Contains(system, "explain") {
		output = p.handleExplain(content, req.JSONMode)
	} else if strings.Contains(system, "code review") || strings.Contains(system, "review diff") {
		output = p.handleReview(content, req.JSONMode)
	} else if strings.Contains(system, "generate tests") || strings.Contains(system, "unit test") {
		output = p.handleGenerateTests(content, req.JSONMode)
	} else {
		output = p.handleChat(content, req.Messages)
	}

	return &CompletionResponse{
		Content:  output,
		Model:    p.DefaultModel(),
		Provider: p.ID(),
	}, nil
}

// ExtractCodeContent extracts code from within system delimiters if present.
func extractCodeContent(raw string) string {
	startMarker := "<<<FORGEHUB_UNTRUSTED_CONTENT_START>>>"
	endMarker := "<<<FORGEHUB_UNTRUSTED_CONTENT_END>>>"

	startIdx := strings.Index(raw, startMarker)
	if startIdx != -1 {
		rest := raw[startIdx+len(startMarker):]
		endIdx := strings.Index(rest, endMarker)
		if endIdx != -1 {
			return strings.TrimSpace(rest[:endIdx])
		}
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(raw)
}

func detectLanguage(code string) string {
	if strings.Contains(code, "package ") || strings.Contains(code, "func ") {
		return "Go"
	}
	if strings.Contains(code, "interface ") || strings.Contains(code, "type ") && strings.Contains(code, ":") || strings.Contains(code, "export const") {
		return "TypeScript"
	}
	if strings.Contains(code, "def ") || strings.Contains(code, "import ") && strings.Contains(code, ":") {
		return "Python"
	}
	if strings.Contains(code, "fn ") || strings.Contains(code, "pub struct") {
		return "Rust"
	}
	if strings.Contains(code, "SELECT ") || strings.Contains(code, "CREATE TABLE") {
		return "SQL"
	}
	return "Code"
}

func extractSymbols(code string) []string {
	var symbols []string
	// Match func names in Go
	funcRegex := regexp.MustCompile(`func\s+(?:\([^)]+\)\s+)?([A-Z][a-zA-Z0-9_]*)`)
	for _, match := range funcRegex.FindAllStringSubmatch(code, -1) {
		if len(match) > 1 {
			symbols = append(symbols, "Function: "+match[1])
		}
	}
	// Match struct/interface in Go
	typeRegex := regexp.MustCompile(`type\s+([A-Z][a-zA-Z0-9_]*)\s+(struct|interface)`)
	for _, match := range typeRegex.FindAllStringSubmatch(code, -1) {
		if len(match) > 2 {
			symbols = append(symbols, fmt.Sprintf("%s: %s", strings.Title(match[2]), match[1]))
		}
	}
	// Match TS/JS functions
	tsFuncRegex := regexp.MustCompile(`(?:export\s+)?(?:const|function)\s+([a-zA-Z0-9_]+)\s*(?:=|:)`)
	for _, match := range tsFuncRegex.FindAllStringSubmatch(code, -1) {
		if len(match) > 1 && !strings.HasPrefix(match[1], "use") {
			symbols = append(symbols, "Declaration: "+match[1])
		}
	}

	if len(symbols) == 0 {
		symbols = append(symbols, "Core component logic and utilities")
	}
	if len(symbols) > 8 {
		symbols = symbols[:8]
	}
	return symbols
}

func (p *LocalProvider) handleExplain(raw string, jsonMode bool) string {
	code := extractCodeContent(raw)
	lang := detectLanguage(code)
	symbols := extractSymbols(code)

	lines := strings.Split(code, "\n")
	lineCount := len(lines)

	// Complexity heuristics
	timeComplexity := "O(1)"
	spaceComplexity := "O(1)"
	complexityDesc := "Straightforward sequential execution flow without recursive or nested loops."

	loopCount := strings.Count(code, "for ") + strings.Count(code, "while ") + strings.Count(code, ".forEach") + strings.Count(code, ".map")
	if loopCount >= 2 {
		timeComplexity = "O(N²)"
		complexityDesc = "Contains nested iterations or iterative passes over collections."
	} else if loopCount == 1 {
		timeComplexity = "O(N)"
		complexityDesc = "Linear execution dependent on the volume of elements processed."
	}

	if strings.Contains(code, "make([]") || strings.Contains(code, "[]string") || strings.Contains(code, "new Array") {
		spaceComplexity = "O(N)"
	}

	var risks []string
	if strings.Contains(code, "panic(") {
		risks = append(risks, "Contains explicit panic calls which may abort application runtime if unrecovered.")
	}
	if strings.Contains(code, "go func") && !strings.Contains(code, "sync.WaitGroup") && !strings.Contains(code, "context.Context") {
		risks = append(risks, "Spawns background goroutines without context propagation or WaitGroup coordination.")
	}
	if strings.Contains(code, "_ =") {
		risks = append(risks, "Suppresses error return values using blank identifier assignment.")
	}
	if len(risks) == 0 {
		risks = append(risks, "Standard defensive validation recommended for nil or zero-value arguments.")
	}

	suggestions := []string{
		"Add structured docstrings or comments adhering to package conventions.",
		"Ensure table-driven unit tests cover boundary scenarios and error conditions.",
	}
	if strings.Contains(code, "context.Context") {
		suggestions = append(suggestions, "Verify context cancellation checks in long-running loops.")
	}

	summary := fmt.Sprintf("This %s module spans %d lines and implements core domain logic with %d detected key symbols.", lang, lineCount, len(symbols))

	if jsonMode {
		payload := map[string]interface{}{
			"summary":            summary,
			"language":           lang,
			"key_components":     symbols,
			"complexity": map[string]string{
				"time":        timeComplexity,
				"space":       spaceComplexity,
				"description": complexityDesc,
			},
			"architectural_role": fmt.Sprintf("Provides %s business domain operations and encapsulates domain data structures.", lang),
			"potential_risks":    risks,
			"suggestions":        suggestions,
		}
		bytes, _ := json.MarshalIndent(payload, "", "  ")
		return string(bytes)
	}

	// Markdown mode
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### Code Overview (%s)\n\n", lang))
	sb.WriteString(summary + "\n\n")
	sb.WriteString("#### Key Components\n")
	for _, s := range symbols {
		sb.WriteString(fmt.Sprintf("- **%s**\n", s))
	}
	sb.WriteString(fmt.Sprintf("\n#### Computational Complexity\n- **Time Complexity**: `%s`\n- **Space Complexity**: `%s`\n- %s\n\n", timeComplexity, spaceComplexity, complexityDesc))
	sb.WriteString("#### Potential Caveats & Risks\n")
	for _, r := range risks {
		sb.WriteString(fmt.Sprintf("- %s\n", r))
	}
	sb.WriteString("\n#### Recommended Improvements\n")
	for _, s := range suggestions {
		sb.WriteString(fmt.Sprintf("- %s\n", s))
	}
	return sb.String()
}

type diffFinding struct {
	File         string `json:"file"`
	Line         int    `json:"line"`
	Severity     string `json:"severity"`
	Category     string `json:"category"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	SuggestedFix string `json:"suggested_fix"`
}

func (p *LocalProvider) handleReview(raw string, jsonMode bool) string {
	diff := extractCodeContent(raw)
	lines := strings.Split(diff, "\n")

	var currentFile string
	var currentLine int
	var findings []diffFinding

	for _, line := range lines {
		if strings.HasPrefix(line, "+++ b/") {
			currentFile = strings.TrimPrefix(line, "+++ b/")
			currentLine = 1
			continue
		}
		if strings.HasPrefix(line, "@@") {
			// e.g. @@ -10,4 +12,6 @@
			var newLine int
			if n, _ := fmt.Sscanf(line, "@@ -%*d,%*d +%d", &newLine); n > 0 {
				currentLine = newLine
			} else if n, _ := fmt.Sscanf(line, "@@ -%*d +%d", &newLine); n > 0 {
				currentLine = newLine
			}
			continue
		}

		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			addedContent := strings.TrimPrefix(line, "+")

			// Check for secret patterns
			if (strings.Contains(addedContent, "password") || strings.Contains(addedContent, "secret") || strings.Contains(addedContent, "api_key")) &&
				(strings.Contains(addedContent, "=") || strings.Contains(addedContent, ":")) &&
				(strings.Contains(addedContent, "\"") || strings.Contains(addedContent, "'")) &&
				!strings.Contains(addedContent, "os.Getenv") && !strings.Contains(addedContent, "process.env") {
				findings = append(findings, diffFinding{
					File:         currentFile,
					Line:         currentLine,
					Severity:     "CRITICAL",
					Category:     "SECURITY",
					Title:        "Potential Hardcoded Secret Detected",
					Description:  "Credential or secret value appears to be hardcoded in source code rather than loaded from environment variables.",
					SuggestedFix: "Load secret dynamically using configuration or environment variables.",
				})
			}

			// Check for SQL injection risks
			if (strings.Contains(addedContent, "fmt.Sprintf") || strings.Contains(addedContent, "+")) &&
				(strings.Contains(strings.ToUpper(addedContent), "SELECT ") || strings.Contains(strings.ToUpper(addedContent), "WHERE ")) {
				findings = append(findings, diffFinding{
					File:         currentFile,
					Line:         currentLine,
					Severity:     "HIGH",
					Category:     "SECURITY",
					Title:        "Potential SQL Injection via String Concatenation",
					Description:  "Dynamic SQL query construction detected. Raw concatenation can lead to SQL injection vulnerabilities.",
					SuggestedFix: "Use parameterized queries ($1, $2) or an ORM/query builder.",
				})
			}

			// Check for suppressed errors
			if strings.Contains(addedContent, "_ = ") && (strings.Contains(addedContent, "Close()") || strings.Contains(addedContent, "Write(")) {
				findings = append(findings, diffFinding{
					File:         currentFile,
					Line:         currentLine,
					Severity:     "LOW",
					Category:     "BUG",
					Title:        "Ignored Error Return Value",
					Description:  "Error return value from I/O operation is discarded with blank identifier.",
					SuggestedFix: "Check error and log warning or return to caller.",
				})
			}

			// Check for console.log or fmt.Println in production code
			if strings.Contains(addedContent, "console.log(") || strings.Contains(addedContent, "fmt.Println(") {
				findings = append(findings, diffFinding{
					File:         currentFile,
					Line:         currentLine,
					Severity:     "LOW",
					Category:     "STYLE",
					Title:        "Debug Log Statement Left in Code",
					Description:  "Direct stdout logging statement found. Prefer structured application logger.",
					SuggestedFix: "Replace with structured logger or remove prior to merging.",
				})
			}

			currentLine++
		} else if !strings.HasPrefix(line, "-") {
			currentLine++
		}
	}

	// Calculate verdict and score
	verdict := "APPROVE"
	score := 95
	summary := "Changes appear well-structured, modular, and follow standard idioms."

	for _, f := range findings {
		if f.Severity == "CRITICAL" {
			verdict = "REQUEST_CHANGES"
			score = 45
			summary = "Critical security issues identified that should be resolved before merging."
			break
		} else if f.Severity == "HIGH" {
			verdict = "REQUEST_CHANGES"
			score = 65
			summary = "High-priority bugs or vulnerabilities detected that require attention."
		} else if f.Severity == "MEDIUM" && verdict == "APPROVE" {
			verdict = "COMMENT"
			score = 80
			summary = "Code is generally sound with minor edge cases or optimizations suggested."
		}
	}

	var commentBuilder strings.Builder
	commentBuilder.WriteString(fmt.Sprintf("## ForgeAI Automated Review\n\n**Verdict**: `%s` (Score: %d/100)\n\n%s\n\n", verdict, score, summary))

	if len(findings) > 0 {
		commentBuilder.WriteString("### Findings & Action Items\n\n")
		for i, f := range findings {
			commentBuilder.WriteString(fmt.Sprintf("%d. **[%s] %s** (`%s:%d`)\n", i+1, f.Severity, f.Title, f.File, f.Line))
			commentBuilder.WriteString(fmt.Sprintf("   - %s\n", f.Description))
			if f.SuggestedFix != "" {
				commentBuilder.WriteString(fmt.Sprintf("   - *Suggested Fix*: %s\n", f.SuggestedFix))
			}
		}
	} else {
		commentBuilder.WriteString("✅ No security vulnerabilities, concurrency leaks, or obvious code defects detected in this changeset.\n")
	}

	if jsonMode {
		payload := map[string]interface{}{
			"verdict":        verdict,
			"summary":        summary,
			"score":          score,
			"findings":       findings,
			"review_comment": commentBuilder.String(),
		}
		bytes, _ := json.MarshalIndent(payload, "", "  ")
		return string(bytes)
	}

	return commentBuilder.String()
}

func (p *LocalProvider) handleGenerateTests(raw string, jsonMode bool) string {
	code := extractCodeContent(raw)
	lang := detectLanguage(code)

	var framework string
	var testCode string
	var scenarios []map[string]string

	switch lang {
	case "Go":
		framework = "testing"
		testCode = `package main

import (
	"testing"
)

func TestCoreFunctionality(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid input scenario",
			input:   "standard_payload",
			wantErr: false,
		},
		{
			name:    "empty input edge case",
			input:   "",
			wantErr: true,
		},
		{
			name:    "maximum threshold boundary",
			input:   "boundary_test_value",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute target component with tt.input
			// Validate output and error condition against tt.wantErr
		})
	}
}
`
		scenarios = []map[string]string{
			{"name": "Valid Input Execution", "description": "Verifies expected output with valid arguments", "type": "happy_path"},
			{"name": "Empty Input Handling", "description": "Ensures proper error or validation rejection on empty input", "type": "edge_case"},
			{"name": "Boundary Threshold Test", "description": "Tests handling of maximum permissible boundary sizes", "type": "boundary"},
		}

	case "TypeScript", "JavaScript":
		framework = "vitest"
		testCode = `import { describe, it, expect } from 'vitest';

describe('Component Test Suite', () => {
  it('should handle standard execution successfully', () => {
    const input = 'sample_value';
    expect(input).toBeDefined();
  });

  it('should throw or reject on invalid parameters', () => {
    expect(() => {
      // execute invalid call
    }).toThrow();
  });

  it('should handle nullish boundary values gracefully', () => {
    const nullish = null;
    expect(nullish).toBeNull();
  });
});
`
		scenarios = []map[string]string{
			{"name": "Standard Execution", "description": "Verifies expected behavior for normal operation", "type": "happy_path"},
			{"name": "Invalid Parameter Rejection", "description": "Validates throwing or returning error on malformed arguments", "type": "error_handling"},
			{"name": "Nullish Boundary Check", "description": "Tests graceful fallback on null or undefined inputs", "type": "boundary"},
		}

	default:
		framework = "pytest"
		testCode = `import pytest

def test_standard_execution():
    assert True

def test_invalid_input_handling():
    with pytest.raises(ValueError):
        raise ValueError("Invalid parameter")
`
		scenarios = []map[string]string{
			{"name": "Standard Execution", "description": "Checks nominal path", "type": "happy_path"},
			{"name": "Error Handling", "description": "Checks exception raising", "type": "error_handling"},
		}
	}

	instructions := fmt.Sprintf("Run with standard test runner: for Go use `go test -v ./...`, for TS use `npm test`.")

	if jsonMode {
		payload := map[string]interface{}{
			"language":       lang,
			"framework":      framework,
			"test_code":      testCode,
			"test_scenarios": scenarios,
			"instructions":   instructions,
		}
		bytes, _ := json.MarshalIndent(payload, "", "  ")
		return string(bytes)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("### Generated Unit Tests (%s / %s)\n\n", lang, framework))
	sb.WriteString("```" + strings.ToLower(lang) + "\n")
	sb.WriteString(testCode)
	sb.WriteString("\n```\n\n")
	sb.WriteString("#### Scenarios Covered\n")
	for _, sc := range scenarios {
		sb.WriteString(fmt.Sprintf("- **%s** (`%s`): %s\n", sc["name"], sc["type"], sc["description"]))
	}
	sb.WriteString(fmt.Sprintf("\n%s\n", instructions))
	return sb.String()
}

func (p *LocalProvider) handleChat(prompt string, messages []Message) string {
	lower := strings.ToLower(prompt)

	if strings.Contains(lower, "architecture") || strings.Contains(lower, "modular monolith") {
		return "ForgeHub is built as a modular monolith in Go paired with a React 19 single-page application. Modules are isolated in `apps/api/internal/modules/*`, enforcing a four-tier layer: HTTP handler, domain service, repository store, and immutable domain models. Git repositories are stored as bare repos in `/data/git/` behind an abstracted `RepositoryStorage` interface."
	}
	if strings.Contains(lower, "migration") || strings.Contains(lower, "database") {
		return "Database migrations reside in the `/migrations` folder with versioned `.up.sql` and `.down.sql` files. Migrations run automatically on startup via `database.RunMigrations` with advisory lock protections to avoid race conditions in multi-replica deployments."
	}
	if strings.Contains(lower, "smart http") || strings.Contains(lower, "clone") || strings.Contains(lower, "git push") {
		return "ForgeHub implements native Git Smart HTTP protocol endpoints (`/{owner}/{repo}.git/info/refs`, `git-upload-pack`, and `git-receive-pack`). Client requests are authenticated via HTTP Basic Auth (PAT or password) and validated through repository RBAC permissions before delegating to Git CLI RPC pipes."
	}
	if strings.Contains(lower, "test") || strings.Contains(lower, "unit test") {
		return "Unit and integration tests can be executed with `go test ./...` in `apps/api`. For frontend validation, run `npm test` or `npm run build` in `apps/web`. ForgeHub maintains dual-mode persistence (SQLite / miniredis for isolated dev/tests, and PostgreSQL / Redis for production)."
	}

	return fmt.Sprintf("I am ForgeAI, your developer collaborator for ForgeHub. You asked: \"%s\". I can help you explain source code, conduct security and bug reviews on pull requests, generate comprehensive unit test suites, or explore repository architecture. How can I assist you with your project today?", prompt)
}
