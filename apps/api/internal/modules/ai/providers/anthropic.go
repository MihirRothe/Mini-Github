package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type AnthropicProvider struct {
	apiKey     string
	model      string
	maxTokens  int
	httpClient *http.Client
}

func NewAnthropicProvider(apiKey, model string, maxTokens int) *AnthropicProvider {
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	return &AnthropicProvider{
		apiKey:    apiKey,
		model:     model,
		maxTokens: maxTokens,
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

func (p *AnthropicProvider) ID() string {
	return "anthropic"
}

func (p *AnthropicProvider) Name() string {
	return "Anthropic Claude"
}

func (p *AnthropicProvider) Description() string {
	return "Anthropic Claude 3.5 Sonnet for nuanced software architecture and code review"
}

func (p *AnthropicProvider) DefaultModel() string {
	return p.model
}

func (p *AnthropicProvider) IsConfigured() bool {
	return p.apiKey != ""
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicReq struct {
	Model       string         `json:"model"`
	System      string         `json:"system,omitempty"`
	Messages    []anthropicMsg `json:"messages"`
	MaxTokens   int            `json:"max_tokens"`
	Temperature float64        `json:"temperature,omitempty"`
}

type anthropicResp struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *AnthropicProvider) Generate(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if !p.IsConfigured() {
		return nil, ErrProviderNotConfigured
	}

	url := "https://api.anthropic.com/v1/messages"

	var messages []anthropicMsg
	for _, m := range req.Messages {
		role := m.Role
		if role != "user" && role != "assistant" {
			role = "user"
		}
		messages = append(messages, anthropicMsg{
			Role:    role,
			Content: m.Content,
		})
	}

	if len(messages) == 0 && req.UserPrompt != "" {
		messages = append(messages, anthropicMsg{
			Role:    "user",
			Content: req.UserPrompt,
		})
	}

	body := anthropicReq{
		Model:       p.model,
		System:      req.SystemPrompt,
		Messages:    messages,
		MaxTokens:   p.maxTokens,
		Temperature: req.Temperature,
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to encode anthropic request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create anthropic http request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	httpResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read anthropic response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic returned status %d: %s", httpResp.StatusCode, string(respBody))
	}

	var aResp anthropicResp
	if err := json.Unmarshal(respBody, &aResp); err != nil {
		return nil, fmt.Errorf("failed to parse anthropic response: %w", err)
	}

	if aResp.Error != nil {
		return nil, fmt.Errorf("anthropic error: %s", aResp.Error.Message)
	}

	if len(aResp.Content) == 0 {
		return nil, fmt.Errorf("anthropic returned no content")
	}

	return &CompletionResponse{
		Content:  aResp.Content[0].Text,
		Model:    p.model,
		Provider: p.ID(),
	}, nil
}
