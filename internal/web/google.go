package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const googleSearchURL = "https://www.googleapis.com/customsearch/v1"

type GoogleProvider struct {
	apiKey      string
	cx          string
	client      *http.Client
	rateLimiter *RateLimiter
}

func NewGoogleProvider(apiKey, cx string, rateLimiter *RateLimiter, userAgent string) (*GoogleProvider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("google api key is required")
	}
	if cx == "" {
		return nil, fmt.Errorf("google custom search engine ID (cx) is required")
	}
	return &GoogleProvider{
		apiKey: apiKey,
		cx:     cx,
		client: &http.Client{Timeout: 15 * time.Second},
		rateLimiter: rateLimiter,
	}, nil
}

func (g *GoogleProvider) Name() string { return "google" }

type googleResp struct {
	Items []struct {
		Title   string `json:"title"`
		Link    string `json:"link"`
		Snippet string `json:"snippet"`
	} `json:"items"`
}

func (g *GoogleProvider) Search(query string, maxResults int) ([]SearchResult, error) {
	g.rateLimiter.Wait()

	u, _ := url.Parse(googleSearchURL)
	q := u.Query()
	q.Set("key", g.apiKey)
	q.Set("cx", g.cx)
	q.Set("q", query)
	q.Set("num", fmt.Sprintf("%d", maxResults))
	u.RawQuery = q.Encode()

	resp, err := g.client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("google search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google search returned HTTP %d", resp.StatusCode)
	}

	var result googleResp
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode google response: %w", err)
	}

	results := make([]SearchResult, 0, len(result.Items))
	for _, item := range result.Items {
		results = append(results, SearchResult{
			Title:   item.Title,
			URL:     item.Link,
			Snippet: item.Snippet,
		})
	}
	return results, nil
}
