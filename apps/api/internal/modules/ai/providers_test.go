package ai

import (
	"context"
	"strings"
	"testing"

	"forgehub/apps/api/internal/config"
	"forgehub/apps/api/internal/modules/ai/providers"
)

func TestProviderManagerFallback(t *testing.T) {
	cfg := &config.Config{
		AIProvider: "auto",
	}

	mgr := providers.NewManager(cfg)
	def := mgr.GetDefault()

	if def.ID() != "local" {
		t.Fatalf("Expected default provider to be 'local' without API keys, got %s", def.ID())
	}

	summaries := mgr.ListSummaries()
	if len(summaries) < 5 {
		t.Errorf("Expected at least 5 registered providers, got %d", len(summaries))
	}

	// Unconfigured provider should error
	_, err := mgr.GetProvider("gemini")
	if err == nil {
		t.Errorf("Expected error requesting unconfigured gemini provider without key")
	}

	// Local provider should always be found and configured
	local, err := mgr.GetProvider("local")
	if err != nil {
		t.Fatalf("Failed to get local provider: %v", err)
	}
	if !local.IsConfigured() {
		t.Errorf("Local provider must always be marked configured")
	}
}

func TestLocalProviderExplain(t *testing.T) {
	p := providers.NewLocalProvider()
	ctx := context.Background()

	code := `package auth

import "context"

type TokenService interface {
	ValidateToken(ctx context.Context, token string) bool
}

func HashPassword(raw string) (string, error) {
	for i := 0; i < 10; i++ {
		// nested iteration
		for j := 0; j < 5; j++ {}
	}
	return "hashed", nil
}
`

	resp, err := p.Generate(ctx, providers.CompletionRequest{
		SystemPrompt: "Code explanation assistant",
		UserPrompt:   code,
		JSONMode:     true,
	})
	if err != nil {
		t.Fatalf("Local provider explain failed: %v", err)
	}

	if !strings.Contains(resp.Content, "HashPassword") {
		t.Errorf("Expected explanation to mention HashPassword function, got: %s", resp.Content)
	}
	if !strings.Contains(resp.Content, "Go") {
		t.Errorf("Expected Go language detection, got: %s", resp.Content)
	}
}

func TestLocalProviderReviewSecurityFinding(t *testing.T) {
	p := providers.NewLocalProvider()
	ctx := context.Background()

	insecureDiff := `diff --git a/internal/config.go b/internal/config.go
--- a/internal/config.go
+++ b/internal/config.go
@@ -10,2 +10,3 @@
+	apiKey := "super_secret_token_12345"
+	query := fmt.Sprintf("SELECT * FROM users WHERE email = '%s'", email)
`

	resp, err := p.Generate(ctx, providers.CompletionRequest{
		SystemPrompt: "Code review diff",
		UserPrompt:   insecureDiff,
		JSONMode:     true,
	})
	if err != nil {
		t.Fatalf("Local provider review failed: %v", err)
	}

	if !strings.Contains(resp.Content, "REQUEST_CHANGES") {
		t.Errorf("Expected REQUEST_CHANGES verdict for hardcoded secret and SQL injection, got: %s", resp.Content)
	}
	if !strings.Contains(resp.Content, "SECURITY") {
		t.Errorf("Expected SECURITY finding category, got: %s", resp.Content)
	}
}

func TestLocalProviderGenerateTests(t *testing.T) {
	p := providers.NewLocalProvider()
	ctx := context.Background()

	code := `package math

func Add(a, b int) int {
	return a + b
}
`

	resp, err := p.Generate(ctx, providers.CompletionRequest{
		SystemPrompt: "Generate unit tests",
		UserPrompt:   code,
		JSONMode:     true,
	})
	if err != nil {
		t.Fatalf("Local provider generate tests failed: %v", err)
	}

	if !strings.Contains(resp.Content, "TestCoreFunctionality") && !strings.Contains(resp.Content, "testing") {
		t.Errorf("Expected Go testing framework in generated tests, got: %s", resp.Content)
	}
}

func TestLocalProviderChat(t *testing.T) {
	p := providers.NewLocalProvider()
	ctx := context.Background()

	resp, err := p.Generate(ctx, providers.CompletionRequest{
		SystemPrompt: "Developer assistant",
		UserPrompt:   "Explain the architecture of ForgeHub",
	})
	if err != nil {
		t.Fatalf("Local provider chat failed: %v", err)
	}

	if !strings.Contains(resp.Content, "modular monolith") {
		t.Errorf("Expected architectural description to mention modular monolith, got: %s", resp.Content)
	}
}
