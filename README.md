# KG-RAG

Simple KG-RAG implementation in Go using GraphDB and local inference via ollama.

## How to run

Prerequisites:

- mise
- docker
- docker compose

Build with mise:

```shell
mise run
```

Start the project with prepared script:

```shell
bin/kg-rag-up
```

Download the required models:

```shell
docker exec -it ollama ollama pull llama3.1 nomic-embed-text
```

After that, the UI will be available at http://localhost:3080

Manage the project with scripts:

```shell
# Stop the project
bin/kg-rag-down

# Check containers
bin/kg-rag-compose ps
```

## Architecture

- [RAG Algorithm](docs/algorithm.md)
- [C4](docs/c4.md)
- [Sequence](docs/sequence.md)
- [UML](docs/uml.md)
