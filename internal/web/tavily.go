package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const tavilyURL = "https://api.tavily.com/search"

type TavilyProvider struct {
	apiKey      string
	client      *http.Client
	rateLimiter *RateLimiter
}

func NewTavilyProvider(apiKey string, rateLimiter *RateLimiter) (*TavilyProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("tavily api key is required")
	}
	return &TavilyProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 15 * time.Second},
		rateLimiter: rateLimiter,
	}, nil
}

func (t *TavilyProvider) Name() string { return "tavily" }

type tavilyRequest struct {
	APIKey     string `json:"api_key"`
	Query      string `json:"query"`
	MaxResults int    `json:"max_results"`
}

type tavilyResponse struct {
	Results []struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Content string `json:"content"`
	} `json:"results"`
}

func (t *TavilyProvider) Search(query string, maxResults int) ([]SearchResult, error) {
	t.rateLimiter.Wait()

	body := tavilyRequest{
		APIKey:     t.apiKey,
		Query:      query,
		MaxResults: maxResults,
	}

	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := t.client.Post(tavilyURL, "application/json", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("tavily request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tavily returned HTTP %d", resp.StatusCode)
	}

	var result tavilyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode tavily response: %w", err)
	}

	results := make([]SearchResult, 0, len(result.Results))
	for _, r := range result.Results {
		results = append(results, SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Content,
		})
	}
	return results, nil
}
