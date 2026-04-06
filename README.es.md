<div align="center">

# ttl

### Tu Archivo Personal de Conocimiento, Impulsado por IA

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![MCP](https://img.shields.io/badge/MCP-Supported-green.svg)](https://modelcontextprotocol.io/)

[English](README.md) | [简体中文](README.zh-CN.md) | [日本語](README.ja.md) | [Français](README.fr.md) | [Português](README.pt.md)

---

*Una herramienta CLI ligera para gestión de datos personales. Almacena cualquier cosa como pares clave-valor, busca al instante y deja que la IA gestione todo con lenguaje natural.*

</div>

---

## 📖 La Historia

Todo desarrollador ha estado ahí:

> "Espera, ¿cuál era ese comando de Docker que usé el mes pasado?"
> "¿Dónde compartió mi compañero ese archivo de configuración?"
> "Sé que leí un artículo sobre esto... pero no lo encuentro por ningún lado."

Guardamos el conocimiento en todas partes — pestañas del navegador, mensajes de Slack, correos, carpetas de marcadores, aplicaciones de notas. Cuando realmente lo necesitamos, perdemos tiempo cavando en pestañas infinitas y desplazándonos por el historial de chat.

**Este también era mi dolor.**

Por eso construí **ttl**.

El nombre viene de "Time to Live" — pero con un significado diferente. En lugar de expirar, se trata de dar a tu conocimiento un **tiempo para vivir** para siempre.

- Almacena todo en un solo lugar como pares clave-valor
- Etiquétalo para fácil organización
- Busca instantáneamente por palabra clave
- Usa IA para gestionarlo con lenguaje natural

No más buscar en correos antiguos o desplazarse por el historial de Slack. Solo `ttl get <palabra clave>` y lo tienes.

**ttl es tu archivo personal de conocimiento — todo lo que necesitas, cuando lo necesitas.**

---

## ✨ Características

| Característica | Descripción |
|------|------|
| 🗄️ **Almacenamiento KV Local** | Base de datos embebida rápida, sin configuración (bbolt) |
| 🏷️ **Sistema de Etiquetas** | Organiza recursos con etiquetas flexibles y buscables |
| 🔍 **Búsqueda Difusa** | Encuentra lo que necesitas al instante entre claves y etiquetas |
| 🤖 **Agente IA** | Gestiona datos con lenguaje natural (OpenAI / DeepSeek / Ollama) |
| 📝 **Registro de Trabajo** | Rastrea trabajo diario, genera reportes semanales/mensuales con IA |
| 🔗 **Protocolo MCP** | Permite a herramientas IA (Claude Code, Cursor) operar tus datos directamente |
| ☁️ **Sincronización en la Nube** | Aloja tu propio servidor multi-tenant con aislamiento de datos por usuario |
| 🔒 **Privacidad Primero** | La IA nunca envía tus valores — solo claves y etiquetas |
| 🚀 **Apertura Inteligente** | Abre URLs y archivos con programas predeterminados del sistema |
| 📤 **Exportar** | Exporta datos como JSON o CSV |

---

## 🚀 Inicio Rápido

### Instalación

#### Linux / macOS

```bash
# Instalar desde GitHub releases
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/ZHANGSHUNLIN/TTL-CLI/main/install.sh)"

# O compilar desde el código fuente
go build -o ttl .
sudo mv ttl /usr/local/bin/
```

#### Windows

```powershell
# Instalar desde GitHub releases
irm https://raw.githubusercontent.com/ZHANGSHUNLIN/TTL-CLI/main/install.ps1 | iex
```

#### URL de Descarga Personalizada

Para redes internas o mirrors personalizados:

```bash
# Linux/macOS
TTL_DOWNLOAD_URL="https://your-mirror.com/ttl-cli-v1.0.0-linux-amd64" /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/ZHANGSHUNLIN/TTL-CLI/main/install.sh)"
```

```powershell
# Windows
$env:TTL_DOWNLOAD_URL="https://your-mirror.com/ttl-cli-v1.0.0-windows-amd64.zip"; irm https://raw.githubusercontent.com/ZHANGSHUNLIN/TTL-CLI/main/install.ps1 | iex
```

### Uso Básico

```bash
# Agregar un recurso
ttl add my-link https://example.com

# Agregar con etiquetas
ttl add docker-cmd "docker run -d -p 8080:80 nginx"
ttl tag docker-cmd dev ops

# Buscar recursos
ttl get docker

# Abrir en navegador
ttl open my-link

# Eliminar
ttl del old-key
```

---

## 🤖 Agente IA

Un comando reemplaza diez. Solo describe lo que quieres en lenguaje natural.

```bash
# Primero configura la IA
ttl config ai

# Luego usa lenguaje natural
ttl ai "guarda esta configuración nginx: puerto 8080 cambiado a 443"
ttl ai "encuentra todos mis recursos relacionados con docker"
ttl ai "abre el dashboard de sugar"
ttl ai "¿qué guardé recientemente?"
ttl ai "etiqueta nginx-config con ops y deploy"
```

### Privacidad por Diseño

El Agente IA solo envía **claves y etiquetas de recursos** al LLM. Tus valores — que pueden contener contraseñas, tokens, URLs internas o datos sensibles — nunca abandonan tu máquina. Solo el contenido del registro de trabajo se envía completo para generar resúmenes.

### Modelos Compatibles

- OpenAI (GPT-4, GPT-4o, GPT-4o-mini)
- DeepSeek
- Moonshot
- Ollama (modelos locales)
- Cualquier modelo compatible con OpenAI Chat Completions API

---

## 📝 Registro de Trabajo

Rastrea tu trabajo diario y deja que la IA genere reportes.

```bash
# Escribir una entrada de registro
ttl log write "Terminé la refactorización del módulo de usuarios" --tags "proyectoA,dev"

# Ver registros
ttl log list                    # Registros de hoy
ttl log list --range week       # Esta semana
ttl log list --range month      # Este mes

# Reporte semanal con IA
ttl ai "resume mis registros de trabajo de esta semana"
ttl ai "genera un reporte semanal en formato markdown"
```

---

## 🔗 Integración con Protocolo MCP

Permite que herramientas IA operen tus datos vía [Model Context Protocol](https://modelcontextprotocol.io/).

### Iniciar Servidor MCP

```bash
ttl mcp
```

### Integración con Claude Code

Agrega a `~/.claude/claude_code_config.json`:

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

¡Ahora Claude Code puede leer, escribir y gestionar tus recursos directamente!

**Herramientas MCP disponibles:**
- `ttl_get` — Obtener recursos
- `ttl_add` — Agregar nuevos recursos
- `ttl_update` — Actualizar recursos existentes
- `ttl_delete` — Eliminar recursos
- `ttl_tag` — Agregar etiquetas
- `ttl_dtag` — Remover etiquetas
- `ttl_open` — Abrir recursos
- `ttl_rename` — Renombrar recursos

---

## ☁️ Servidor en la Nube y Sincronización

### Iniciar Tu Servidor

```bash
# Crear un usuario
ttl server user add alice

# Iniciar servidor multi-tenant
ttl server start --port 8080
```

### Sincronizar Tus Datos

```bash
# Configurar servidor remoto
ttl config
# Edita la sección server con tu endpoint y API key

# Sincronizar local <-> remoto
ttl sync
```

**Arquitectura:**
- Diseño multi-tenant con bases de datos aisladas por usuario
- Autenticación via API Key
- REST API para acceso programático
- Endpoint HTTP MCP en `/mcp` para clientes IA

---

## ⚙️ Configuración

Archivo de configuración: `~/.ttl/ttl.ini`

```ini
[default]
db_path = ~/.ttl/data.db

[ai]
api_key   = tu-api-key-aqui
base_url  = https://api.openai.com
model     = gpt-4o-mini
timeout   = 30

[server]
endpoint  = https://tu-servidor.com
api_key   = tu-user-api-key
```

```bash
# Ver configuración actual
ttl config

# Configurar IA interactivamente
ttl config ai
```

---

## 📤 Exportar Tus Datos

```bash
# Exportar como JSON
ttl export --format json

# Exportar como CSV
ttl export --format csv

# Exportar a archivo específico
ttl export --format json --output backup.json
```

---

## 🏗️ Estructura del Proyecto

```
ttl
├── main.go              # Punto de entrada, configuración CLI (cobra)
├── command/             # Definiciones de comandos CLI
│   ├── commands.go      # Comandos principales (get/add/del/tag/open...)
│   ├── ai.go            # Comando del Agente IA
│   ├── log.go           # Comandos de registro de trabajo
│   ├── export.go        # Comando de exportación
│   └── server.go        # Comandos del servidor
├── ai/                  # Agente IA (ciclo ReAct)
│   ├── client.go        # Cliente HTTP LLM
│   ├── agent.go         # Motor ReAct + ejecución de herramientas
│   └── prompt.go        # Prompt del sistema
├── db/                  # Capa de almacenamiento (bbolt)
│   ├── db.go            # Inicialización de base de datos
│   ├── storage.go       # Implementación de almacenamiento local
│   ├── tenant_storage.go# Router de almacenamiento multi-tenant
│   ├── user_store.go    # CRUD de usuarios (users.json)
│   └── context.go       # Almacenamiento por petición
├── api/                 # Servidor HTTP
│   ├── server.go        # Inicio del servidor
│   ├── handlers.go      # Handlers REST API
│   └── middleware.go     # Middleware de autenticación
├── mcp/                 # Protocolo MCP
│   ├── tools.go         # Definiciones de herramientas MCP
│   └── handlers.go      # Handlers de herramientas MCP
├── sync/                # Lógica de sincronización de datos
├── models/              # Modelos de datos compartidos
├── conf/                # Manejo de archivo de configuración (INI)
└── util/                # Funciones de utilidad
```

---

## 🔧 Stack Tecnológico

| Componente | Tecnología |
|------|------|
| Lenguaje | [Go 1.23](https://golang.org) |
| Framework CLI | [cobra](https://github.com/spf13/cobra) |
| Almacenamiento | [bbolt](https://github.com/etcd-io/bbolt) |
| Configuración | [ini.v1](https://gopkg.in/ini.v1) |
| Protocolo MCP | [mcp-go](https://github.com/mark3labs/mcp-go) |
| API IA | OpenAI Chat Completions API (compatible) |

---

## 🌐 Traducciones

- [English](README.md)
- [简体中文](README.zh-CN.md)
- [日本語](README.ja.md)
- [Français](README.fr.md)
- [Português](README.pt.md)

---

## 🤝 Contribuir

¡Las contribuciones son bienvenidas! Aquí está cómo puedes ayudar:

1. Haz fork del repositorio
2. Crea una rama de característica (`git checkout -b feature/amazing-feature`)
3. Confirma tus cambios (`git commit -m 'Add amazing feature'`)
4. Empuja a la rama (`git push origin feature/amazing-feature`)
5. Abre un Pull Request

Para cambios importantes, por favor abre primero un issue para discutir lo que te gustaría cambiar.

---

## 📄 Licencia

Este proyecto está licenciado bajo la Apache License 2.0 - ver el archivo [LICENSE](LICENSE) para más detalles.

---

## 🙏 Agradecimientos

- [cobra](https://github.com/spf13/cobra) por el excelente framework CLI
- [bbolt](https://github.com/etcd-io/bbolt) por el almacenamiento key-value embebido confiable
- [mcp-go](https://github.com/mark3labs/mcp-go) por el soporte del protocolo MCP
- La comunidad open-source

---

<div align="center">

**Hecho con ❤️ por desarrolladores que odian buscar conocimiento perdido**

</div>
