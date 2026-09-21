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

type OpenAIProvider struct {
	apiKey     string
	model      string
	maxTokens  int
	httpClient *http.Client
}

func NewOpenAIProvider(apiKey, model string, maxTokens int) *OpenAIProvider {
	if model == "" {
		model = "gpt-4o-mini"
	}
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	return &OpenAIProvider{
		apiKey:    apiKey,
		model:     model,
		maxTokens: maxTokens,
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

func (p *OpenAIProvider) ID() string {
	return "openai"
}

func (p *OpenAIProvider) Name() string {
	return "OpenAI"
}

func (p *OpenAIProvider) Description() string {
	return "OpenAI GPT-4o / GPT-4o-mini models for advanced code reasoning and refactoring"
}

func (p *OpenAIProvider) DefaultModel() string {
	return p.model
}

func (p *OpenAIProvider) IsConfigured() bool {
	return p.apiKey != ""
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIReq struct {
	Model          string          `json:"model"`
	Messages       []openAIMessage `json:"messages"`
	Temperature    float64         `json:"temperature,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	ResponseFormat *struct {
		Type string `json:"type"`
	} `json:"response_format,omitempty"`
}

type openAIResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func (p *OpenAIProvider) Generate(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if !p.IsConfigured() {
		return nil, ErrProviderNotConfigured
	}

	url := "https://api.openai.com/v1/chat/completions"

	var messages []openAIMessage
	if req.SystemPrompt != "" {
		messages = append(messages, openAIMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	for _, m := range req.Messages {
		messages = append(messages, openAIMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	if len(req.Messages) == 0 && req.UserPrompt != "" {
		messages = append(messages, openAIMessage{
			Role:    "user",
			Content: req.UserPrompt,
		})
	}

	body := openAIReq{
		Model:       p.model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   p.maxTokens,
	}

	if req.JSONMode {
		body.ResponseFormat = &struct {
			Type string `json:"type"`
		}{Type: "json_object"}
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to encode openai request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create openai http request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	httpResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read openai response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai returned status %d: %s", httpResp.StatusCode, string(respBody))
	}

	var oResp openAIResp
	if err := json.Unmarshal(respBody, &oResp); err != nil {
		return nil, fmt.Errorf("failed to parse openai response: %w", err)
	}

	if oResp.Error != nil {
		return nil, fmt.Errorf("openai error: %s", oResp.Error.Message)
	}

	if len(oResp.Choices) == 0 {
		return nil, fmt.Errorf("openai returned no choices")
	}

	return &CompletionResponse{
		Content:  oResp.Choices[0].Message.Content,
		Model:    p.model,
		Provider: p.ID(),
	}, nil
}
