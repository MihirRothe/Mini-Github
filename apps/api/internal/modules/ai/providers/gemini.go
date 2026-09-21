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

type GeminiProvider struct {
	apiKey     string
	model      string
	maxTokens  int
	httpClient *http.Client
}

func NewGeminiProvider(apiKey, model string, maxTokens int) *GeminiProvider {
	if model == "" {
		model = "gemini-1.5-flash"
	}
	if maxTokens <= 0 {
		maxTokens = 4096
	}
	return &GeminiProvider{
		apiKey:    apiKey,
		model:     model,
		maxTokens: maxTokens,
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
	}
}

func (p *GeminiProvider) ID() string {
	return "gemini"
}

func (p *GeminiProvider) Name() string {
	return "Google Gemini"
}

func (p *GeminiProvider) Description() string {
	return "Google Gemini models for high-speed multi-modal code analysis and generation"
}

func (p *GeminiProvider) DefaultModel() string {
	return p.model
}

func (p *GeminiProvider) IsConfigured() bool {
	return p.apiKey != ""
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiConfig struct {
	Temperature      float64 `json:"temperature,omitempty"`
	MaxOutputTokens  int     `json:"maxOutputTokens,omitempty"`
	ResponseMimeType string  `json:"responseMimeType,omitempty"`
}

type geminiReq struct {
	SystemInstruction *geminiContent `json:"system_instruction,omitempty"`
	Contents          []geminiContent `json:"contents"`
	GenerationConfig  geminiConfig    `json:"generationConfig"`
}

type geminiResp struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

func (p *GeminiProvider) Generate(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	if !p.IsConfigured() {
		return nil, ErrProviderNotConfigured
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", p.model, p.apiKey)

	var contents []geminiContent
	for _, m := range req.Messages {
		role := "user"
		if m.Role == "assistant" || m.Role == "model" {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}

	if len(contents) == 0 && req.UserPrompt != "" {
		contents = append(contents, geminiContent{
			Role:  "user",
			Parts: []geminiPart{{Text: req.UserPrompt}},
		})
	}

	gReq := geminiReq{
		Contents: contents,
		GenerationConfig: geminiConfig{
			Temperature:     req.Temperature,
			MaxOutputTokens: p.maxTokens,
		},
	}

	if req.SystemPrompt != "" {
		gReq.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: req.SystemPrompt}},
		}
	}

	if req.JSONMode {
		gReq.GenerationConfig.ResponseMimeType = "application/json"
	}

	bodyBytes, err := json.Marshal(gReq)
	if err != nil {
		return nil, fmt.Errorf("failed to encode gemini request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("gemini api request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read gemini response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini api returned status %d: %s", httpResp.StatusCode, string(respBody))
	}

	var gResp geminiResp
	if err := json.Unmarshal(respBody, &gResp); err != nil {
		return nil, fmt.Errorf("failed to parse gemini response: %w", err)
	}

	if gResp.Error != nil {
		return nil, fmt.Errorf("gemini error (%d): %s", gResp.Error.Code, gResp.Error.Message)
	}

	if len(gResp.Candidates) == 0 || len(gResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini returned no content candidates")
	}

	return &CompletionResponse{
		Content:  gResp.Candidates[0].Content.Parts[0].Text,
		Model:    p.model,
		Provider: p.ID(),
	}, nil
}
