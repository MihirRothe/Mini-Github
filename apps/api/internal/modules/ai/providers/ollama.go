package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OllamaProvider struct {
	baseURL    string
	model      string
	maxTokens  int
	configured bool
	httpClient *http.Client
}

func NewOllamaProvider(baseURL, model string, maxTokens int) *OllamaProvider {
	configured := baseURL != ""
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "codellama"
	}
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	return &OllamaProvider{
		baseURL:    strings.TrimRight(baseURL, "/"),
		model:      model,
		maxTokens:  maxTokens,
		configured: configured,
		httpClient: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

func (p *OllamaProvider) ID() string {
	return "ollama"
}

func (p *OllamaProvider) Name() string {
	return "Ollama (Self-Hosted)"
}

func (p *OllamaProvider) Description() string {
	return "Self-hosted local open-source LLMs running privately via Ollama"
}

func (p *OllamaProvider) DefaultModel() string {
	return p.model
}

func (p *OllamaProvider) IsConfigured() bool {
	return p.configured
}

type ollamaReq struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	System string `json:"system,omitempty"`
	Stream bool   `json:"stream"`
	Format string `json:"format,omitempty"`
}

type ollamaResp struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

func (p *OllamaProvider) Generate(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	url := p.baseURL + "/api/generate"

	prompt := req.UserPrompt
	if len(req.Messages) > 0 {
		var sb strings.Builder
		for _, m := range req.Messages {
			sb.WriteString(fmt.Sprintf("%s: %s\n\n", strings.Title(m.Role), m.Content))
		}
		prompt = sb.String()
	}

	body := ollamaReq{
		Model:  p.model,
		Prompt: prompt,
		System: req.SystemPrompt,
		Stream: false,
	}

	if req.JSONMode {
		body.Format = "json"
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to encode ollama request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama connection failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read ollama response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d: %s", httpResp.StatusCode, string(respBody))
	}

	var oResp ollamaResp
	if err := json.Unmarshal(respBody, &oResp); err != nil {
		return nil, fmt.Errorf("failed to parse ollama response: %w", err)
	}

	if oResp.Error != "" {
		return nil, fmt.Errorf("ollama error: %s", oResp.Error)
	}

	return &CompletionResponse{
		Content:  oResp.Response,
		Model:    p.model,
		Provider: p.ID(),
	}, nil
}
