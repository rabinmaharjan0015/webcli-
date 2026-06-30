package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const anthropicURL = "https://api.anthropic.com/v1/messages"

type AnthropicProvider struct {
	apiKey string
	client *http.Client
}

func NewAnthropic(apiKey string) (*AnthropicProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("anthropic API key is required (set ANTHROPIC_API_KEY or WEBCLI_ANTHROPIC_KEY)")
	}
	return &AnthropicProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (a *AnthropicProvider) Name() string { return "anthropic" }

type anthropicReq struct {
	Model     string          `json:"model"`
	System    string          `json:"system,omitempty"`
	Messages  []anthropicMsg  `json:"messages"`
	MaxTokens int             `json:"max_tokens"`
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResp struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	Model      string `json:"model"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (a *AnthropicProvider) Chat(req ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	messages := make([]anthropicMsg, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = anthropicMsg{Role: m.Role, Content: m.Content}
	}

	body := anthropicReq{
		Model:     model,
		System:    req.System,
		Messages:  messages,
		MaxTokens: 4096,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequest("POST", anthropicURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	start := time.Now()
	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	duration := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result anthropicResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	var content string
	for _, c := range result.Content {
		content += c.Text
	}

	return &ChatResponse{
		Content:      content,
		Model:        result.Model,
		Provider:     "anthropic",
		InputTokens:  result.Usage.InputTokens,
		OutputTokens: result.Usage.OutputTokens,
		Duration:     duration,
	}, nil
}
