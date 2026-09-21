package ai

// ExplainRequest represents input for code explanation.
type ExplainRequest struct {
	Code      string `json:"code"`
	Language  string `json:"language,omitempty"`
	FilePath  string `json:"file_path,omitempty"`
	Context   string `json:"context,omitempty"`
}

// ExplainResponse represents structured code analysis.
type ExplainResponse struct {
	Summary           string                 `json:"summary"`
	Language          string                 `json:"language"`
	KeyComponents     []string               `json:"key_components"`
	Complexity        ComplexityInfo         `json:"complexity"`
	ArchitecturalRole string                 `json:"architectural_role"`
	PotentialRisks    []string               `json:"potential_risks"`
	Suggestions       []string               `json:"suggestions"`
	RawMarkdown       string                 `json:"raw_markdown,omitempty"`
}

// ComplexityInfo contains estimated computational complexity.
type ComplexityInfo struct {
	Time        string `json:"time"`
	Space       string `json:"space"`
	Description string `json:"description"`
}

// ReviewRequest represents input for automated PR / diff code review.
type ReviewRequest struct {
	Diff        string `json:"diff"`
	BaseBranch  string `json:"base_branch,omitempty"`
	HeadBranch  string `json:"head_branch,omitempty"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
}

// ReviewFinding represents an actionable line or file finding.
type ReviewFinding struct {
	File         string `json:"file"`
	Line         int    `json:"line,omitempty"`
	Severity     string `json:"severity"` // CRITICAL, HIGH, MEDIUM, LOW, INFO
	Category     string `json:"category"` // SECURITY, BUG, PERFORMANCE, STYLE, ARCHITECTURE
	Title        string `json:"title"`
	Description  string `json:"description"`
	SuggestedFix string `json:"suggested_fix,omitempty"`
}

// ReviewResponse represents an automated code review output.
type ReviewResponse struct {
	Verdict       string          `json:"verdict"` // APPROVE, REQUEST_CHANGES, COMMENT
	Summary       string          `json:"summary"`
	Score         int             `json:"score"`   // 1 - 100 quality score
	Findings      []ReviewFinding `json:"findings"`
	ReviewComment string          `json:"review_comment"` // Ready-to-publish Markdown
}

// GenerateTestsRequest represents input for unit test generation.
type GenerateTestsRequest struct {
	Code      string `json:"code"`
	Language  string `json:"language,omitempty"`
	FilePath  string `json:"file_path,omitempty"`
	Framework string `json:"framework,omitempty"`
}

// TestCaseDescription summarizes an individual generated test case scenario.
type TestCaseDescription struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"` // happy_path, edge_case, error_handling, boundary
}

// GenerateTestsResponse represents generated test suite and instructions.
type GenerateTestsResponse struct {
	Language      string                `json:"language"`
	Framework     string                `json:"framework"`
	TestCode      string                `json:"test_code"`
	TestScenarios []TestCaseDescription `json:"test_scenarios"`
	Instructions  string                `json:"instructions"`
}

// ChatMessage represents a conversational prompt or reply.
type ChatMessage struct {
	Role    string `json:"role"` // system, user, assistant
	Content string `json:"content"`
}

// ChatRequest represents an interactive chat turn with contextual metadata.
type ChatRequest struct {
	Messages     []ChatMessage `json:"messages"`
	Repository   string        `json:"repository,omitempty"`
	Ref          string        `json:"ref,omitempty"`
	FilePath     string        `json:"file_path,omitempty"`
	SelectedCode string        `json:"selected_code,omitempty"`
}

// ChatResponse represents conversational assistant response.
type ChatResponse struct {
	Message  ChatMessage `json:"message"`
	Model    string      `json:"model"`
	Provider string      `json:"provider"`
}

// ProviderInfo represents telemetry and status of an AI engine.
type ProviderInfo struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Active      bool   `json:"active"`
	Configured  bool   `json:"configured"`
	Model       string `json:"model"`
	Description string `json:"description"`
}

// ProvidersResponse lists available providers and default selection.
type ProvidersResponse struct {
	ActiveProvider string         `json:"active_provider"`
	Providers      []ProviderInfo `json:"providers"`
}
