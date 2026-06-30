# WebCLI — Web access & shared memory for AI agents

> **⚠ WARNING:** WebCLI handles API keys for AI providers. Never commit your `~/.webcli/` directory or `.env` files to git. The `.gitignore` excludes them by default.

WebCLI is a CLI tool and MCP server that gives AI agents (Claude, DeepSeek, Qwen, etc.)
web search, page fetching, and **persistent shared memory** — so models can learn from each other
across sessions.

```
┌────────────┐    ┌────────────┐    ┌────────────┐
│   Claude   │    │  DeepSeek  │    │    Qwen    │
│  (MCP cli) │    │  (MCP cli) │    │  (MCP cli) │
└─────┬──────┘    └─────┬──────┘    └─────┬──────┘
      │                 │                 │
      └─────────────────┼─────────────────┘
                        │
                 ┌──────▼──────┐
                 │   webcli    │
                 │  MCP :8931  │
                 └──────┬──────┘
                        │
                 ┌──────▼──────┐
                 │  ~/.webcli/ │
                 │  memory.json│
                 └─────────────┘
```

## Features

| Feature | Description |
|---------|-------------|
| **Web search** | DuckDuckGo, Tavily, or Google search — auto-saved as shared knowledge |
| **Web fetch** | Extract text from any page, auto-saved as shared knowledge |
| **AI chat** | Chat with Claude, Gemini, GPT, DeepSeek using **your own API keys** |
| **Shared memory** | Persistent store with categories: `fact`, `discovery`, `preference`, `instruction` |
| **Cross-model sharing** | All models reading/writing the same memory learn from each other |
| **MCP server** | stdio (local agents) and SSE (remote agents) modes |
| **Context injection** | `memory_context` + auto-injected into chat system prompt |
| **Health endpoint** | `GET /health` for monitoring |
| **Rate limiting** | Token bucket — 30 req/min default |
| **Backup & restore** | Automated memory backups |
| **Config file** | YAML config with env var & CLI flag overrides |

## Quick Start

```bash
# Option A: One-liner (macOS/Linux)
curl -fsSL https://raw.githubusercontent.com/yenya/webcli/main/install.sh | bash

# Option B: npx (no install)
npx webcli --help

# Option C: Go
go install github.com/yenya/webcli@latest
```

▶️ **[Full installation guide → INSTALL.md](INSTALL.md)** — covers shell, npx, Go, Docker, source build, and prebuilt binaries.

```bash
# After install:
webcli config init              # create config
webcli search "AI news"         # search web (free, no key)
webcli chat "hello world"       # chat with AI (needs API key)
webcli serve --transport sse    # start MCP server
```

### Set up your provider

**Local model via Ollama (no API key needed):**
```bash
# Install Ollama, pull a model, and run:
ollama pull qwen2.5
ollama serve
# webcli auto-detects Ollama on http://localhost:11434
```

**Local model via OpenAI-compatible endpoint:**
```bash
export WEBCLI_LOCAL_ENDPOINT="http://localhost:8000/v1/chat/completions"
export WEBCLI_LOCAL_MODEL="qwen2.5-7b-instruct"
```

**Cloud API keys (pick one or more):**
```bash
export ANTHROPIC_API_KEY="sk-ant-..."   # Claude
export GOOGLE_API_KEY="AIza..."         # Gemini
export OPENAI_API_KEY="sk-proj-..."     # GPT
export DEEPSEEK_API_KEY="sk-..."        # DeepSeek

# For Tavily search (optional, replaces DuckDuckGo):
export WEBCLI_SEARCH_API_KEY="tvly-..."
```

## Usage

### CLI Commands

```bash
# Web search
webcli search "latest AI developments"

# Fetch a page
webcli fetch https://example.com

# AI Chat — auto-detects provider
webcli chat "Explain quantum computing"

# Explicit provider
webcli chat --provider ollama "Explain with Qwen"
webcli chat --provider local "Using my own endpoint"
webcli chat --provider google "What is the speed of light?"
webcli chat --provider deepseek --model deepseek-reasoner "Solve this"
webcli chat --context "Tell me about Go"   # injects shared memory

# Shared memory
webcli memory remember "fact:go" "Go is compiled by Google" -c fact -t "programming,language"
webcli memory recall "fact:go"
webcli memory forget "fact:go"
webcli memory context -q "programming"

# Backup
webcli backup create
webcli backup list
webcli backup restore ~/.webcli/backups/memory_20260630_120000.json

# Config
webcli config init          # create ~/.webcli/config.yaml
webcli config show          # view current config
webcli config validate      # validate config

# MCP server
webcli serve                # stdio mode (for local AI IDEs)
webcli serve --transport sse  # SSE mode (for remote agents)
```

### MCP Server

**Stdio mode** — used by AI IDEs that spawn the process directly:

```bash
webcli serve
```

**SSE mode** — used by remote agents over HTTP:

```bash
webcli serve --transport sse
```

Health check: `curl http://localhost:8931/health`

### Connecting AI Models

#### Claude (via MCP client)

In your MCP client config:

```json
{
  "mcpServers": {
    "webcli": {
      "command": "webcli",
      "args": ["serve"],
      "env": {}
    }
  }
}
```

#### Any MCP-compatible agent (DeepSeek, Qwen, etc.)

```json
{
  "mcpServers": {
    "webcli": {
      "url": "http://your-server:8931/sse"
    }
  }
}
```

### MCP Tools

Once connected, agents have access to these tools:

| Tool | Description |
|------|-------------|
| `web_search` | Search the web (auto-saves as discovery) |
| `web_fetch` | Fetch a page (auto-saves as discovery) |
| `chat` | Chat with any AI model (Claude, Gemini, GPT, DeepSeek, or local via Ollama/OpenAI-compatible endpoint) |
| `memory_save` | Save knowledge (key, value, category, tags) |
| `memory_recall` | Retrieve a specific memory by key |
| `memory_search` | Search across all memories |
| `memory_context` | Get relevant context from other models |
| `memory_list` | List all shared knowledge |
| `memory_forget` | Delete a memory |

### Cross-Model Learning Flow

```
1. Claude discovers:  web_search("Go vs Python performance")
   → auto-saves as discovery → stored in memory.json

2. DeepSeek starts task:
   → calls memory_context("programming languages")
   → gets Claude's discovery → learns from it

3. Qwen (local) asks:
   → calls memory_context("performance")
   → gets both Claude + DeepSeek's knowledge

4. Any model can contribute:
   → memory_save("fact:python", "Python is interpreted", "fact", ["language"])

5. Another model can chat using any provider:
   → chat(provider="anthropic", prompt="Compare Go and Python")
   → shared memory context is auto-injected into the system prompt
```

## Configuration

### Config file: `~/.webcli/config.yaml`

```yaml
port: 8931
host: "127.0.0.1"
user_agent: "Mozilla/5.0 (compatible; WebCLI/1.0)"
search_timeout: 10s
fetch_timeout: 30s
max_results: 10
log_level: "info"
log_format: "text"
rate_limit:
  requests_per_minute: 30
  burst: 5
store:
  memory_file: ""
  backup_dir: ""
search:
  provider: "duckduckgo"
  max_retries: 2
fetch:
  max_body_size: 5242880
  max_text_len: 10000
```

### Configuration file

`webcli` loads config from `~/.webcli/config.yaml` (or `config.json`). You can also use `config.yml` or `~/.config/webcli/config.{yaml,json}`.

```bash
# Create default YAML config
webcli config init

# Create default JSON config
webcli config init --json

# View current config (YAML)
webcli config show

# View current config (JSON)
webcli config show --format json

# Validate config
webcli config validate
```

Edit `~/.webcli/config.json` to configure your local server:

```json
{
  "providers": {
    "ollama_url": "http://localhost:11434",
    "ollama_model": "qwen2.5",
    "local_endpoint": "http://localhost:8000/v1/chat/completions",
    "local_model": "qwen2.5-7b-instruct"
  }
}
```

### Environment variables

| Variable | Overrides |
|----------|-----------|
| `WEBCLI_PORT` | `port` |
| `WEBCLI_HOST` | `host` |
| `WEBCLI_USER_AGENT` | `user_agent` |
| `WEBCLI_SEARCH_TIMEOUT` | `search_timeout` |
| `WEBCLI_FETCH_TIMEOUT` | `fetch_timeout` |
| `WEBCLI_MAX_RESULTS` | `max_results` |
| `WEBCLI_LOG_LEVEL` | `log_level` |
| `WEBCLI_LOG_FORMAT` | `log_format` |
| `WEBCLI_RATE_LIMIT` | `rate_limit.requests_per_minute` |
| `WEBCLI_MEMORY_FILE` | `store.memory_file` |
| `WEBCLI_BACKUP_DIR` | `store.backup_dir` |
| `WEBCLI_SEARCH_API_KEY` | `providers.search_api_key` (Tavily) |
| `WEBCLI_GOOGLE_CX` | `providers.google_cx` (Google CSE) |
| **AI Provider Keys** | |
| `ANTHROPIC_API_KEY` or `WEBCLI_ANTHROPIC_KEY` | Claude |
| `GOOGLE_API_KEY` or `WEBCLI_GOOGLE_KEY` | Gemini |
| `OPENAI_API_KEY` or `WEBCLI_OPENAI_KEY` | GPT |
| `DEEPSEEK_API_KEY` or `WEBCLI_DEEPSEEK_KEY` | DeepSeek |
| `OLLAMA_URL` or `WEBCLI_OLLAMA_URL` | Ollama endpoint (default: http://localhost:11434) |
| `WEBCLI_OLLAMA_MODEL` | Default Ollama model (default: qwen2.5) |
| `WEBCLI_LOCAL_ENDPOINT` | OpenAI-compatible local endpoint URL |
| `WEBCLI_LOCAL_MODEL` | Model name for local endpoint |
| `WEBCLI_LOCAL_API_KEY` | Optional API key for local endpoint |

> **Security:** Cloud API keys should be set via environment variables, never written to the config file. Local provider config (Ollama URL, endpoint) is safe to persist in config.

## Docker

```bash
docker compose up -d
```

▶️ **[Full Docker setup → INSTALL.md](INSTALL.md#4-docker)**

## Backup

```bash
webcli backup create              # timestamped backup
webcli backup list                # list backups
webcli backup restore <file>      # restore from backup
```

## Architecture

```
webcli/
├── main.go                 # Entry point
├── cmd/                    # CLI commands (cobra)
│   ├── root.go
│   ├── search.go
│   ├── fetch.go
│   ├── serve.go
│   ├── config.go
│   ├── backup.go
│   └── memory.go
├── internal/
│   ├── config/             # YAML config with validation
│   ├── log/                # Structured logging (text + JSON)
│   ├── server/             # MCP server (stdio + SSE)
│   ├── store/              # Persistent key-value store
│   └── web/                # Web search + fetch with rate limiting
├── npm/                    # npm wrapper package (for npx)
├── INSTALL.md              # Full installation guide
├── Makefile                # Build automation
├── Dockerfile
├── docker-compose.yml
├── install.sh              # Shell one-liner installer
├── README.md
└── .gitignore
```

## Production Considerations

- **Rate limiting**: Default 30 req/min prevents IP blocks
- **Logging**: Use `--log-format json` for structured logs
- **Backup**: Schedule `webcli backup create` via cron
- **Monitoring**: Health endpoint at `/health` returns memory count and uptime
- **Docker**: Healthcheck configured, auto-restarts on failure
- **Config validation**: `webcli config validate` before deploying

## Development

```bash
git clone https://github.com/yenya/webcli.git
cd webcli
go mod download
go build -o webcli .
./webcli --help
```
