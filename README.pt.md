<div align="center">

# ttl

### Seu Arquivo Pessoal de Conhecimento, Impulsionado por IA

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![MCP](https://img.shields.io/badge/MCP-Supported-green.svg)](https://modelcontextprotocol.io/)

[English](README.md) | [简体中文](README.zh-CN.md) | [日本語](README.ja.md) | [Español](README.es.md) | [Français](README.fr.md)

---

*Uma ferramenta CLI leve para gerenciamento de dados pessoais. Armazene qualquer coisa como pares chave-valor, busque instantaneamente e deixe a IA gerenciar tudo com linguagem natural.*

</div>

---

## 📖 A História

Todo desenvolvedor já passou por isso:

> "Espera, qual era aquele comando Docker que usei mês passado?"
> "Onde meu colega compartilhou aquele arquivo de configuração?"
> "Eu sei que li um artigo sobre isso... mas não consigo encontrar."

Armazenamos conhecimento em todo lugar — abas do navegador, mensagens do Slack, emails, pastas de favoritos, aplicativos de notas. Quando realmente precisamos, perdemos tempo cavando em abas infinitas e rolando o histórico de chat.

**Essa também era minha dor.**

Por isso criei **ttl**.

O nome vem de "Time to Live" — mas com um significado diferente. Em vez de expirar, é sobre dar ao seu conhecimento um **tempo para viver** para sempre.

- Armazene tudo em um só lugar como pares chave-valor
- Marque com tags para fácil organização
- Busque instantaneamente por palavra-chave
- Use IA para gerenciar com linguagem natural

Chega de procurar em emails antigos ou rolar o histórico do Slack. Apenas `ttl get <palavra-chave>` e você tem.

**ttl é seu arquivo pessoal de conhecimento — tudo que você precisa, quando você precisa.**

---

## ✨ Funcionalidades

| Funcionalidade | Descrição |
|------|------|
| 🗄️ **Armazenamento KV Local** | Banco de dados embarcado rápido, sem configuração (bbolt) |
| 🏷️ **Sistema de Tags** | Organize recursos com tags flexíveis e pesquisáveis |
| 🔍 **Busca Fuzzy** | Encontre o que precisa instantaneamente entre chaves e tags |
| 🤖 **Agente IA** | Gerencie dados com linguagem natural (OpenAI / DeepSeek / Ollama) |
| 📝 **Log de Trabalho** | Acompanhe o trabalho diário, gere relatórios semanais/mensais com IA |
| 🔗 **Protocolo MCP** | Deixe ferramentas IA (Claude Code, Cursor) operar seus dados diretamente |
| ☁️ **Sincronização em Nuvem** | Auto-hospede um servidor multi-tenant com isolamento de dados por usuário |
| 🔒 **Privacidade Primeiro** | A IA nunca envia seus valores — apenas chaves e tags |
| 🚀 **Abertura Inteligente** | Abra URLs e arquivos com programas padrão do sistema |
| 📤 **Exportar** | Exporte dados como JSON ou CSV |

---

## 🚀 Início Rápido

### Instalação

```bash
# Compilar a partir do código-fonte
go build -o ttl .
sudo mv ttl /usr/local/bin/

# Ou usar o script de instalação
bash install.sh
```

### Uso Básico

```bash
# Adicionar um recurso
ttl add my-link https://example.com

# Adicionar com tags
ttl add docker-cmd "docker run -d -p 8080:80 nginx"
ttl tag docker-cmd dev ops

# Buscar recursos
ttl get docker

# Abrir no navegador
ttl open my-link

# Deletar
ttl del old-key
```

---

## 🤖 Agente IA

Um comando substitui dez. Apenas descreva o que quer em linguagem natural.

```bash
# Primeiro configure a IA
ttl config ai

# Depois use linguagem natural
ttl ai "salva esta config nginx: porta 8080 mudada para 443"
ttl ai "encontre todos meus recursos relacionados a docker"
ttl ai "abra o dashboard sugar"
ttl ai "o que eu salvei recentemente?"
ttl ai "tag nginx-config com ops e deploy"
```

### Privacidade por Design

O Agente IA apenas envia **chaves e tags de recursos** para o LLM. Seus valores — que podem conter senhas, tokens, URLs internas ou dados sensíveis — nunca deixam sua máquina. Apenas o conteúdo do log de trabalho é enviado completo para gerar resumos.

### Modelos Compatíveis

- OpenAI (GPT-4, GPT-4o, GPT-4o-mini)
- DeepSeek
- Moonshot
- Ollama (modelos locais)
- Qualquer modelo compatível com OpenAI Chat Completions API

---

## 📝 Log de Trabalho

Acompanhe seu trabalho diário e deixe a IA gerar relatórios.

```bash
# Escrever uma entrada de log
ttl log write "Refatoração do módulo de usuários concluída" --tags "projetoA,dev"

# Ver logs
ttl log list                    # Logs de hoje
ttl log list --range week       # Esta semana
ttl log list --range month      # Este mês

# Relatório semanal com IA
ttl ai "resuma meus logs de trabalho desta semana"
ttl ai "gere um relatório semanal em formato markdown"
```

---

## 🔗 Integração com Protocolo MCP

Deixe ferramentas IA operar seus dados via [Model Context Protocol](https://modelcontextprotocol.io/).

### Iniciar Servidor MCP

```bash
ttl mcp
```

### Integração com Claude Code

Adicione ao `~/.claude/claude_code_config.json`:

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

Agora Claude Code pode ler, escrever e gerenciar seus recursos diretamente!

**Ferramentas MCP disponíveis:**
- `ttl_get` — Obter recursos
- `ttl_add` — Adicionar novos recursos
- `ttl_update` — Atualizar recursos existentes
- `ttl_delete` — Deletar recursos
- `ttl_tag` — Adicionar tags
- `ttl_dtag` — Remover tags
- `ttl_open` — Abrir recursos
- `ttl_rename` — Renomear recursos

---

## ☁️ Servidor em Nuvem e Sincronização

### Iniciar Seu Servidor

```bash
# Criar um usuário
ttl server user add alice

# Iniciar servidor multi-tenant
ttl server start --port 8080
```

### Sincronizar Seus Dados

```bash
# Configurar servidor remoto
ttl config
# Edite a seção server com seu endpoint e API key

# Sincronizar local <-> remoto
ttl sync
```

**Arquitetura:**
- Design multi-tenant com bancos de dados isolados por usuário
- Autenticação via API Key
- REST API para acesso programático
- Endpoint HTTP MCP em `/mcp` para clientes IA

---

## ⚙️ Configuração

Arquivo de configuração: `~/.ttl/ttl.ini`

```ini
[default]
db_path = ~/.ttl/data.db

[ai]
api_key   = sua-api-key-aqui
base_url  = https://api.openai.com
model     = gpt-4o-mini
timeout   = 30

[server]
endpoint  = https://seu-servidor.com
api_key   = sua-user-api-key
```

```bash
# Ver configuração atual
ttl config

# Configurar IA interativamente
ttl config ai
```

---

## 📤 Exportar Seus Dados

```bash
# Exportar como JSON
ttl export --format json

# Exportar como CSV
ttl export --format csv

# Exportar para arquivo específico
ttl export --format json --output backup.json
```

---

## 🏗️ Estrutura do Projeto

```
ttl
├── main.go              # Ponto de entrada, configuração CLI (cobra)
├── command/             # Definições de comandos CLI
│   ├── commands.go      # Comandos principais (get/add/del/tag/open...)
│   ├── ai.go            # Comando do Agente IA
│   ├── log.go           # Comandos de log de trabalho
│   ├── export.go        # Comando de exportação
│   └── server.go        # Comandos do servidor
├── ai/                  # Agente IA (loop ReAct)
│   ├── client.go        # Cliente HTTP LLM
│   ├── agent.go         # Motor ReAct + execução de ferramentas
│   └── prompt.go        # Prompt do sistema
├── db/                  # Camada de armazenamento (bbolt)
│   ├── db.go            # Inicialização do banco de dados
│   ├── storage.go       # Implementação do armazenamento local
│   ├── tenant_storage.go# Roteador de armazenamento multi-tenant
│   ├── user_store.go    # CRUD de usuários (users.json)
│   └── context.go       # Armazenamento por requisição
├── api/                 # Servidor HTTP
│   ├── server.go        # Inicialização do servidor
│   ├── handlers.go      # Handlers da API REST
│   └── middleware.go     # Middleware de autenticação
├── mcp/                 # Protocolo MCP
│   ├── tools.go         # Definições das ferramentas MCP
│   └── handlers.go      # Handlers das ferramentas MCP
├── sync/                # Lógica de sincronização de dados
├── models/              # Modelos de dados compartilhados
├── conf/                # Manipulação do arquivo de configuração (INI)
└── util/                # Funções utilitárias
```

---

## 🔧 Stack Tecnológico

| Componente | Tecnologia |
|------|------|
| Linguagem | [Go 1.23](https://golang.org) |
| Framework CLI | [cobra](https://github.com/spf13/cobra) |
| Armazenamento | [bbolt](https://github.com/etcd-io/bbolt) |
| Configuração | [ini.v1](https://gopkg.in/ini.v1) |
| Protocolo MCP | [mcp-go](https://github.com/mark3labs/mcp-go) |
| API IA | OpenAI Chat Completions API (compatível) |

---

## 🌐 Traduções

- [English](README.md)
- [简体中文](README.zh-CN.md)
- [日本語](README.ja.md)
- [Español](README.es.md)
- [Français](README.fr.md)

---

## 🤝 Contribuindo

Contribuições são bem-vindas! Aqui está como você pode ajudar:

1. Faça fork do repositório
2. Crie uma branch de funcionalidade (`git checkout -b feature/amazing-feature`)
3. Commite suas mudanças (`git commit -m 'Add amazing feature'`)
4. Push para a branch (`git push origin feature/amazing-feature`)
5. Abra um Pull Request

Para mudanças grandes, por favor abra primeiro uma issue para discutir o que você gostaria de mudar.

---

## 📄 Licença

Este projeto está licenciado sob a Apache License 2.0 - veja o arquivo [LICENSE](LICENSE) para mais detalhes.

---

## 🙏 Agradecimentos

- [cobra](https://github.com/spf13/cobra) pelo excelente framework CLI
- [bbolt](https://github.com/etcd-io/bbolt) pelo armazenamento key-value embarcado confiável
- [mcp-go](https://github.com/mark3labs/mcp-go) pelo suporte ao protocolo MCP
- A comunidade open-source

---

<div align="center">

**Feito com ❤️ por desenvolvedores que odeiam procurar conhecimento perdido**

</div>
