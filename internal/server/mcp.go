package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/yenya/webcli/internal/ai"
	"github.com/yenya/webcli/internal/config"
	"github.com/yenya/webcli/internal/log"
	"github.com/yenya/webcli/internal/store"
	"github.com/yenya/webcli/internal/web"
)

const version = "1.0.0"

func RunMCPServer(cfg *config.Config, transport string) error {
	s := server.NewMCPServer(
		"webcli",
		version,
	)

	rateLimiter := web.NewRateLimiter(cfg.RateLimit.RequestsPerMinute, cfg.RateLimit.Burst)
	searcher, err := web.NewSearchProvider(
		cfg.Search.Provider,
		cfg.Providers.SearchAPIKey,
		cfg.Providers.GoogleCX,
		rateLimiter,
		cfg.UserAgent,
	)
	if err != nil {
		return fmt.Errorf("search provider: %w", err)
	}
	fetcher := web.NewFetcher(cfg.FetchTimeout.ToDuration(), cfg.UserAgent, rateLimiter)

	memStore, err := store.New(cfg.Store.MemoryFile)
	if err != nil {
		return fmt.Errorf("init memory store: %w", err)
	}

	log.Info("Memory store loaded (%d items)", memStore.Count())

	registerWebTools(s, searcher, fetcher, memStore, cfg.MaxResults)
	registerMemoryTools(s, memStore)
	registerChatTools(s, cfg, memStore)

	switch transport {
	case "sse":
		addr := cfg.Addr()
		log.Info("Starting MCP SSE server on %s", addr)

		mux := http.NewServeMux()

		sseServer := server.NewSSEServer(s)
		mux.Handle("/sse", sseServer)
		mux.Handle("/message", sseServer)
		mux.HandleFunc("/health", healthHandler(cfg, memStore))
		mux.Handle("/config.json", configHandler(cfg))
		mux.HandleFunc("/", landingHandler(cfg.Addr()))

		httpServer := &http.Server{
			Addr:    addr,
			Handler: withLogging(mux),
		}

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-quit
			log.Info("Shutting down MCP server...")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			httpServer.Shutdown(ctx)
		}()

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
		return nil

	default:
		log.Info("Starting MCP stdio server")
		return server.ServeStdio(s)
	}
}

func configHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cfg)
	}
}

func landingHandler(addr string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>WebCLI</title>
<style>
  *{margin:0;padding:0;box-sizing:border-box}
  body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:#212121;color:#ececec;display:flex;height:100vh;align-items:center;justify-content:center}
  .container{max-width:640px;width:100%%;padding:32px;text-align:center}
  h1{font-size:28px;font-weight:700;margin-bottom:4px;background:linear-gradient(135deg,#19c37d,#10a37f);-webkit-background-clip:text;-webkit-text-fill-color:transparent}
  .sub{color:#8e8ea0;font-size:14px;margin-bottom:32px}
  .grid{display:grid;grid-template-columns:1fr 1fr;gap:12px;margin-bottom:32px;text-align:left}
  .card{background:#2f2f2f;border-radius:10px;padding:16px;transition:background .2s;cursor:default}
  .card:hover{background:#3a3a3a}
  .card .icon{font-size:20px;margin-bottom:6px}
  .card .name{font-size:14px;font-weight:600;margin-bottom:4px}
  .card .desc{font-size:12px;color:#8e8ea0;line-height:1.5}
  .code{display:inline-block;background:#2f2f2f;color:#19c37d;padding:2px 8px;border-radius:4px;font-family:monospace;font-size:13px}
  .endpoints{margin-top:24px;text-align:left}
  .endpoints h3{font-size:13px;color:#8e8ea0;text-transform:uppercase;letter-spacing:.5px;margin-bottom:12px}
  .ep{display:flex;align-items:center;gap:12px;padding:10px 12px;background:#2f2f2f;border-radius:8px;margin-bottom:6px}
  .ep .route{font-family:monospace;font-size:13px;color:#19c37d;min-width:100px}
  .ep .desc{font-size:13px;color:#b4b4b4}
  .footer{margin-top:32px;font-size:12px;color:#5d5d5d}
  @media(max-width:500px){.grid{grid-template-columns:1fr}}
</style>
</head>
<body>
<div class="container">
  <h1>WebCLI</h1>
  <div class="sub">Web access &amp; shared memory for AI agents</div>

  <div class="grid">
    <div class="card">
      <div class="icon">&#x1F50D;</div>
      <div class="name">Web Search</div>
      <div class="desc">Search the web via DuckDuckGo. Free, no API key needed.</div>
    </div>
    <div class="card">
      <div class="icon">&#x1F4E1;</div>
      <div class="name">Fetch Pages</div>
      <div class="desc">Extract text content from any URL. Auto-saves to memory.</div>
    </div>
    <div class="card">
      <div class="icon">&#x1F4AC;</div>
      <div class="name">AI Chat</div>
      <div class="desc">Chat with Claude, Gemini, GPT, DeepSeek, or local models.</div>
    </div>
    <div class="card">
      <div class="icon">&#x1F9E0;</div>
      <div class="name">Shared Memory</div>
      <div class="desc">Persistent memory across all models. Learn from each other.</div>
    </div>
  </div>

  <div class="endpoints">
    <h3>Server Endpoints</h3>
    <div class="ep"><span class="route">/sse</span><span class="desc">MCP Server-Sent Events stream for AI agents</span></div>
    <div class="ep"><span class="route">/message</span><span class="desc">Send JSON-RPC messages to call tools</span></div>
    <div class="ep"><span class="route">/health</span><span class="desc">Server status, version, memory count</span></div>
    <div class="ep"><span class="route">/config.json</span><span class="desc">Server configuration with provider status</span></div>
  </div>

  <div class="footer">
    Running on %s &middot; <span class="code">webcli serve</span>
  </div>
</div>
</body>
</html>`, addr)
	}
}

func healthHandler(cfg *config.Config, memStore *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "ok",
			"app":        "webcli",
			"version":    version,
			"uptime":     time.Now().Unix(),
			"memories":   memStore.Count(),
		})
	}
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Debug("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func registerWebTools(s *server.MCPServer, searcher web.SearchProvider, fetcher *web.Fetcher, memStore *store.Store, maxResults int) {
	searchTool := mcp.NewTool("web_search",
		mcp.WithDescription("Search the web. The result is automatically saved as a discovery in shared memory so other models can learn from it."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("The search query"),
		),
	)

	fetchTool := mcp.NewTool("web_fetch",
		mcp.WithDescription("Fetch and extract text content from a web page URL. The result is automatically saved as a discovery in shared memory so other models can learn from it."),
		mcp.WithString("url",
			mcp.Required(),
			mcp.Description("The URL to fetch"),
		),
	)

	s.AddTool(searchTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments: expected object"), nil
		}

		query, ok := args["query"].(string)
		if !ok || query == "" {
			return mcp.NewToolResultError("query is required and must be a non-empty string"), nil
		}

		log.Info("web_search: %s", query)
		results, err := searcher.Search(query, maxResults)
		if err != nil {
			log.Error("web_search failed: %s", err.Error())
			return mcp.NewToolResultError(fmt.Sprintf("search failed: %s", err.Error())), nil
		}

		if len(results) > 0 {
			var summaries []string
			for _, r := range results[:min(3, len(results))] {
				summaries = append(summaries, fmt.Sprintf("- %s: %s", r.Title, r.Snippet))
			}
			summary := fmt.Sprintf("Web search results for \"%s\":\n%s", query, strings.Join(summaries, "\n"))
			discoveryKey := fmt.Sprintf("search:%s", query)
			tags := extractTags(query)
			memStore.SaveDiscovery(discoveryKey, summary, "web_search", append(tags, query))
			log.Debug("Saved discovery: %s", discoveryKey)
		}

		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("encode results: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	})

	s.AddTool(fetchTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments: expected object"), nil
		}

		rawURL, ok := args["url"].(string)
		if !ok || rawURL == "" {
			return mcp.NewToolResultError("url is required and must be a non-empty string"), nil
		}

		log.Info("web_fetch: %s", rawURL)
		result, err := fetcher.Fetch(rawURL)
		if err != nil {
			log.Error("web_fetch failed: %s", err.Error())
			return mcp.NewToolResultError(fmt.Sprintf("fetch failed: %s", err.Error())), nil
		}

		if result.Content != "" {
			content := fmt.Sprintf("Title: %s\nURL: %s\n\n%s", result.Title, result.URL, truncateString(result.Content, 2000))
			discoveryKey := fmt.Sprintf("fetch:%s", result.URL)
			memStore.SaveDiscovery(discoveryKey, content, "web_fetch", []string{result.Title, result.URL})
			log.Debug("Saved discovery: %s", discoveryKey)
		}

		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("encode result: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	})
}

func registerMemoryTools(s *server.MCPServer, memStore *store.Store) {
	saveTool := mcp.NewTool("memory_save",
		mcp.WithDescription("Save knowledge to persistent shared memory. All models (Claude, DeepSeek, Qwen, etc.) connected to this server can access this. Use for facts, preferences, discoveries, and instructions."),
		mcp.WithString("key",
			mcp.Required(),
			mcp.Description("Unique key for the memory (use namespaced keys like 'user:name', 'fact:topic', 'pref:setting')"),
		),
		mcp.WithString("value",
			mcp.Required(),
			mcp.Description("The content to remember"),
		),
		mcp.WithString("category",
			mcp.Description("Category: fact (default), discovery, preference, instruction"),
		),
		mcp.WithString("tags",
			mcp.Description("Comma-separated tags for better searchability"),
		),
	)

	recallTool := mcp.NewTool("memory_recall",
		mcp.WithDescription("Retrieve a specific memory by its key. Returns what another model previously saved."),
		mcp.WithString("key",
			mcp.Required(),
			mcp.Description("Key of the memory to retrieve"),
		),
	)

	forgetTool := mcp.NewTool("memory_forget",
		mcp.WithDescription("Delete a memory by its key."),
		mcp.WithString("key",
			mcp.Required(),
			mcp.Description("Key of the memory to delete"),
		),
	)

	listTool := mcp.NewTool("memory_list",
		mcp.WithDescription("List all memories across all categories. Discover what other models have learned."),
	)

	searchTool := mcp.NewTool("memory_search",
		mcp.WithDescription("Search across all shared memories by content. Find what other models know about a topic."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query"),
		),
	)

	contextTool := mcp.NewTool("memory_context",
		mcp.WithDescription("Get relevant shared knowledge from other models about a topic. Use at the start of a task to learn from what other models have already discovered."),
		mcp.WithString("topic",
			mcp.Description("Topic to get context about. Leave empty to get all shared knowledge."),
		),
	)

	s.AddTool(saveTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments: expected object"), nil
		}

		key, _ := args["key"].(string)
		value, _ := args["value"].(string)

		if key == "" {
			return mcp.NewToolResultError("key is required"), nil
		}
		if value == "" {
			return mcp.NewToolResultError("value is required"), nil
		}

		cat := store.CategoryFact
		if c, ok := args["category"].(string); ok {
			switch store.Category(c) {
			case store.CategoryDiscovery, store.CategoryPreference, store.CategoryInstruction:
				cat = store.Category(c)
			}
		}

		var tags []string
		if t, ok := args["tags"].(string); ok && t != "" {
			for _, tag := range strings.Split(t, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					tags = append(tags, tag)
				}
			}
		}

		item, err := memStore.Save(key, value, store.WithCategory(cat), store.WithTags(tags))
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("save failed: %s", err.Error())), nil
		}

		log.Info("memory_save: %s [%s]", key, cat)
		data, err := json.MarshalIndent(item, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("encode: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	})

	s.AddTool(recallTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments: expected object"), nil
		}

		key, _ := args["key"].(string)
		if key == "" {
			return mcp.NewToolResultError("key is required"), nil
		}

		item, err := memStore.Get(key)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("recall failed: %s", err.Error())), nil
		}

		data, err := json.MarshalIndent(item, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("encode: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	})

	s.AddTool(forgetTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments: expected object"), nil
		}

		key, _ := args["key"].(string)
		if key == "" {
			return mcp.NewToolResultError("key is required"), nil
		}

		if err := memStore.Delete(key); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("forget failed: %s", err.Error())), nil
		}

		log.Info("memory_forget: %s", key)
		return mcp.NewToolResultText(fmt.Sprintf("memory '%s' deleted", key)), nil
	})

	s.AddTool(listTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		items := memStore.List()
		if items == nil {
			items = []store.MemoryItem{}
		}

		data, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("encode: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	})

	s.AddTool(searchTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments: expected object"), nil
		}

		query, _ := args["query"].(string)
		if query == "" {
			return mcp.NewToolResultError("query is required"), nil
		}

		items := memStore.Search(query)
		if items == nil {
			items = []store.MemoryItem{}
		}

		data, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("encode: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	})

	s.AddTool(contextTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments: expected object"), nil
		}

		topic, _ := args["topic"].(string)
		context := memStore.GetContext(topic)

		if context == "" {
			if topic != "" {
				return mcp.NewToolResultText(fmt.Sprintf("No shared knowledge found about \"%s\" yet.", topic)), nil
			}
			return mcp.NewToolResultText("No shared knowledge stored yet. Use web_search, web_fetch, or memory_save to start building shared knowledge."), nil
		}

		return mcp.NewToolResultText(context), nil
	})
}

func registerChatTools(s *server.MCPServer, cfg *config.Config, memStore *store.Store) {
	chatTool := mcp.NewTool("chat",
		mcp.WithDescription("Send a prompt to an AI model (Claude, Gemini, GPT, DeepSeek, or local models like Qwen via Ollama). Configure keys via environment variables."),
		mcp.WithString("provider",
			mcp.Description("Provider: ollama, local, anthropic, google, openai, deepseek (auto-detected if omitted)"),
		),
		mcp.WithString("model",
			mcp.Description("Model name (e.g. claude-sonnet-4-20250514, gemini-2.5-flash, gpt-4o, deepseek-chat, qwen2.5). Defaults per provider."),
		),
		mcp.WithString("prompt",
			mcp.Required(),
			mcp.Description("The prompt to send"),
		),
		mcp.WithString("system",
			mcp.Description("System prompt (shared memory context is auto-injected)"),
		),
	)

	s.AddTool(chatTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, ok := request.Params.Arguments.(map[string]interface{})
		if !ok {
			return mcp.NewToolResultError("invalid arguments: expected object"), nil
		}

		prompt, _ := args["prompt"].(string)
		if prompt == "" {
			return mcp.NewToolResultError("prompt is required and must be a non-empty string"), nil
		}

		providerName, _ := args["provider"].(string)
		modelName, _ := args["model"].(string)
		systemPrompt, _ := args["system"].(string)

		providerCfg := ai.ProviderConfig{
			AnthropicKey:  cfg.Providers.AnthropicKey,
			GoogleKey:     cfg.Providers.GoogleKey,
			OpenAIKey:     cfg.Providers.OpenAIKey,
			DeepSeekKey:   cfg.Providers.DeepSeekKey,
			OllamaURL:     cfg.Providers.OllamaURL,
			OllamaModel:   cfg.Providers.OllamaModel,
			LocalEndpoint: cfg.Providers.LocalEndpoint,
			LocalModel:    cfg.Providers.LocalModel,
			LocalAPIKey:   cfg.Providers.LocalAPIKey,
		}

		// Auto-detect provider
		if providerName == "" {
			providerName = ai.DetectProvider(providerCfg)
			if providerName == "" {
				return mcp.NewToolResultError(
					"no provider available. Set an API key or run Ollama locally.\n" +
						"  ollama (local)  → ensure Ollama is running on http://localhost:11434\n" +
						"  local           → set WEBCLI_LOCAL_ENDPOINT and WEBCLI_LOCAL_MODEL\n" +
						"  anthropic       → set ANTHROPIC_API_KEY\n" +
						"  google          → set GOOGLE_API_KEY\n" +
						"  openai          → set OPENAI_API_KEY\n" +
						"  deepseek        → set DEEPSEEK_API_KEY",
				), nil
			}
		}

		provider, err := ai.NewProvider(providerName, providerCfg)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("init provider: %s", err.Error())), nil
		}

		// Auto-inject shared memory context
		if memStore != nil {
			context := memStore.GetContext(prompt)
			if context != "" {
				if systemPrompt != "" {
					systemPrompt += "\n\n"
				}
				systemPrompt += context
			}
		}

		log.Info("chat: provider=%s model=%s", providerName, modelName)
		resp, err := provider.Chat(ai.ChatRequest{
			Model:  modelName,
			System: systemPrompt,
			Messages: []ai.Message{
				{Role: "user", Content: prompt},
			},
		})
		if err != nil {
			log.Error("chat failed: %s", err.Error())
			return mcp.NewToolResultError(fmt.Sprintf("chat failed: %s", err.Error())), nil
		}

		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("encode: %s", err.Error())), nil
		}

		return mcp.NewToolResultText(string(data)), nil
	})
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
