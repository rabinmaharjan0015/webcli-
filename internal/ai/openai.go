package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const openaiURL = "https://api.openai.com/v1/chat/completions"

type OpenAIProvider struct {
	apiKey string
	client *http.Client
}

func NewOpenAI(apiKey string) (*OpenAIProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required (set OPENAI_API_KEY or WEBCLI_OPENAI_KEY)")
	}
	return &OpenAIProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (o *OpenAIProvider) Name() string { return "openai" }

type openaiReq struct {
	Model    string       `json:"model"`
	Messages []openaiMsg  `json:"messages"`
}

type openaiMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (o *OpenAIProvider) Chat(req ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = "gpt-4o"
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

	httpReq, err := http.NewRequest("POST", openaiURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	start := time.Now()
	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	duration := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai returned HTTP %d: %s", resp.StatusCode, string(respBody))
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
		Provider:     "openai",
		InputTokens:  result.Usage.PromptTokens,
		OutputTokens: result.Usage.CompletionTokens,
		Duration:     duration,
	}, nil
}
