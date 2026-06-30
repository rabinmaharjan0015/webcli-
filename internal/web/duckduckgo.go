package web

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

const duckduckgoURL = "https://html.duckduckgo.com/html/"

type DuckDuckGoProvider struct {
	client *http.Client
}

func NewDuckDuckGoProvider(rateLimiter *RateLimiter, userAgent string) *DuckDuckGoProvider {
	return &DuckDuckGoProvider{
		client: &http.Client{
			Timeout: defaultTimeout,
			Transport: &userAgentTransport{
				userAgent: userAgent,
				inner:     http.DefaultTransport,
			},
		},
	}
}

func (d *DuckDuckGoProvider) Name() string { return "duckduckgo" }

func (d *DuckDuckGoProvider) Search(query string, maxResults int) ([]SearchResult, error) {
	form := url.Values{}
	form.Set("q", query)

	resp, err := d.client.PostForm(duckduckgoURL, form)
	if err != nil {
		return nil, fmt.Errorf("duckduckgo request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("duckduckgo returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parse duckduckgo results: %w", err)
	}

	var results []SearchResult
	doc.Find(".result").Each(func(i int, sel *goquery.Selection) {
		if i >= maxResults {
			return
		}
		titleEl := sel.Find(".result__title a")
		snippetEl := sel.Find(".result__snippet")

		href, _ := titleEl.Attr("href")
		parsed, err := url.Parse(href)
		if err == nil {
			if redirect := parsed.Query().Get("uddg"); redirect != "" {
				if decoded, err := url.QueryUnescape(redirect); err == nil {
					href = decoded
				}
			}
		}

		result := SearchResult{
			Title:   strings.TrimSpace(titleEl.Text()),
			URL:     href,
			Snippet: strings.TrimSpace(snippetEl.Text()),
		}
		if result.Title != "" {
			results = append(results, result)
		}
	})

	if results == nil {
		results = []SearchResult{}
	}
	return results, nil
}
