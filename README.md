<div align="center">

# ttl

### Your Personal Knowledge Archive, Powered by AI

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![MCP](https://img.shields.io/badge/MCP-Supported-green.svg)](https://modelcontextprotocol.io/)

[简体中文](README.zh-CN.md) | [日本語](README.ja.md) | [Español](README.es.md) | [Français](README.fr.md) | [Português](README.pt.md)

---

*A lightweight CLI tool for personal data management. Store anything as key-value pairs, search instantly, and let AI manage it all with natural language.*

</div>

---

## 📖 The Story

Every developer has been there:

> "Wait, what was that Docker command I used last month?"
> "Where did my colleague share that config file?"
> "I know I read an article about this... but can't find it anywhere."

We store knowledge everywhere — browser tabs, Slack messages, emails, bookmark folders, notes apps. When we actually need it, we waste time digging through endless tabs and scrolling through chat history.

**This was my pain too.**

So I built **ttl**.

The name comes from "Time to Live" — but with a different meaning. Instead of expiring, it's about giving your knowledge a **time to live** forever.

- Store everything in one place as key-value pairs
- Tag it for easy organization
- Search instantly by keyword
- Use AI to manage it with natural language

No more searching through old emails or scrolling through Slack history. Just `ttl get <keyword>` and you have it.

**ttl is your personal knowledge archive — everything you need, when you need it.**

---

## ✨ Features

| Feature | Description |
|---------|-------------|
| 🗄️ **Local KV Storage** | Fast, zero-config embedded database (bbolt) |
| 🏷️ **Tag System** | Organize resources with flexible, searchable tags |
| 🔍 **Fuzzy Search** | Find what you need instantly across keys and tags |
| 🤖 **AI Agent** | Manage data with natural language (OpenAI / DeepSeek / Ollama) |
| 📝 **Work Log** | Track daily work, generate weekly/monthly reports with AI |
| 🔗 **MCP Protocol** | Let AI tools (Claude Code, Cursor) operate your data directly |
| ☁️ **Cloud Sync** | Self-host multi-tenant server with per-user data isolation |
| 🔒 **Privacy First** | AI never sends your values — only keys and tags |
| 🚀 **Smart Open** | Open URLs and files with system default programs |
| 📤 **Export** | Export data as JSON or CSV |

---

## 🚀 Quick Start

### Installation

```bash
# Build from source
go build -o ttl .
sudo mv ttl /usr/local/bin/

# Or use the install script
bash install.sh
```

### Basic Usage

```bash
# Add a resource
ttl add my-link https://example.com

# Add with tags
ttl add docker-cmd "docker run -d -p 8080:80 nginx"
ttl tag docker-cmd dev ops

# Search resources
ttl get docker

# Open in browser
ttl open my-link

# Delete
ttl del old-key
```

---

## 🤖 AI Agent

One command replaces ten. Just describe what you want in natural language.

```bash
# Configure AI first
ttl config ai

# Then use natural language
ttl ai "save this nginx config: port 8080 changed to 443"
ttl ai "find all my docker related resources"
ttl ai "open the sugar dashboard"
ttl ai "what did I store recently?"
ttl ai "tag nginx-config with ops and deploy"
```

### Privacy by Design

The AI Agent only sends **resource keys and tags** to the LLM. Your values — which may contain passwords, tokens, internal URLs, or sensitive data — never leave your machine. Only work log content is sent in full for summarization.

### Compatible Models

- OpenAI (GPT-4, GPT-4o, GPT-4o-mini)
- DeepSeek
- Moonshot
- Ollama (local models)
- Any OpenAI Chat Completions API compatible model

---

## 📝 Work Log

Track your daily work and let AI generate reports.

```bash
# Write a log entry
ttl log write "Finished user module refactoring" --tags "projectA,dev"

# View logs
ttl log list                    # Today's logs
ttl log list --range week       # This week
ttl log list --range month      # This month

# AI-powered weekly report
ttl ai "summarize my work logs this week"
ttl ai "generate a weekly report in markdown format"
```

---

## 🔗 MCP Protocol Integration

Let AI tools operate your data via [Model Context Protocol](https://modelcontextprotocol.io/).

### Start MCP Server

```bash
ttl mcp
```

### Claude Code Integration

Add to `~/.claude/claude_code_config.json`:

```json
{
  "mcpServers": {
    "ttl": {
      "command": "ttl",
      "args": ["mcp"]
    }
  }
}
```

Now Claude Code can read, write, and manage your resources directly!

**Available MCP Tools:**
- `ttl_get` — Retrieve resources
- `ttl_add` — Add new resources
- `ttl_update` — Update existing resources
- `ttl_delete` — Delete resources
- `ttl_tag` — Add tags
- `ttl_dtag` — Remove tags
- `ttl_open` — Open resources
- `ttl_rename` — Rename resources

---

## ☁️ Cloud Server & Sync

### Start Your Server

```bash
# Create a user
ttl server user add alice

# Start multi-tenant server
ttl server start --port 8080
```

### Sync Your Data

```bash
# Configure remote server
ttl config
# Edit the server section with your endpoint and API key

# Sync local <-> remote
ttl sync
```

**Architecture:**
- Multi-tenant design with per-user isolated databases
- API Key authentication
- REST API for programmatic access
- MCP HTTP endpoint at `/mcp` for AI clients

---

## ⚙️ Configuration

Config file: `~/.ttl/ttl.ini`

```ini
[default]
db_path = ~/.ttl/data.db

[ai]
api_key   = your-api-key-here
base_url  = https://api.openai.com
model     = gpt-4o-mini
timeout   = 30

[server]
endpoint  = https://your-server.com
api_key   = your-user-api-key
```

```bash
# View current config
ttl config

# Configure AI interactively
ttl config ai
```

---

## 📤 Export Your Data

```bash
# Export as JSON
ttl export --format json

# Export as CSV
ttl export --format csv

# Export to specific file
ttl export --format json --output backup.json
```

---

## 🏗️ Project Structure

```
ttl
├── main.go              # Entry point, CLI setup (cobra)
├── command/             # CLI command definitions
│   ├── commands.go      # Core commands (get/add/del/tag/open...)
│   ├── ai.go            # AI Agent command
│   ├── log.go           # Work log commands
│   ├── export.go        # Export command
│   └── server.go        # Server commands
├── ai/                  # AI Agent (ReAct loop)
│   ├── client.go        # LLM HTTP client
│   ├── agent.go         # ReAct engine + tool execution
│   └── prompt.go        # System prompt
├── db/                  # Storage layer (bbolt)
│   ├── db.go            # Database initialization
│   ├── storage.go       # Local storage implementation
│   ├── tenant_storage.go# Multi-tenant storage router
│   ├── user_store.go    # User CRUD (users.json)
│   └── context.go       # Request-scoped storage
├── api/                 # HTTP server
│   ├── server.go        # Server startup
│   ├── handlers.go      # REST API handlers
│   └── middleware.go     # Auth middleware
├── mcp/                 # MCP protocol
│   ├── tools.go         # MCP tool definitions
│   └── handlers.go      # MCP tool handlers
├── sync/                # Data sync logic
├── models/              # Shared data models
├── conf/                # Config file (INI) handling
└── util/                # Utility functions
```

---

## 🔧 Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | [Go 1.23](https://golang.org) |
| CLI Framework | [cobra](https://github.com/spf13/cobra) |
| Storage | [bbolt](https://github.com/etcd-io/bbolt) |
| Configuration | [ini.v1](https://gopkg.in/ini.v1) |
| MCP Protocol | [mcp-go](https://github.com/mark3labs/mcp-go) |
| AI API | OpenAI Chat Completions API (compatible) |

---

## 🌐 Translations

- [简体中文](README.zh-CN.md)
- [日本語](README.ja.md)
- [Español](README.es.md)
- [Français](README.fr.md)
- [Português](README.pt.md)

---

## 🤝 Contributing

Contributions are welcome! Here's how you can help:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

For major changes, please open an issue first to discuss what you'd like to change.

---

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

---

## 🙏 Acknowledgments

- [cobra](https://github.com/spf13/cobra) for the excellent CLI framework
- [bbolt](https://github.com/etcd-io/bbolt) for the reliable embedded key-value store
- [mcp-go](https://github.com/mark3labs/mcp-go) for MCP protocol support
- The open-source community

---

<div align="center">

**Made with ❤️ by developers who hate searching for lost knowledge**

</div>
