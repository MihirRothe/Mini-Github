package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"forgehub/apps/api/internal/modules/ai/providers"
	"forgehub/apps/api/internal/modules/pulls"
	"forgehub/apps/api/internal/modules/repos"
)

var (
	ErrBinaryFileNotSupported = errors.New("cannot perform AI analysis on binary files")
	ErrEmptyDiff              = errors.New("pull request contains no diff to review")
)

type Service interface {
	ExplainCode(ctx context.Context, req ExplainRequest, providerID string) (*ExplainResponse, error)
	ReviewDiff(ctx context.Context, req ReviewRequest, providerID string) (*ReviewResponse, error)
	ReviewPullRequest(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repo string, pullNumber int, postReviewComment bool, providerID string) (*ReviewResponse, error)
	GenerateTests(ctx context.Context, req GenerateTestsRequest, providerID string) (*GenerateTestsResponse, error)
	Chat(ctx context.Context, req ChatRequest, providerID string) (*ChatResponse, error)
	GetProvidersStatus() ProvidersResponse
	ExplainBlob(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repo, ref, path string, providerID string) (*ExplainResponse, error)
	GenerateBlobTests(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repo, ref, path string, providerID string) (*GenerateTestsResponse, error)
}

type service struct {
	manager  *providers.Manager
	reposSvc repos.Service
	pullsSvc pulls.Service
}

func NewService(mgr *providers.Manager, reposSvc repos.Service, pullsSvc pulls.Service) Service {
	return &service{
		manager:  mgr,
		reposSvc: reposSvc,
		pullsSvc: pullsSvc,
	}
}

func cleanJSONResponse(raw string) string {
	s := strings.TrimSpace(raw)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		if idx := strings.LastIndex(s, "```"); idx != -1 {
			s = s[:idx]
		}
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		if idx := strings.LastIndex(s, "```"); idx != -1 {
			s = s[:idx]
		}
	}
	return strings.TrimSpace(s)
}

func (s *service) ExplainCode(ctx context.Context, req ExplainRequest, providerID string) (*ExplainResponse, error) {
	if err := ValidateInput(req.Code); err != nil {
		return nil, err
	}

	p, err := s.manager.GetProvider(providerID)
	if err != nil {
		return nil, err
	}

	wrapped := WrapUntrustedContent("code", req.Code)
	if req.FilePath != "" {
		wrapped = fmt.Sprintf("File Path: %s\n%s", req.FilePath, wrapped)
	}

	systemPrompt := `You are ForgeAI, an expert software architect and code analysis assistant.
Analyze the provided code snippet thoroughly.
Output valid JSON with the following structure:
{
  "summary": "Brief executive summary of what this code accomplishes",
  "language": "Programming language name",
  "key_components": ["Symbol1: description", "Symbol2: description"],
  "complexity": {
    "time": "O(N)",
    "space": "O(1)",
    "description": "Explanation of computational complexity"
  },
  "architectural_role": "How this module fits into clean architecture",
  "potential_risks": ["Risk 1", "Risk 2"],
  "suggestions": ["Suggestion 1", "Suggestion 2"]
}`

	comp, err := p.Generate(ctx, providers.CompletionRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   wrapped,
		JSONMode:     true,
		Temperature:  0.2,
	})
	if err != nil {
		return nil, fmt.Errorf("provider execution failed: %w", err)
	}

	cleaned := cleanJSONResponse(comp.Content)
	var resp ExplainResponse
	if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
		// Fallback if provider returned raw markdown instead of valid JSON
		return &ExplainResponse{
			Summary:           "Code analysis completed.",
			Language:          req.Language,
			KeyComponents:     []string{"Analysis completed with formatted markdown output"},
			ArchitecturalRole: "Domain module implementation",
			RawMarkdown:       comp.Content,
		}, nil
	}

	resp.RawMarkdown = comp.Content
	return &resp, nil
}

func (s *service) ReviewDiff(ctx context.Context, req ReviewRequest, providerID string) (*ReviewResponse, error) {
	if err := ValidateInput(req.Diff); err != nil {
		return nil, err
	}

	p, err := s.manager.GetProvider(providerID)
	if err != nil {
		return nil, err
	}

	wrapped := WrapUntrustedContent("diff", req.Diff)
	if req.Title != "" {
		wrapped = fmt.Sprintf("Pull Request Title: %s\nDescription: %s\n%s", req.Title, req.Description, wrapped)
	}

	systemPrompt := `You are ForgeAI, an expert code reviewer and application security engineer.
Conduct an automated code review on the provided Git diff.
Analyze for:
1. Security vulnerabilities (SQL injection, XSS, unauthenticated paths, secret leaks)
2. Correctness bugs, nil pointer dereferences, race conditions
3. Performance issues, resource leaks, N+1 patterns
4. Style and idiomatic clean code conventions

Output valid JSON with the following structure:
{
  "verdict": "APPROVE" | "REQUEST_CHANGES" | "COMMENT",
  "summary": "High-level architectural assessment of the changeset",
  "score": 85,
  "findings": [
    {
      "file": "path/to/file.go",
      "line": 42,
      "severity": "CRITICAL" | "HIGH" | "MEDIUM" | "LOW" | "INFO",
      "category": "SECURITY" | "BUG" | "PERFORMANCE" | "STYLE",
      "title": "Short title",
      "description": "Clear explanation of the issue",
      "suggested_fix": "Concrete recommended replacement code or action"
    }
  ],
  "review_comment": "Complete formatted Markdown suitable for posting as a review comment"
}`

	comp, err := p.Generate(ctx, providers.CompletionRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   wrapped,
		JSONMode:     true,
		Temperature:  0.2,
	})
	if err != nil {
		return nil, fmt.Errorf("provider execution failed: %w", err)
	}

	cleaned := cleanJSONResponse(comp.Content)
	var resp ReviewResponse
	if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
		return &ReviewResponse{
			Verdict:       "COMMENT",
			Summary:       "Automated review completed.",
			Score:         80,
			ReviewComment: comp.Content,
		}, nil
	}

	return &resp, nil
}

func (s *service) ReviewPullRequest(
	ctx context.Context,
	currentUserID string,
	isSiteAdmin bool,
	owner, repo string,
	pullNumber int,
	postReviewComment bool,
	providerID string,
) (*ReviewResponse, error) {
	diffResult, err := s.pullsSvc.GetPRDiff(ctx, currentUserID, isSiteAdmin, owner, repo, pullNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pull request diff: %w", err)
	}

	if diffResult == nil || len(diffResult.Files) == 0 {
		return nil, ErrEmptyDiff
	}

	var diffBuilder strings.Builder
	for _, f := range diffResult.Files {
		diffBuilder.WriteString(fmt.Sprintf("diff --git a/%s b/%s\n", f.OldPath, f.NewPath))
		diffBuilder.WriteString(fmt.Sprintf("--- a/%s\n", f.OldPath))
		diffBuilder.WriteString(fmt.Sprintf("+++ b/%s\n", f.NewPath))
		diffBuilder.WriteString(f.Patch)
		diffBuilder.WriteString("\n")
	}
	unifiedDiff := diffBuilder.String()
	if strings.TrimSpace(unifiedDiff) == "" {
		return nil, ErrEmptyDiff
	}

	prDetail, err := s.pullsSvc.GetPullRequest(ctx, currentUserID, isSiteAdmin, owner, repo, pullNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pull request metadata: %w", err)
	}

	revResp, err := s.ReviewDiff(ctx, ReviewRequest{
		Diff:        unifiedDiff,
		BaseBranch:  prDetail.TargetBranch,
		HeadBranch:  prDetail.SourceBranch,
		Title:       prDetail.Title,
		Description: prDetail.Body,
	}, providerID)
	if err != nil {
		return nil, err
	}

	// Post review directly to PR if requested
	if postReviewComment && currentUserID != "" {
		var state pulls.ReviewState
		switch revResp.Verdict {
		case "APPROVE":
			state = pulls.ReviewApproved
		case "REQUEST_CHANGES":
			state = pulls.ReviewChangesRequested
		default:
			state = pulls.ReviewCommented
		}

		_, _ = s.pullsSvc.CreateReview(ctx, currentUserID, isSiteAdmin, owner, repo, pullNumber, pulls.CreateReviewRequest{
			State: state,
			Body:  revResp.ReviewComment,
		})
	}

	return revResp, nil
}

func (s *service) GenerateTests(ctx context.Context, req GenerateTestsRequest, providerID string) (*GenerateTestsResponse, error) {
	if err := ValidateInput(req.Code); err != nil {
		return nil, err
	}

	p, err := s.manager.GetProvider(providerID)
	if err != nil {
		return nil, err
	}

	wrapped := WrapUntrustedContent("code", req.Code)
	if req.FilePath != "" {
		wrapped = fmt.Sprintf("File Path: %s\nTarget Framework: %s\n%s", req.FilePath, req.Framework, wrapped)
	}

	systemPrompt := `You are ForgeAI, an expert test engineer and automated quality assurance assistant.
Generate a comprehensive, idiomatic, ready-to-run unit test suite for the provided code.
Cover:
- Happy paths with standard inputs
- Boundary thresholds and zero/empty values
- Error conditions and exception paths
- Concurrency or edge cases

Output valid JSON with the following structure:
{
  "language": "Go",
  "framework": "testing",
  "test_code": "package main ... complete runnable test code",
  "test_scenarios": [
    {
      "name": "Scenario Title",
      "description": "What this test scenario asserts",
      "type": "happy_path" | "edge_case" | "error_handling" | "boundary"
    }
  ],
  "instructions": "How to execute this test suite (e.g. go test -v ./...)"
}`

	comp, err := p.Generate(ctx, providers.CompletionRequest{
		SystemPrompt: systemPrompt,
		UserPrompt:   wrapped,
		JSONMode:     true,
		Temperature:  0.2,
	})
	if err != nil {
		return nil, fmt.Errorf("provider execution failed: %w", err)
	}

	cleaned := cleanJSONResponse(comp.Content)
	var resp GenerateTestsResponse
	if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
		return &GenerateTestsResponse{
			Language:     req.Language,
			Framework:    req.Framework,
			TestCode:     comp.Content,
			Instructions: "Run tests with your standard framework CLI.",
		}, nil
	}

	return &resp, nil
}

func (s *service) Chat(ctx context.Context, req ChatRequest, providerID string) (*ChatResponse, error) {
	if len(req.Messages) == 0 {
		return nil, ErrEmptyInput
	}

	p, err := s.manager.GetProvider(providerID)
	if err != nil {
		return nil, err
	}

	systemPrompt := "You are ForgeAI, an intelligent developer assistant integrated into ForgeHub. Help developers with code analysis, architecture, pull requests, debugging, and repository navigation."

	if req.Repository != "" || req.FilePath != "" || req.SelectedCode != "" {
		systemPrompt += fmt.Sprintf("\n\nContext:\n- Repository: %s\n- Ref: %s\n- File Path: %s", req.Repository, req.Ref, req.FilePath)
		if req.SelectedCode != "" {
			systemPrompt += "\n- Active Selection:\n" + WrapUntrustedContent("selection", req.SelectedCode)
		}
	}

	providerMsgs := make([]providers.Message, 0, len(req.Messages))
	for _, m := range req.Messages {
		providerMsgs = append(providerMsgs, providers.Message{
			Role:    m.Role,
			Content: SanitizeUserInput(m.Content),
		})
	}

	comp, err := p.Generate(ctx, providers.CompletionRequest{
		SystemPrompt: systemPrompt,
		Messages:     providerMsgs,
		Temperature:  0.4,
	})
	if err != nil {
		return nil, fmt.Errorf("provider execution failed: %w", err)
	}

	return &ChatResponse{
		Message: ChatMessage{
			Role:    "assistant",
			Content: comp.Content,
		},
		Model:    comp.Model,
		Provider: comp.Provider,
	}, nil
}

func (s *service) GetProvidersStatus() ProvidersResponse {
	summaries := s.manager.ListSummaries()
	provInfos := make([]ProviderInfo, 0, len(summaries))
	var activeID string

	for _, sum := range summaries {
		provInfos = append(provInfos, ProviderInfo{
			ID:          sum.ID,
			Name:        sum.Name,
			Active:      sum.Active,
			Configured:  sum.Configured,
			Model:       sum.Model,
			Description: sum.Description,
		})
		if sum.Active {
			activeID = sum.ID
		}
	}

	return ProvidersResponse{
		ActiveProvider: activeID,
		Providers:      provInfos,
	}
}

func (s *service) ExplainBlob(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repo, ref, path string, providerID string) (*ExplainResponse, error) {
	blob, err := s.reposSvc.GetBlob(ctx, currentUserID, isSiteAdmin, owner, repo, ref, path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blob: %w", err)
	}
	if blob.IsBinary {
		return nil, ErrBinaryFileNotSupported
	}

	return s.ExplainCode(ctx, ExplainRequest{
		Code:     blob.Content,
		FilePath: path,
	}, providerID)
}

func (s *service) GenerateBlobTests(ctx context.Context, currentUserID string, isSiteAdmin bool, owner, repo, ref, path string, providerID string) (*GenerateTestsResponse, error) {
	blob, err := s.reposSvc.GetBlob(ctx, currentUserID, isSiteAdmin, owner, repo, ref, path)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blob: %w", err)
	}
	if blob.IsBinary {
		return nil, ErrBinaryFileNotSupported
	}

	return s.GenerateTests(ctx, GenerateTestsRequest{
		Code:     blob.Content,
		FilePath: path,
	}, providerID)
}
