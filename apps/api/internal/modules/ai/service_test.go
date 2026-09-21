package ai

import (
	"context"
	"testing"

	"forgehub/apps/api/internal/config"
	"forgehub/apps/api/internal/modules/ai/providers"
)

func TestServiceExplainCode(t *testing.T) {
	cfg := &config.Config{AIProvider: "local"}
	mgr := providers.NewManager(cfg)
	svc := NewService(mgr, nil, nil)
	ctx := context.Background()

	resp, err := svc.ExplainCode(ctx, ExplainRequest{
		Code: `package utils

func Sanitize(input string) string {
	return strings.TrimSpace(input)
}
`,
		Language: "Go",
		FilePath: "utils.go",
	}, "local")

	if err != nil {
		t.Fatalf("ExplainCode returned error: %v", err)
	}

	if resp == nil {
		t.Fatal("ExplainCode returned nil response")
	}

	if resp.Language != "Go" {
		t.Errorf("Expected language Go, got %s", resp.Language)
	}
	if len(resp.KeyComponents) == 0 {
		t.Errorf("Expected detected components, got empty")
	}
}

func TestServiceReviewDiff(t *testing.T) {
	cfg := &config.Config{AIProvider: "local"}
	mgr := providers.NewManager(cfg)
	svc := NewService(mgr, nil, nil)
	ctx := context.Background()

	diff := `diff --git a/server.go b/server.go
--- a/server.go
+++ b/server.go
@@ -1,5 +1,6 @@
 func Start() {
+	token := "secret_12345_token"
+	console.log("Starting server")
 }
`

	resp, err := svc.ReviewDiff(ctx, ReviewRequest{
		Diff:  diff,
		Title: "Add logging and token",
	}, "local")

	if err != nil {
		t.Fatalf("ReviewDiff returned error: %v", err)
	}

	if resp.Verdict != "REQUEST_CHANGES" {
		t.Errorf("Expected REQUEST_CHANGES verdict, got %s", resp.Verdict)
	}

	if len(resp.Findings) == 0 {
		t.Errorf("Expected findings for hardcoded secret, got 0")
	}

	if resp.ReviewComment == "" {
		t.Errorf("Expected formatted ReviewComment")
	}
}

func TestServiceGenerateTests(t *testing.T) {
	cfg := &config.Config{AIProvider: "local"}
	mgr := providers.NewManager(cfg)
	svc := NewService(mgr, nil, nil)
	ctx := context.Background()

	resp, err := svc.GenerateTests(ctx, GenerateTestsRequest{
		Code: `export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  return bytes + ' B';
}
`,
		FilePath: "format.ts",
	}, "local")

	if err != nil {
		t.Fatalf("GenerateTests returned error: %v", err)
	}

	if resp.Framework == "" {
		t.Errorf("Expected non-empty test framework")
	}
	if resp.TestCode == "" {
		t.Errorf("Expected generated test code")
	}
	if len(resp.TestScenarios) == 0 {
		t.Errorf("Expected test scenarios list")
	}
}

func TestServiceChat(t *testing.T) {
	cfg := &config.Config{AIProvider: "local"}
	mgr := providers.NewManager(cfg)
	svc := NewService(mgr, nil, nil)
	ctx := context.Background()

	resp, err := svc.Chat(ctx, ChatRequest{
		Messages: []ChatMessage{
			{Role: "user", Content: "How are bare git repositories stored?"},
		},
		Repository: "MihirRothe/Mini-Github",
	}, "local")

	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	if resp.Message.Content == "" {
		t.Errorf("Expected non-empty chat assistant message")
	}
}

func TestServiceGetProvidersStatus(t *testing.T) {
	cfg := &config.Config{AIProvider: "local"}
	mgr := providers.NewManager(cfg)
	svc := NewService(mgr, nil, nil)

	resp := svc.GetProvidersStatus()
	if resp.ActiveProvider != "local" {
		t.Errorf("Expected active provider 'local', got %s", resp.ActiveProvider)
	}
	if len(resp.Providers) == 0 {
		t.Errorf("Expected non-empty list of providers")
	}
}
