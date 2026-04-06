<div align="center">

# ttl

### 你的个人知识归档，由 AI 驱动

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![MCP](https://img.shields.io/badge/MCP-Supported-green.svg)](https://modelcontextprotocol.io/)

[English](README.md) | [日本語](README.ja.md) | [Español](README.es.md) | [Français](README.fr.md) | [Português](README.pt.md)

---

*一个轻量级的个人数据管理 CLI 工具。将任何内容以键值对形式存储，即时搜索，并让 AI 用自然语言管理一切。*

</div>

---

## 📖 项目故事

每个开发者都经历过这样的时刻：

> "那个上个月用的 Docker 命令是什么来着？"
> "同事分享的配置文件放哪了？"
> "我看过一篇相关的文章……但找不到了。"

我们把知识存得到处都是 —— 浏览器标签、Slack 消息、邮件、书签夹、笔记应用。真正需要的时候，却要在无尽的标签和历史记录中浪费时间翻找。

**这也是我的痛点。**

所以我开发了 **ttl**。

名字来自 "Time to Live" —— 但有不一样的含义。不是过期，而是让知识**永久保存**。

- 把所有内容存到一个地方，用键值对的方式
- 加上标签，方便整理
- 按关键词即时搜索
- 用 AI 和自然语言来管理它

不再翻找旧邮件、滚动聊天记录。只需要 `ttl get <关键词>`，就能立刻找到。

**ttl 就像你的个人知识归档库 —— 你需要什么，什么时候都有。**

---

## ✨ 功能特性

| 功能 | 描述 |
|------|------|
| 🗄️ **本地 KV 存储** | 快速、零配置的嵌入式数据库 (bbolt) |
| 🏷️ **标签系统** | 用灵活、可搜索的标签组织资源 |
| 🔍 **模糊搜索** | 跨键名和标签即时查找所需内容 |
| 🤖 **AI Agent** | 用自然语言管理数据 (OpenAI / DeepSeek / Ollama) |
| 📝 **工作日志** | 记录每日工作，用 AI 生成周报/月报 |
| 🔗 **MCP 协议** | 让 AI 工具 (Claude Code、Cursor) 直接操作你的数据 |
| ☁️ **云端同步** | 自建多租户服务器，每用户数据隔离 |
| 🔒 **隐私优先** | AI 永不发送你的值 —— 只发送键和标签 |
| 🚀 **智能打开** | 用系统默认程序打开 URL 和文件 |
| 📤 **数据导出** | 导出为 JSON 或 CSV 格式 |

---

## 🚀 快速开始

### 安装

#### Linux / macOS

```bash
# 从 GitHub releases 安装
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/ZHANGSHUNLIN/TTL-CLI/main/install.sh)"

# 或从源码构建
go build -o ttl .
sudo mv ttl /usr/local/bin/
```

#### Windows

```powershell
# 从 GitHub releases 安装
irm https://raw.githubusercontent.com/ZHANGSHUNLIN/TTL-CLI/main/install.ps1 | iex
```

#### 自定义下载地址

适用于内网或自定义镜像：

```bash
# Linux/macOS
TTL_DOWNLOAD_URL="https://your-mirror.com/ttl-cli-v1.0.0-linux-amd64" /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/ZHANGSHUNLIN/TTL-CLI/main/install.sh)"
```

```powershell
# Windows
$env:TTL_DOWNLOAD_URL="https://your-mirror.com/ttl-cli-v1.0.0-windows-amd64.zip"; irm https://raw.githubusercontent.com/ZHANGSHUNLIN/TTL-CLI/main/install.ps1 | iex
```

### 基础用法

```bash
# 添加资源
ttl add my-link https://example.com

# 添加并打标签
ttl add docker-cmd "docker run -d -p 8080:80 nginx"
ttl tag docker-cmd dev ops

# 搜索资源
ttl get docker

# 在浏览器中打开
ttl open my-link

# 删除
ttl del old-key
```

---

## 🤖 AI Agent

一个命令替代十个。用自然语言描述你想要的操作。

```bash
# 先配置 AI
ttl config ai

# 然后使用自然语言
ttl ai "保存这个 nginx 配置：端口 8080 改为 443"
ttl ai "查找所有 docker 相关的资源"
ttl ai "打开 sugar 仪表板"
ttl ai "我最近保存了什么？"
ttl ai "给 nginx-config 加上 ops 和 deploy 标签"
```

### 隐私设计

AI Agent 只向 LLM 发送**资源键和标签**。你的值 —— 可能包含密码、令牌、内部 URL 或敏感数据 —— 永远不会离开你的机器。只有工作日志内容会完整发送用于生成摘要。

### 兼容模型

- OpenAI (GPT-4、GPT-4o、GPT-4o-mini)
- DeepSeek
- Moonshot
- Ollama (本地模型)
- 任何兼容 OpenAI Chat Completions API 的模型

---

## 📝 工作日志

记录每日工作，让 AI 生成报告。

```bash
# 写日志
ttl log write "完成用户模块重构" --tags "项目A,开发"

# 查看日志
ttl log list                    # 今天的日志
ttl log list --range week       # 本周
ttl log list --range month      # 本月

# AI 生成周报
ttl ai "总结本周工作日志"
ttl ai "生成一份 Markdown 格式的周报"
```

---

## 🔗 MCP 协议集成

通过 [Model Context Protocol](https://modelcontextprotocol.io/) 让 AI 工具操作你的数据。

### 启动 MCP 服务器

```bash
ttl mcp
```

### Claude Code 集成

添加到 `~/.claude/claude_code_config.json`：

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

现在 Claude Code 可以直接读取、写入和管理你的资源了！

**可用的 MCP 工具：**
- `ttl_get` — 获取资源
- `ttl_add` — 添加新资源
- `ttl_update` — 更新资源
- `ttl_delete` — 删除资源
- `ttl_tag` — 添加标签
- `ttl_dtag` — 移除标签
- `ttl_open` — 打开资源
- `ttl_rename` — 重命名资源

---

## ☁️ 云端服务器与同步

### 启动你的服务器

```bash
# 创建用户
ttl server user add alice

# 启动多租户服务器
ttl server start --port 8080
```

### 同步数据

```bash
# 配置远程服务器
ttl config
# 编辑 server 部分，填入端点和 API 密钥

# 同步本地与远程数据
ttl sync
```

**架构特点：**
- 多租户设计，每用户独立数据库
- API Key 认证
- REST API 用于程序化访问
- MCP HTTP 端点 `/mcp` 供 AI 客户端使用

---

## ⚙️ 配置

配置文件：`~/.ttl/ttl.ini`

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
# 查看当前配置
ttl config

# 交互式配置 AI
ttl config ai
```

---

## 📤 导出数据

```bash
# 导出为 JSON
ttl export --format json

# 导出为 CSV
ttl export --format csv

# 导出到指定文件
ttl export --format json --output backup.json
```

---

## 🏗️ 项目结构

```
ttl
├── main.go              # 入口点，CLI 设置 (cobra)
├── command/             # CLI 命令定义
│   ├── commands.go      # 核心命令 (get/add/del/tag/open...)
│   ├── ai.go            # AI Agent 命令
│   ├── log.go           # 工作日志命令
│   ├── export.go        # 导出命令
│   └── server.go        # 服务器命令
├── ai/                  # AI Agent (ReAct 循环)
│   ├── client.go        # LLM HTTP 客户端
│   ├── agent.go         # ReAct 引擎 + 工具执行
│   └── prompt.go        # 系统提示词
├── db/                  # 存储层 (bbolt)
│   ├── db.go            # 数据库初始化
│   ├── storage.go       # 本地存储实现
│   ├── tenant_storage.go# 多租户存储路由
│   ├── user_store.go    # 用户 CRUD (users.json)
│   └── context.go       # 请求范围的存储
├── api/                 # HTTP 服务器
│   ├── server.go        # 服务器启动
│   ├── handlers.go      # REST API 处理器
│   └── middleware.go     # 认证中间件
├── mcp/                 # MCP 协议
│   ├── tools.go         # MCP 工具定义
│   └── handlers.go      # MCP 工具处理器
├── sync/                # 数据同步逻辑
├── models/              # 共享数据模型
├── conf/                # 配置文件 (INI) 处理
└── util/                # 工具函数
```

---

## 🔧 技术栈

| 组件 | 技术 |
|------|------|
| 语言 | [Go 1.23](https://golang.org) |
| CLI 框架 | [cobra](https://github.com/spf13/cobra) |
| 存储 | [bbolt](https://github.com/etcd-io/bbolt) |
| 配置 | [ini.v1](https://gopkg.in/ini.v1) |
| MCP 协议 | [mcp-go](https://github.com/mark3labs/mcp-go) |
| AI API | OpenAI Chat Completions API (兼容) |

---

## 🌐 翻译

- [English](README.md)
- [日本語](README.ja.md)
- [Español](README.es.md)
- [Français](README.fr.md)
- [Português](README.pt.md)

---

## 🤝 贡献

欢迎贡献！你可以这样帮忙：

1. Fork 仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 打开 Pull Request

对于重大更改，请先开 Issue 讨论你想做什么。

---

## 📄 许可证

本项目采用 Apache License 2.0 许可证 — 详见 [LICENSE](LICENSE) 文件。

---

## 🙏 致谢

- [cobra](https://github.com/spf13/cobra) 提供优秀的 CLI 框架
- [bbolt](https://github.com/etcd-io/bbolt) 提供可靠的嵌入式键值存储
- [mcp-go](https://github.com/mark3labs/mcp-go) 提供 MCP 协议支持
- 开源社区

---

<div align="center">

**由讨厌寻找丢失知识的开发者用 ❤️ 打造**

</div>
