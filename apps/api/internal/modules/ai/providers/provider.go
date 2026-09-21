package providers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"forgehub/apps/api/internal/config"
)

var (
	ErrProviderNotConfigured = errors.New("requested AI provider is not configured")
	ErrProviderNotFound     = errors.New("requested AI provider not found")
)

// Message represents a prompt turn.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CompletionRequest defines parameters for text/code completion.
type CompletionRequest struct {
	SystemPrompt string    `json:"system_prompt"`
	UserPrompt   string    `json:"user_prompt"`
	Messages     []Message `json:"messages,omitempty"`
	MaxTokens    int       `json:"max_tokens,omitempty"`
	Temperature  float64   `json:"temperature,omitempty"`
	JSONMode     bool      `json:"json_mode,omitempty"`
}

// CompletionResponse holds the generated text and provider metadata.
type CompletionResponse struct {
	Content  string `json:"content"`
	Model    string `json:"model"`
	Provider string `json:"provider"`
}

// Provider represents an LLM engine or local heuristic provider.
type Provider interface {
	ID() string
	Name() string
	Description() string
	DefaultModel() string
	IsConfigured() bool
	Generate(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
}

// Manager orchestrates provider registration, selection, and fallback.
type Manager struct {
	providers       map[string]Provider
	defaultProvider string
}

// NewManager creates and registers all supported providers based on config.
func NewManager(cfg *config.Config) *Manager {
	m := &Manager{
		providers: make(map[string]Provider),
	}

	// 1. Local Heuristic Engine (Always available, 100% offline)
	localProv := NewLocalProvider()
	m.Register(localProv)

	// 2. Google Gemini Provider
	geminiProv := NewGeminiProvider(cfg.GeminiAPIKey, cfg.GeminiModel, cfg.AIMaxTokens)
	m.Register(geminiProv)

	// 3. OpenAI Provider
	openaiProv := NewOpenAIProvider(cfg.OpenAIAPIKey, cfg.OpenAIModel, cfg.AIMaxTokens)
	m.Register(openaiProv)

	// 4. Anthropic Claude Provider
	anthropicProv := NewAnthropicProvider(cfg.AnthropicAPIKey, cfg.AnthropicModel, cfg.AIMaxTokens)
	m.Register(anthropicProv)

	// 5. Ollama Local LLM Provider
	ollamaProv := NewOllamaProvider(cfg.OllamaBaseURL, cfg.OllamaModel, cfg.AIMaxTokens)
	m.Register(ollamaProv)

	// Determine default provider
	m.defaultProvider = m.resolveDefaultProvider(cfg)

	return m
}

func (m *Manager) Register(p Provider) {
	m.providers[strings.ToLower(p.ID())] = p
}

func (m *Manager) resolveDefaultProvider(cfg *config.Config) string {
	pref := strings.ToLower(strings.TrimSpace(cfg.AIProvider))
	if pref != "" && pref != "auto" {
		if p, ok := m.providers[pref]; ok && p.IsConfigured() {
			return p.ID()
		}
	}

	// Auto-detection based on configured keys
	if m.providers["gemini"].IsConfigured() {
		return "gemini"
	}
	if m.providers["openai"].IsConfigured() {
		return "openai"
	}
	if m.providers["anthropic"].IsConfigured() {
		return "anthropic"
	}
	if m.providers["ollama"].IsConfigured() {
		return "ollama"
	}

	// Default fallback: Local Heuristic Engine
	return "local"
}

// GetDefault returns the resolved active provider.
func (m *Manager) GetDefault() Provider {
	if p, ok := m.providers[m.defaultProvider]; ok {
		return p
	}
	return m.providers["local"]
}

// GetProvider retrieves a provider by ID or returns an error.
func (m *Manager) GetProvider(id string) (Provider, error) {
	if id == "" || id == "default" || id == "auto" {
		return m.GetDefault(), nil
	}
	p, ok := m.providers[strings.ToLower(id)]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, id)
	}
	if !p.IsConfigured() {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotConfigured, id)
	}
	return p, nil
}

// ProviderSummary provides public info on provider configuration status.
type ProviderSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Active      bool   `json:"active"`
	Configured  bool   `json:"configured"`
	Model       string `json:"model"`
	Description string `json:"description"`
}

// ListSummaries returns status metadata for all providers.
func (m *Manager) ListSummaries() []ProviderSummary {
	ids := []string{"local", "gemini", "openai", "anthropic", "ollama"}
	res := make([]ProviderSummary, 0, len(ids))
	for _, id := range ids {
		p, ok := m.providers[id]
		if !ok {
			continue
		}
		res = append(res, ProviderSummary{
			ID:          p.ID(),
			Name:        p.Name(),
			Active:      p.ID() == m.defaultProvider,
			Configured:  p.IsConfigured(),
			Model:       p.DefaultModel(),
			Description: p.Description(),
		})
	}
	return res
}
