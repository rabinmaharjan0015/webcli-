package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const geminiURL = "https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s"

type GeminiProvider struct {
	apiKey string
	client *http.Client
}

func NewGemini(apiKey string) (*GeminiProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Google AI API key is required (set GOOGLE_API_KEY or WEBCLI_GOOGLE_KEY)")
	}
	return &GeminiProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (g *GeminiProvider) Name() string { return "google" }

type geminiReq struct {
	Contents []geminiContent `json:"contents"`
	SystemInstruction *geminiContent `json:"system_instruction,omitempty"`
}

type geminiContent struct {
	Role  string      `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResp struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
}

func (g *GeminiProvider) Chat(req ChatRequest) (*ChatResponse, error) {
	model := req.Model
	if model == "" {
		model = "gemini-2.5-flash"
	}

	contents := make([]geminiContent, 0, len(req.Messages))

	// Map messages: first user message is user, rest alternate
	for _, m := range req.Messages {
		role := m.Role
		if role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: m.Content}},
		})
	}

	gReq := geminiReq{Contents: contents}
	if req.System != "" {
		gReq.SystemInstruction = &geminiContent{
			Parts: []geminiPart{{Text: req.System}},
		}
	}

	data, err := json.Marshal(gReq)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}

	url := fmt.Sprintf(geminiURL, model, g.apiKey)
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := g.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	duration := time.Since(start)

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result geminiResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	var content string
	if len(result.Candidates) > 0 {
		for _, p := range result.Candidates[0].Content.Parts {
			content += p.Text
		}
	}

	return &ChatResponse{
		Content:      content,
		Model:        model,
		Provider:     "google",
		InputTokens:  result.UsageMetadata.PromptTokenCount,
		OutputTokens: result.UsageMetadata.CandidatesTokenCount,
		Duration:     duration,
	}, nil
}
