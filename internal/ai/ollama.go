package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

func NewOllama(baseURL, model string) (*OllamaProvider, error) {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "qwen2.5"
	}
	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: 300 * time.Second},
	}, nil
}

func (o *OllamaProvider) Name() string { return "ollama" }

type ollamaReq struct {
	Model    string        `json:"model"`
	Messages []openaiMsg   `json:"messages"`
	Stream   bool          `json:"stream"`
}

type ollamaResp struct {
	Message struct {
		Content string `json:"content"`
	} `json:"message"`
	Model     string `json:"model"`
	Done      bool   `json:"done"`
	EvalCount int    `json:"eval_count"`
	PromptEvalCount int `json:"prompt_eval_count"`
}

func (o *OllamaProvider) Chat(req ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = o.model
	}

	messages := make([]openaiMsg, 0, len(req.Messages)+1)
	if req.System != "" {
		messages = append(messages, openaiMsg{Role: "system", Content: req.System})
	}
	for _, m := range req.Messages {
		messages = append(messages, openaiMsg{Role: m.Role, Content: m.Content})
	}

	body := ollamaReq{
		Model:    model,
		Messages: messages,
		Stream:   false,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	url := fmt.Sprintf("%s/api/chat", o.baseURL)
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := o.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()
	duration := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result ollamaResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode ollama response: %w", err)
	}

	providerName := fmt.Sprintf("ollama (%s)", model)
	return &ChatResponse{
		Content:      result.Message.Content,
		Model:        model,
		Provider:     providerName,
		InputTokens:  result.PromptEvalCount,
		OutputTokens: result.EvalCount,
		Duration:     duration,
	}, nil
}

// CheckOllama checks if Ollama is running and accessible
func CheckOllama(baseURL string) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("%s/api/tags", baseURL))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
