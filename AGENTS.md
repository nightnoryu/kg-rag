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
cmd/rag-server/          — binary entrypoint, env config
api/server/ragapi.yaml   — OpenAPI spec (source of truth for routes)
pkg/app/                 — domain interfaces (KGClient, LLMClient, AIKnowledgeService)
pkg/infrastructure/      — concrete implementations (graphdb/, llm/, api/)
```

## Env vars (required at runtime)

| Variable             | Example                                |
|----------------------|----------------------------------------|
| `GRAPHDB_ENDPOINT`   | `http://graphdb:7200/repositories/rag` |
| `OLLAMA_URL`         | `http://ollama:11434`                  |
| `OLLAMA_MODEL`       | `llama3`                               |
| `SERVE_REST_ADDRESS` | `:8080` (default)                      |

Parsed via `kelseyhightower/envconfig` in `cmd/rag-server/config.go`.

## Lint and format

```
brewkit build check          # lint (see .golangci.yml, 20+ linters enabled)
```

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
- No tests exist yet.

## RAG algorithm

See `docs/algorithm.md`: entity extraction → SPARQL retrieval → embedding ranking → prompt augmentation → answer generation.
