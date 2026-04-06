<div align="center">

# ttl

### Votre Archive Personnelle de Connaissances, Propulsée par l'IA

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![MCP](https://img.shields.io/badge/MCP-Supported-green.svg)](https://modelcontextprotocol.io/)

[English](README.md) | [简体中文](README.zh-CN.md) | [日本語](README.ja.md) | [Español](README.es.md) | [Português](README.pt.md)

---

*Un outil CLI léger pour la gestion de données personnelles. Stockez n'importe quoi sous forme de paires clé-valeur, recherchez instantanément, et laissez l'IA tout gérer en langage naturel.*

</div>

---

## 📖 L'Histoire

Chaque développeur y a déjà été confronté :

> "Attends, c'était quoi cette commande Docker que j'ai utilisée le mois dernier ?"
> "Où mon collègue a-t-il partagé ce fichier de configuration ?"
> "Je sais que j'ai lu un article là-dessus... mais je ne le retrouve pas."

Nous stockons nos connaissances partout — onglets du navigateur, messages Slack, emails, dossiers de favoris, applications de notes. Quand nous en avons vraiment besoin, nous perdons du temps à fouiller dans des onglets sans fin et à défiler l'historique des conversations.

**C'était aussi mon problème.**

C'est pourquoi j'ai créé **ttl**.

Le nom vient de "Time to Live" — mais avec une signification différente. Au lieu d'expirer, il s'agit de donner à vos connaissances un **temps pour vivre** éternellement.

- Stockez tout au même endroit sous forme de paires clé-valeur
- Étiquetez pour une organisation facile
- Recherchez instantanément par mot-clé
- Utilisez l'IA pour gérer le tout en langage naturel

Plus besoin de chercher dans les vieux emails ou de défiler l'historique Slack. Juste `ttl get <mot-clé>` et c'est là.

**ttl est votre archive personnelle de connaissances — tout ce dont vous avez besoin, quand vous en avez besoin.**

---

## ✨ Fonctionnalités

| Fonctionnalité | Description |
|------|------|
| 🗄️ **Stockage KV Local** | Base de données embarquée rapide, sans configuration (bbolt) |
| 🏷️ **Système de Tags** | Organisez les ressources avec des tags flexibles et recherchables |
| 🔍 **Recherche Floue** | Trouvez ce dont vous avez besoin instantanément parmi clés et tags |
| 🤖 **Agent IA** | Gérez les données en langage naturel (OpenAI / DeepSeek / Ollama) |
| 📝 **Journal de Travail** | Suivez le travail quotidien, générez des rapports hebdomadaires/mensuels avec l'IA |
| 🔗 **Protocole MCP** | Laissez les outils IA (Claude Code, Cursor) opérer vos données directement |
| ☁️ **Sync Cloud** | Auto-hébergez un serveur multi-tenant avec isolation des données par utilisateur |
| 🔒 **Confidentialité d'Abord** | L'IA n'envoie jamais vos valeurs — seulement les clés et tags |
| 🚀 **Ouverture Intelligente** | Ouvrez URLs et fichiers avec les programmes par défaut du système |
| 📤 **Export** | Exportez les données en JSON ou CSV |

---

## 🚀 Démarrage Rapide

### Installation

#### Linux / macOS

```bash
# Installer depuis GitHub releases
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/ZHANGSHUNLIN/TTL-CLI/main/install.sh)"

# Ou compiler depuis les sources
go build -o ttl .
sudo mv ttl /usr/local/bin/
```

#### Windows

```powershell
# Installer depuis GitHub releases
irm https://raw.githubusercontent.com/ZHANGSHUNLIN/TTL-CLI/main/install.ps1 | iex
```

#### URL de Téléchargement Personnalisé

Pour les réseaux internes ou mirrors personnalisés :

```bash
# Linux/macOS
TTL_DOWNLOAD_URL="https://your-mirror.com/ttl-cli-v1.0.0-linux-amd64" /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/ZHANGSHUNLIN/TTL-CLI/main/install.sh)"
```

```powershell
# Windows
irm https://raw.githubusercontent.com/ZHANGSHUNLIN/TTL-CLI/main/install.ps1 | iex
install.ps1 -DownloadUrl "https://your-mirror.com/ttl-cli-v1.0.0-windows-amd64.zip"
```

### Utilisation de Base

```bash
# Ajouter une ressource
ttl add my-link https://example.com

# Ajouter avec des tags
ttl add docker-cmd "docker run -d -p 8080:80 nginx"
ttl tag docker-cmd dev ops

# Rechercher des ressources
ttl get docker

# Ouvrir dans le navigateur
ttl open my-link

# Supprimer
ttl del old-key
```

---

## 🤖 Agent IA

Une commande remplace dix. Décrivez simplement ce que vous voulez en langage naturel.

```bash
# D'abord configurer l'IA
ttl config ai

# Puis utiliser le langage naturel
ttl ai "sauvegarde cette config nginx : port 8080 changé en 443"
ttl ai "trouve toutes mes ressources liées à docker"
ttl ai "ouvre le dashboard sugar"
ttl ai "qu'ai-je sauvegardé récemment ?"
ttl ai "tag nginx-config avec ops et deploy"
```

### Confidentialité par Conception

L'Agent IA envoie uniquement **les clés et tags des ressources** au LLM. Vos valeurs — qui peuvent contenir des mots de passe, tokens, URLs internes ou données sensibles — ne quittent jamais votre machine. Seul le contenu du journal de travail est envoyé en entier pour générer des résumés.

### Modèles Compatibles

- OpenAI (GPT-4, GPT-4o, GPT-4o-mini)
- DeepSeek
- Moonshot
- Ollama (modèles locaux)
- Tout modèle compatible avec l'API OpenAI Chat Completions

---

## 📝 Journal de Travail

Suivez votre travail quotidien et laissez l'IA générer des rapports.

```bash
# Écrire une entrée de journal
ttl log write "Refactoring du module utilisateur terminé" --tags "projetA,dev"

# Voir les journaux
ttl log list                    # Journaux d'aujourd'hui
ttl log list --range week       # Cette semaine
ttl log list --range month      # Ce mois

# Rapport hebdomadaire par IA
ttl ai "résume mes journaux de travail de cette semaine"
ttl ai "génère un rapport hebdomadaire en format markdown"
```

---

## 🔗 Intégration Protocole MCP

Laissez les outils IA opérer vos données via le [Model Context Protocol](https://modelcontextprotocol.io/).

### Démarrer le Serveur MCP

```bash
ttl mcp
```

### Intégration Claude Code

Ajoutez à `~/.claude/claude_code_config.json` :

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

Maintenant Claude Code peut lire, écrire et gérer vos ressources directement !

**Outils MCP disponibles :**
- `ttl_get` — Récupérer des ressources
- `ttl_add` — Ajouter de nouvelles ressources
- `ttl_update` — Mettre à jour des ressources existantes
- `ttl_delete` — Supprimer des ressources
- `ttl_tag` — Ajouter des tags
- `ttl_dtag` — Retirer des tags
- `ttl_open` — Ouvrir des ressources
- `ttl_rename` — Renommer des ressources

---

## ☁️ Serveur Cloud & Synchronisation

### Démarrer Votre Serveur

```bash
# Créer un utilisateur
ttl server user add alice

# Démarrer le serveur multi-tenant
ttl server start --port 8080
```

### Synchroniser Vos Données

```bash
# Configurer le serveur distant
ttl config
# Éditez la section server avec votre endpoint et clé API

# Synchroniser local <-> distant
ttl sync
```

**Architecture :**
- Conception multi-tenant avec bases de données isolées par utilisateur
- Authentification par clé API
- API REST pour accès programmatique
- Endpoint HTTP MCP sur `/mcp` pour clients IA

---

## ⚙️ Configuration

Fichier de configuration : `~/.ttl/ttl.ini`

```ini
[default]
db_path = ~/.ttl/data.db

[ai]
api_key   = votre-api-key-ici
base_url  = https://api.openai.com
model     = gpt-4o-mini
timeout   = 30

[server]
endpoint  = https://votre-serveur.com
api_key   = votre-user-api-key
```

```bash
# Voir la configuration actuelle
ttl config

# Configurer l'IA interactivement
ttl config ai
```

---

## 📤 Exporter Vos Données

```bash
# Exporter en JSON
ttl export --format json

# Exporter en CSV
ttl export --format csv

# Exporter vers un fichier spécifique
ttl export --format json --output backup.json
```

---

## 🏗️ Structure du Projet

```
ttl
├── main.go              # Point d'entrée, configuration CLI (cobra)
├── command/             # Définitions des commandes CLI
│   ├── commands.go      # Commandes principales (get/add/del/tag/open...)
│   ├── ai.go            # Commande Agent IA
│   ├── log.go           # Commandes journal de travail
│   ├── export.go        # Commande d'export
│   └── server.go        # Commandes serveur
├── ai/                  # Agent IA (boucle ReAct)
│   ├── client.go        # Client HTTP LLM
│   ├── agent.go         # Moteur ReAct + exécution d'outils
│   └── prompt.go        # Prompt système
├── db/                  # Couche de stockage (bbolt)
│   ├── db.go            # Initialisation de la base de données
│   ├── storage.go       # Implémentation du stockage local
│   ├── tenant_storage.go# Routeur de stockage multi-tenant
│   ├── user_store.go    # CRUD utilisateurs (users.json)
│   └── context.go       # Stockage par requête
├── api/                 # Serveur HTTP
│   ├── server.go        # Démarrage du serveur
│   ├── handlers.go      # Handlers API REST
│   └── middleware.go     # Middleware d'authentification
├── mcp/                 # Protocole MCP
│   ├── tools.go         # Définitions des outils MCP
│   └── handlers.go      # Handlers des outils MCP
├── sync/                # Logique de synchronisation des données
├── models/              # Modèles de données partagés
├── conf/                # Gestion du fichier de configuration (INI)
└── util/                # Fonctions utilitaires
```

---

## 🔧 Stack Technique

| Composant | Technologie |
|------|------|
| Langage | [Go 1.23](https://golang.org) |
| Framework CLI | [cobra](https://github.com/spf13/cobra) |
| Stockage | [bbolt](https://github.com/etcd-io/bbolt) |
| Configuration | [ini.v1](https://gopkg.in/ini.v1) |
| Protocole MCP | [mcp-go](https://github.com/mark3labs/mcp-go) |
| API IA | OpenAI Chat Completions API (compatible) |

---

## 🌐 Traductions

- [English](README.md)
- [简体中文](README.zh-CN.md)
- [日本語](README.ja.md)
- [Español](README.es.md)
- [Português](README.pt.md)

---

## 🤝 Contribuer

Les contributions sont les bienvenues ! Voici comment vous pouvez aider :

1. Forkez le dépôt
2. Créez une branche de fonctionnalité (`git checkout -b feature/amazing-feature`)
3. Commitez vos changements (`git commit -m 'Add amazing feature'`)
4. Poussez vers la branche (`git push origin feature/amazing-feature`)
5. Ouvrez une Pull Request

Pour les changements majeurs, veuillez d'abord ouvrir un issue pour discuter de ce que vous souhaitez modifier.

---

## 📄 Licence

Ce projet est sous licence Apache License 2.0 - voir le fichier [LICENSE](LICENSE) pour plus de détails.

---

## 🙏 Remerciements

- [cobra](https://github.com/spf13/cobra) pour l'excellent framework CLI
- [bbolt](https://github.com/etcd-io/bbolt) pour le stockage key-value embarqué fiable
- [mcp-go](https://github.com/mark3labs/mcp-go) pour le support du protocole MCP
- La communauté open-source

---

<div align="center">

**Fait avec ❤️ par des développeurs qui détestent chercher des connaissances perdues**

</div>
