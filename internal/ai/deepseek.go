package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const deepseekURL = "https://api.deepseek.com/v1/chat/completions"

type DeepSeekProvider struct {
	apiKey string
	client *http.Client
}

func NewDeepSeek(apiKey string) (*DeepSeekProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("DeepSeek API key is required (set DEEPSEEK_API_KEY or WEBCLI_DEEPSEEK_KEY)")
	}
	return &DeepSeekProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (d *DeepSeekProvider) Name() string { return "deepseek" }

func (d *DeepSeekProvider) Chat(req ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = "deepseek-chat"
	}

	messages := make([]openaiMsg, 0, len(req.Messages)+1)
	if req.System != "" {
		messages = append(messages, openaiMsg{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		messages = append(messages, openaiMsg{Role: m.Role, Content: m.Content})
	}

	body := openaiReq{
		Model:    model,
		Messages: messages,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequest("POST", deepseekURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+d.apiKey)

	start := time.Now()
	resp, err := d.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	duration := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("deepseek returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result openaiResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	var content string
	if len(result.Choices) > 0 {
		content = result.Choices[0].Message.Content
	}

	return &ChatResponse{
		Content:      content,
		Model:        result.Model,
		Provider:     "deepseek",
		InputTokens:  result.Usage.PromptTokens,
		OutputTokens: result.Usage.CompletionTokens,
		Duration:     duration,
	}, nil
}
