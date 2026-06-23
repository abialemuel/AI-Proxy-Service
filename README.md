# AI Proxy Service v2

A clean, provider-agnostic proxy for Large Language Models.
Switch between OpenAI, Azure OpenAI, Anthropic, and Google Gemini behind a single OpenAI-compatible API — without touching client code.

> Refactored from v1 to use **Clean / Hexagonal Architecture** (ports & adapters), the **Adapter** pattern for each provider, **Strategy + Registry** for runtime model routing, and **DTO/Mapper** separation at every boundary.

---

## Features

- **Multi-provider out of the box**
  - OpenAI (Chat Completions, also works for OpenAI-compatible gateways)
  - Azure OpenAI (deployment-based routing)
  - Anthropic Claude (Messages API)
  - Google Gemini (`generateContent`)
- **Auto model routing** by id prefix, or explicit `provider` hint per request
- **Stateful user chat** with persisted conversation context (MongoDB)
- **Stateless service chat** for backend-to-backend calls (HTTP Basic)
- **Per-user token quota** with rolling window (Redis)
- **Typed domain errors** mapped to correct HTTP codes
- **OpenTelemetry-friendly** structure; APM integration is a thin wrapper away
- **Graceful shutdown**, health/ready endpoints, request IDs

---

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                       cmd/server (composition)                   │
└──────────────────────────────────────────────────────────────────┘
            │                          │                  │
            ▼                          ▼                  ▼
┌──────────────────┐   ┌──────────────────────┐   ┌──────────────────┐
│  adapter/http    │   │  internal/usecase    │   │  internal/ports  │
│  (Echo, DTOs,    │──▶│  ChatService         │──▶│  LLMProvider     │
│   middleware)    │   │  QuotaService        │   │  ConvRepo, Cache │
└──────────────────┘   └──────────────────────┘   └──────────────────┘
                                                            ▲
            ┌──────────────────────────────────────────────┴──────────────┐
            │                  internal/adapter/llm                       │
            │ ┌────────┐ ┌──────────────┐ ┌───────────┐ ┌─────────────┐   │
            │ │ openai │ │ azureopenai  │ │ anthropic │ │   gemini    │   │
            │ └────────┘ └──────────────┘ └───────────┘ └─────────────┘   │
            └────────────────────────────────────────────────────────────┘
            ┌──────────────────────────────────────────────────────────────┐
            │ internal/adapter/{cache/redis, persistence/mongo, auth/...}  │
            └──────────────────────────────────────────────────────────────┘
```

**Why this layout?**

- `internal/domain` — pure types (`Message`, `ChatRequest`, `Usage`). Zero vendor imports.
- `internal/ports` — interfaces the core depends on (LLMProvider, ConversationRepository, Cache).
- `internal/usecase` — application services. Knows domain + ports, nothing else.
- `internal/adapter/...` — concrete implementations of ports (HTTP, Echo handlers, Mongo, Redis, every LLM vendor).
- `pkg/openaicompat` — public OpenAI-compatible wire types, reusable by clients.

**Patterns applied**

| Pattern              | Where                                                            |
| -------------------- | ---------------------------------------------------------------- |
| Adapter              | `internal/adapter/llm/*` — each provider adapts its native SDK   |
| Strategy + Registry  | `internal/adapter/llm/registry.go` resolves model → provider     |
| Repository           | `internal/adapter/persistence/mongo/conversation.go`             |
| Dependency Injection | Constructor-based wiring in `cmd/server/main.go`                 |
| DTO / Mapper         | `internal/adapter/http/dto` ↔ `internal/domain`                  |
| Domain Errors        | `internal/pkg/errors` mapped to status in `handler/errors.go`    |

---

## Adding a new provider

1. Create `internal/adapter/llm/<name>/adapter.go` implementing `ports.LLMProvider`.
2. Create `internal/adapter/llm/<name>/mapper.go` converting domain ↔ vendor.
3. Add a case in `registerProviders` in `cmd/server/main.go`.
4. Add an entry under `providers:` in `config.yaml`.

No other code needs to change. Routing is driven by `Supports(model)` and the optional `modelPrefix` config field.

---

## API

### `POST /v1/prompt` — user chat (Bearer JWT)

```json
{
  "model": "gpt-4o",
  "messages": [
    { "role": "system",  "content": "You are concise." },
    { "role": "user",    "content": "Hello!" }
  ],
  "temperature": 0.2
}
```

To force a provider regardless of the model id:

```json
{ "model": "my-custom-id", "provider": "anthropic", "messages": [...] }
```

Multimodal (image) input is supported via the OpenAI-style content array:

```json
{ "role": "user", "content": [
    { "type": "text", "text": "Describe this image" },
    { "type": "image_url", "image_url": { "url": "https://..." } }
] }
```

### `POST /v1/prompt/internal` — service chat (HTTP Basic)

Stateless. Otherwise identical body.

### `POST /v1/prompt/new` — clear conversation context for the current user.

### `GET /v1/providers` — list enabled providers.

---

## Run locally

```sh
make tidy
make run
```

Or with Docker Compose (Redis + Mongo + service):

```sh
docker compose up --build
```

Configuration lives in `config.yaml`. Override via env vars by replacing dots with underscores (e.g. `PROVIDERS_OPENAI_APIKEY`).

---

## License

See `LICENSE`.
