# Markdown in the Middle

```
 ███╗   ███╗ █████╗ ██████╗ ██╗  ██╗██████╗  ██████╗ ██╗    ██╗███╗   ██╗
 ████╗ ████║██╔══██╗██╔══██╗██║ ██╔╝██╔══██╗██╔═══██╗██║    ██║████╗  ██║
 ██╔████╔██║███████║██████╔╝█████╔╝ ██║  ██║██║   ██║██║ █╗ ██║██╔██╗ ██║
 ██║╚██╔╝██║██╔══██║██╔══██╗██╔═██╗ ██║  ██║██║   ██║██║███╗██║██║╚██╗██║
 ██║ ╚═╝ ██║██║  ██║██║  ██║██║  ██╗██████╔╝╚██████╔╝╚███╔███╔╝██║ ╚████║
 ╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚═════╝  ╚═════╝  ╚══╝╚══╝ ╚═╝  ╚═══╝
          ██╗███╗   ██╗    ████████╗██╗  ██╗███████╗
          ██║████╗  ██║    ╚══██╔══╝██║  ██║██╔════╝
          ██║██╔██╗ ██║       ██║   ███████║█████╗
          ██║██║╚██╗██║       ██║   ██╔══██║██╔══╝
          ██║██║ ╚████║       ██║   ██║  ██║███████╗
          ╚═╝╚═╝  ╚═══╝       ╚═╝   ╚═╝  ╚═╝╚══════╝
 ███╗   ███╗██╗██████╗ ██████╗ ██╗     ███████╗
 ████╗ ████║██║██╔══██╗██╔══██╗██║     ██╔════╝
 ██╔████╔██║██║██║  ██║██║  ██║██║     █████╗
 ██║╚██╔╝██║██║██║  ██║██║  ██║██║     ██╔══╝
 ██║ ╚═╝ ██║██║██████╔╝██████╔╝███████╗███████╗
 ╚═╝     ╚═╝╚═╝╚═════╝ ╚═════╝ ╚══════╝╚══════╝
```

[![GitHub Stars](https://img.shields.io/github/stars/rickcrawford/markdowninthemiddle?style=flat-square&label=Stars&color=blue)](https://github.com/rickcrawford/markdowninthemiddle)
[![MIT License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)
[![Latest Release](https://img.shields.io/github/v/release/rickcrawford/markdowninthemiddle?style=flat-square&color=orange)](https://github.com/rickcrawford/markdowninthemiddle/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/rickcrawford/markdowninthemiddle?style=flat-square)](go.mod)

**A local proxy that converts websites and APIs to clean Markdown.**

Try it in 2 minutes with Docker. Convert HTML pages, JSON APIs, and dynamic sites. Get token counts for LLM budgeting.

By [Rick Crawford](https://github.com/rickcrawford) | [MIT License](LICENSE)

---

## Keywords
`html-to-markdown` • `api-to-markdown` • `content-conversion` • `llm-proxy` • `token-counting` • `markdown-conversion` • `local-proxy` • `claude-integration` • `mcp-tools` • `web-scraping` • `markdown-generation` • `mitm-proxy`

---

## Table of Contents
- [Install](#-install)
- [Get Started](#-get-started-in-2-minutes)
- [What It Does](#-what-it-does)
- [Examples](#-try-these-examples)
- [Features](#-features)
- [Use Cases](#use-cases)
- [Learn More](#-learn-more)

---

## 📦 Install

**Quick install to current directory:**
```bash
curl -fsSL "https://github.com/rickcrawford/markdowninthemiddle/releases/latest/download/install.sh" | bash
```

**Install to `/usr/local/bin/` (system-wide):**
```bash
curl -fsSL "https://github.com/rickcrawford/markdowninthemiddle/releases/latest/download/install.sh" | bash -s /usr/local/bin
```

**Install specific version:**
```bash
curl -fsSL "https://github.com/rickcrawford/markdowninthemiddle/releases/download/v0.1.0/markdowninthemiddle-linux-amd64.tar.gz" | tar -xz
./markdowninthemiddle/markdowninthemiddle --help
```

**Or download manually** from [GitHub Releases](https://github.com/rickcrawford/markdowninthemiddle/releases).

---

## 🚀 Get Started in 2 Minutes

### With Docker (easiest)

```bash
cd docker && docker compose up -d
curl -x http://localhost:8080 https://example.com
```

### With Go

```bash
go build -o mitm .
./mitm
curl -x http://localhost:8080 https://example.com
```

Done! Your proxy is running on `http://localhost:8080`

---

## 💡 What It Does

**HTML pages** → Clean Markdown
**JSON APIs** → Formatted Markdown (with custom templates)
**JavaScript sites** → Renders first, then converts (optional)
**Token counting** → Estimate LLM costs before processing

---

## 🧪 Try These Examples

```bash
# Convert a GitHub user profile
curl -x http://localhost:8080 https://api.github.com/users/octocat

# Get token count for cost estimation
curl -x http://localhost:8080 https://example.com -sD - | grep X-Token-Count

# Use with Claude Desktop (MCP mode)
./mitm mcp --transport chromedp
# Add to Claude settings, then ask Claude to fetch and summarize URLs

# Save all conversions to files
./mitm --output-dir ./markdown
```

---

## 🎯 Features

- 📄 **HTML to Markdown** - All HTML automatically converted
- 📋 **JSON to Markdown** - Custom Mustache templates for API responses
- 🤖 **Claude Integration** - MCP tools for Claude Desktop
- 🔄 **JavaScript Rendering** - Headless Chrome for dynamic sites
- 💬 **Token Counting** - TikToken counts for cost estimation
- 🔐 **HTTPS & MITM** - Self-signed certificates included
- 💾 **Caching** - RFC 7234 compliant local caching
- 🔍 **URL Filtering** - Regex-based domain restrictions

---

## Use Cases

**Private API Documentation**
Convert internal API responses to clean markdown for Claude or other LLMs without exposing raw JSON or sending data through third-party services.

**Local LLM Development**
Run locally with private documents and internal services. No internet dependency, no data leakage. Perfect for offline LLM workflows.

**Claude Desktop Integration**
Add MCP tools so Claude Desktop can fetch, convert, and analyze URLs locally with full markdown formatting and token counting.

**Web Content for LLM Context**
Convert website content to markdown before feeding it to language models. Save token budgets with accurate token counting.

**Internal Network Crawling**
Access internal documentation, staging sites, and self-signed HTTPS endpoints without going through public proxies.

**Multi-Format Content Processing**
Handle HTML pages, JSON APIs, and dynamic JavaScript-rendered sites all through one local proxy interface.

---

## 📚 Learn More

| Guide | For |
|-------|-----|
| [CONFIGURATION.md](./docs/CONFIGURATION.md) | All command-line options |
| [JSON_CONVERSION.md](./docs/JSON_CONVERSION.md) | Using Mustache templates for APIs |
| [MCP_SERVER.md](./docs/MCP_SERVER.md) | Claude Desktop integration |
| [TROUBLESHOOTING.md](./docs/TROUBLESHOOTING.md) | Common issues |
| [DOCKER.md](./docs/DOCKER.md) | Docker deployment |

---

## About

Built with Go, inspired by [Cloudflare's Markdown for Agents](https://blog.cloudflare.com/markdown-for-agents/). Unlike cloud-based markdown conversion services or web scraping libraries, Markdown in the Middle is a **local HTTPS proxy** that runs on your machine. It converts HTML pages and JSON APIs to clean markdown with token counting, enabling secure processing of private content.

### Perfect For:
- 🏢 Internal networks and private services
- 🔒 Staging/testing environments with self-signed certificates
- 🤖 Local LLM deployments and offline AI workflows
- 🔌 Private APIs and proprietary content without external dependencies
- 🛡️ Organizations with data privacy and security requirements

## Author

**Rick Crawford** — [GitHub](https://github.com/rickcrawford) | [Website](https://linkedin.com/in/rickcrawford)

Building tools for AI and APIs.

## License

MIT - See [LICENSE](LICENSE) for details

