package web

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/net/html"
)

const (
	maxBodySize   = 5 * 1024 * 1024
	maxTextRunes  = 10000
	maxRedirects  = 5
)

type FetchResult struct {
	URL        string `json:"url"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	TextLength int    `json:"text_length"`
}

type Fetcher struct {
	client      *http.Client
	rateLimiter *RateLimiter
}

func NewFetcher(timeout time.Duration, userAgent string, rateLimiter *RateLimiter) *Fetcher {
	if rateLimiter == nil {
		rateLimiter = NewRateLimiter(30, 5)
	}
	return &Fetcher{
		client: &http.Client{
			Timeout: timeout,
			Transport: &userAgentTransport{
				userAgent: userAgent,
				inner:     http.DefaultTransport,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= maxRedirects {
					return fmt.Errorf("too many redirects (%d)", maxRedirects)
				}
				return nil
			},
		},
		rateLimiter: rateLimiter,
	}
}

func (f *Fetcher) Fetch(rawURL string) (*FetchResult, error) {
	f.rateLimiter.Wait()

	if rawURL == "" {
		return nil, fmt.Errorf("url cannot be empty")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url %q: %w", rawURL, err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("unsupported scheme %q (only http/https allowed)", parsed.Scheme)
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %q: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %q returned HTTP %d %s", rawURL, resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") &&
		!strings.HasPrefix(contentType, "text/plain") {
		return nil, fmt.Errorf("unsupported content type %q (only text/html and text/plain allowed)", contentType)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("empty response body")
	}

	finalURL := rawURL
	if resp.Request != nil && resp.Request.URL != nil {
		finalURL = resp.Request.URL.String()
	}

	title, text := extractContent(string(body))

	return &FetchResult{
		URL:        finalURL,
		Title:      title,
		Content:    text,
		TextLength: utf8.RuneCountInString(text),
	}, nil
}

func extractContent(htmlContent string) (string, string) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		fallback := truncateText(htmlContent, maxTextRunes)
		return "", fallback
	}

	var pageTitle strings.Builder
	var buf strings.Builder
	inTitle := false

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n == nil {
			return
		}

		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "noscript", "iframe", "svg", "nav", "footer", "header":
				return
			}
			if n.Data == "title" {
				inTitle = true
			}
		}

		if n.Type == html.TextNode {
			t := strings.TrimSpace(n.Data)
			if t != "" {
				if inTitle {
					if pageTitle.Len() > 0 {
						pageTitle.WriteString(" ")
					}
					pageTitle.WriteString(t)
				} else {
					textLen := buf.Len()
					if textLen > 0 {
						lastByte := buf.String()[textLen-1]
						if lastByte != '\n' {
							buf.WriteByte(' ')
						}
					}
					buf.WriteString(t)
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if utf8.RuneCountInString(buf.String()) > maxTextRunes {
				break
			}
			f(c)
		}

		if n.Type == html.ElementNode {
			if n.Data == "title" {
				inTitle = false
			}
			switch n.Data {
			case "p", "br", "div", "h1", "h2", "h3", "h4", "h5", "h6", "li", "tr", "td":
				buf.WriteByte('\n')
			}
		}
	}
	f(doc)

	text := strings.TrimSpace(buf.String())
	text = truncateText(text, maxTextRunes)
	return pageTitle.String(), text
}

func truncateText(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}
