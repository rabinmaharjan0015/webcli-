package web

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type SearchProvider interface {
	Search(query string, maxResults int) ([]SearchResult, error)
	Name() string
}

func NewSearchProvider(provider, apiKey, googleCX string, rateLimiter *RateLimiter, userAgent string) (SearchProvider, error) {
	switch strings.ToLower(provider) {
	case "", "duckduckgo":
		return NewDuckDuckGoProvider(rateLimiter, userAgent), nil
	case "tavily":
		if apiKey == "" {
			return nil, fmt.Errorf("tavily provider requires WEBCLI_SEARCH_API_KEY or search.api_key in config")
		}
		return NewTavilyProvider(apiKey, rateLimiter)
	case "google":
		if apiKey == "" || googleCX == "" {
			return nil, fmt.Errorf("google provider requires WEBCLI_SEARCH_API_KEY and WEBCLI_GOOGLE_CX (or config) to be set")
		}
		return NewGoogleProvider(apiKey, googleCX, rateLimiter, userAgent)
	default:
		return nil, fmt.Errorf("unknown search provider %q (supported: duckduckgo, tavily, google)", provider)
	}
}

type userAgentTransport struct {
	userAgent string
	inner     http.RoundTripper
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("User-Agent", t.userAgent)
	return t.inner.RoundTrip(req)
}

func extractTags(query string) []string {
	words := strings.Fields(strings.ToLower(query))
	var tags []string
	for _, w := range words {
		w = strings.Trim(w, ",.;:!?\"'()[]{}")
		if len(w) > 2 {
			tags = append(tags, w)
		}
	}
	return tags
}

func truncateString(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}
