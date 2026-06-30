# Installing WebCLI

WebCLI provides **6 ways to install** — pick whichever fits your workflow.

---

## 1. Shell one-liner (macOS / Linux) ⭐

**Quickest way.** Auto-detects your OS/arch and downloads the right binary.

```bash
curl -fsSL https://raw.githubusercontent.com/yenya/webcli/main/install.sh | bash
```

Installs to `~/.local/bin/webcli`. Make sure that directory is in your `$PATH`:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Add that line to your `~/.zshrc` or `~/.bashrc` to make it permanent.

---

## 2. npx / npm (macOS / Linux) ⭐

**No install needed** — runs directly via npx. The binary is downloaded on first use and cached.

```bash
# Run directly (no install)
npx webcli --help
npx webcli search "latest AI news"
npx webcli serve --transport sse

# Global install (optional)
npm install -g webcli
webcli --help
```

**Requirements:** Node.js >= 18

---

## 3. Go install

**For Go developers.** Installs the latest version from source.

```bash
go install github.com/yenya/webcli@latest
```

Make sure `$GOPATH/bin` is in your `$PATH`:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

**Requirements:** Go 1.25+

---

## 4. Docker

**For containerized deployments.** Includes healthcheck and auto-restart.

```bash
# Clone and run
git clone https://github.com/yenya/webcli.git
cd webcli
docker compose up -d

# Or build manually
docker build -t webcli .
docker run -d \
  --name webcli \
  -p 8931:8931 \
  -v webcli_data:/home/webcli/.webcli \
  webcli serve --transport sse
```

The SSE server will be available at `http://localhost:8931`.

Verify:

```bash
curl http://localhost:8931/health
```

**Persistent volume:** `webcli_data` maps to `/home/webcli/.webcli` inside the container, so your shared memory survives restarts.

---

## 5. Build from source

**For contributors and custom builds.**

```bash
git clone https://github.com/yenya/webcli.git
cd webcli

# Build
go build -o webcli .

# Install
mv webcli /usr/local/bin/
webcli --help
```

Or use the Makefile:

```bash
make build     # builds binary
make install   # go install
make test      # run tests
make lint      # vet
```

---

## 6. Prebuilt binaries (manual download)

Download the archive for your platform, extract, and place the binary in your `$PATH`.

| Platform | Download |
|----------|----------|
| macOS (Intel) | [`webcli_darwin_amd64.tar.gz`](https://github.com/yenya/webcli/releases/latest/download/webcli_darwin_amd64.tar.gz) |
| macOS (Apple Silicon) | [`webcli_darwin_arm64.tar.gz`](https://github.com/yenya/webcli/releases/latest/download/webcli_darwin_arm64.tar.gz) |
| Linux (x86_64) | [`webcli_linux_amd64.tar.gz`](https://github.com/yenya/webcli/releases/latest/download/webcli_linux_amd64.tar.gz) |
| Linux (ARM64) | [`webcli_linux_arm64.tar.gz`](https://github.com/yenya/webcli/releases/latest/download/webcli_linux_arm64.tar.gz) |

```bash
# Example: macOS Apple Silicon
curl -LO https://github.com/yenya/webcli/releases/latest/download/webcli_darwin_arm64.tar.gz
tar xzf webcli_darwin_arm64.tar.gz
chmod +x webcli
sudo mv webcli /usr/local/bin/
```

---

## Verify installation

```bash
webcli --help
```

You should see the help output with commands: `search`, `fetch`, `serve`, `memory`, `config`, `backup`.

```bash
# Create default config
webcli config init

# Quick test
webcli search "hello world"

# Start server
webcli serve --transport sse
```

---

## Post-install steps

### 1. Initialize config

```bash
webcli config init
```

Creates `~/.webcli/config.yaml` with defaults. Customize as needed:

```bash
webcli config show       # view current config
webcli config validate   # validate
```

### 2. Configure MCP for your AI agents

**Claude / Cline / Cursor** — add to your MCP config:

```json
{
  "mcpServers": {
    "webcli": {
      "command": "webcli",
      "args": ["serve"]
    }
  }
}
```

**Any remote agent** — point to the SSE endpoint:

```json
{
  "mcpServers": {
    "webcli": {
      "url": "http://your-server:8931/sse"
    }
  }
}
```

### 3. Schedule backups (optional)

```bash
# Add to crontab — daily backup at 3am
0 3 * * * /usr/local/bin/webcli backup create
```

---

## Troubleshooting

| Problem | Solution |
|---------|----------|
| `command not found: webcli` | The install directory is not in `$PATH`. Add `export PATH="$HOME/.local/bin:$PATH"` to your shell config. |
| `npx: command not found` | Install Node.js from [nodejs.org](https://nodejs.org/) |
| `go: command not found` | Install Go from [go.dev](https://go.dev/dl/) |
| `port 8931 already in use` | Use `--port` flag: `webcli serve --port 8932` |
| Permission denied | Try `sudo` or install to `~/.local/bin` instead |
| `too many requests` | Rate limit hit. Wait or adjust `rate_limit` in config |
| Search returns empty | DuckDuckGo may be blocking. Try again later or reduce request rate |

---

## Uninstall

```bash
# Remove binary
rm "$(which webcli)"
rm -rf ~/.webcli

# If installed via npm
npm uninstall -g webcli

# If installed via Go
rm "$(go env GOPATH)/bin/webcli"
```
