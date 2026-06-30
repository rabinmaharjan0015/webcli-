# Security Policy

## API Keys

WebCLI handles API keys for AI providers (Anthropic, Google, OpenAI, DeepSeek) and optional search providers (Tavily).

- **Never commit API keys to git.** The `.gitignore` excludes `~/.webcli/` and `.env` files.
- Cloud API keys should be set via environment variables, not stored in config files.
- If you use `~/.webcli/.env`, it has `0600` permissions and is auto-loaded — but treat it like any credential file.
- `webcli config show` masks all keys: `sk-a****f4`.

## Reporting a Vulnerability

Open an issue at https://github.com/yenya/webcli/issues
