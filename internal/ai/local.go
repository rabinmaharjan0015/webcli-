package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type LocalProvider struct {
	endpoint string
	model    string
	apiKey   string
	client   *http.Client
}

func NewLocal(endpoint, model, apiKey string) (*LocalProvider, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("local endpoint URL is required (set WEBCLI_LOCAL_ENDPOINT)")
	}
	if model == "" {
		return nil, fmt.Errorf("local model name is required (set WEBCLI_LOCAL_MODEL)")
	}
	return &LocalProvider{
		endpoint: endpoint,
		model:    model,
		apiKey:   apiKey,
		client:   &http.Client{Timeout: 300 * time.Second},
	}, nil
}

func (l *LocalProvider) Name() string { return "local" }

func (l *LocalProvider) Chat(req ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = l.model
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

	httpReq, err := http.NewRequest("POST", l.endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if l.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+l.apiKey)
	}

	start := time.Now()
	resp, err := l.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("local endpoint request failed: %w", err)
	}
	defer resp.Body.Close()
	duration := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("local endpoint returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result openaiResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	var content string
	if len(result.Choices) > 0 {
		content = result.Choices[0].Message.Content
	}

	providerName := fmt.Sprintf("local (%s)", model)
	return &ChatResponse{
		Content:      content,
		Model:        result.Model,
		Provider:     providerName,
		InputTokens:  result.Usage.PromptTokens,
		OutputTokens: result.Usage.CompletionTokens,
		Duration:     duration,
	}, nil
}
