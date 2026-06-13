# KG-RAG

Go server that wires a knowledge graph (GraphDB) + LLM (Ollama) behind an OpenAI-compatible REST API.

## Quick start

```
brewkit build
```

Full stack with Docker Compose: `docker compose up` (requires pre-built `bin/rag-server`).

## Codegen — API handlers are generated

- OpenAPI spec: `api/server/ragapi.yaml`
- Generator: `ogen` (config in `api/server/ragapi/.ogen.yml`)
- Generated files: `api/server/ragapi/*_gen.go` (gitignored, never edit)
- Regenerate: `brewkit build generate`
- Implement handlers in `pkg/infrastructure/api/server.go` by filling `restHandler` methods.

## Package layout

```
cmd/rag-server/              — binary entrypoint, env config
api/server/ragapi.yaml       — OpenAPI spec (source of truth for routes)
data/prompts/                — prompt templates (prompts.json, embedded at compile time)
pkg/app/                     — domain interfaces (KGClient, LLMClient, Ranker, AIKnowledgeService)
pkg/infrastructure/          — concrete implementations (graphdb/, llm/, api/, ranker/)
```

## Import order

Enforced by `gci` (configured in `.golangci.yml`). Three sections, separated by blank lines:

1. **Standard library** (`fmt`, `os`, `net/http`, …)
2. **External** (third-party modules like `github.com/gorilla/mux`)
3. **Local** (`rag-server/…`)

`goimports` is also enabled with local prefix `rag-server`.

## Env vars (required at runtime)

| Variable             | Example                                   |
|----------------------|-------------------------------------------|
| `GRAPHDB_ENDPOINT`   | `http://graphdb:7200/repositories/myrepo` |
| `OLLAMA_URL`         | `http://ollama:11434`                     |
| `OLLAMA_MODEL`       | `llama3`                                  |
| `EMBEDDING_MODEL`    | `nomic-embed-text`                        |
| `SERVE_REST_ADDRESS` | `:8080` (default)                         |
| `RAG_TOP_K`          | `3` (default)                             |

Parsed via `kelseyhightower/envconfig` in `cmd/rag-server/config.go`.

## Prompts

Prompt templates live in `data/prompts/prompts.json` and are embedded at compile time via `//go:embed`. Load them with `prompts.LoadPrompts()` — it returns a `map[string]string`. Use constants from `data/prompts/promptsembedder.go` (`prompts.PromptKGAugmentedAnswer`, `prompts.PromptEntityRetrieval`) as map keys. Templates use `fmt.Sprintf`-style `%s` placeholders.

## Lint

See .golangci.yml, 20+ linters enabled. Run linter with `brewkit build check`.

Formatters (`gofmt`, `goimports`, `gci`) are enforced by golangci-lint. Local prefix: `rag-server`.

## Build

Binary output: `bin/rag-server` (gitignored). For local dev:

```
brewkit build rag-server
```

For debug builds (disable inlining): `brewkit build rag-server-debug`

CI/build system uses brewkit (`brewkit.jsonnet`), do not use plain `go build`.

## Dependencies

- GraphDB 10.5.0-free (SPARQL + Lucene full-text search)
- Ollama (streaming LLM via `/api/generate`)

## RAG algorithm

See `docs/algorithm.md`: entity extraction → SPARQL retrieval → embedding ranking → prompt augmentation → answer generation.
