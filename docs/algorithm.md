# RAG Algorithm

The `AIKnowledgeService` (`pkg/app/aiknowledgeservice.go`) orchestrates a five-step retrieval-augmented generation pipeline. Both `GenerateAnswer` (blocking) and `GenerateAnswerStream` (SSE) follow the same augmentation logic, differing only in the final LLM call.

## Steps

### 1. Entity Extraction

The service calls `llmClient.ExtractEntities(prompt)` to identify key entities (people, organizations, locations, technical terms, concepts) from the user's question.

The LLM is prompted with the `entity-retrieval` template (`data/prompts/prompts.json`):
```
Extract the key entities (people, organizations, locations, dates, technical terms, concepts) from the following text.
Return ONLY a JSON array of strings, nothing else. No markdown, no explanation.
Retrieved entities must be ONLY in Russian language.

Text: <prompt>

Entities:
```

The LLM response is parsed as a JSON array. If extraction fails (parsing error, network issue), the service falls back to using the entire prompt as a single search entity.

**Example:**
```
Input:  "What does the Kubernetes scheduler do and how does it interact with etcd?"
Output: ["Kubernetes", "scheduler", "etcd"]
```

### 2. Knowledge Retrieval

For each extracted entity, the service calls `kgClient.RetrieveKnowledge(entity)`. The GraphDB client (`pkg/infrastructure/graphdb/client.go`) issues a SPARQL query that performs case-insensitive exact label matching via `rdfs:label`.

The SPARQL query for each entity:
```sparql
PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
PREFIX : <http://example.org/ru/ontology/>

SELECT ?property ?displayValue WHERE {
    ?entity rdfs:label ?label .
    FILTER(LCASE(STR(?label)) = "<entity>")
    ?entity ?property ?value .
    FILTER(!isBlank(?value))
    OPTIONAL { ?value rdfs:label ?valueLabel . }
    BIND(COALESCE(?valueLabel, ?value) AS ?displayValue)
}
```

Each result returns a property and display value (using `rdfs:label` when available, falling back to the raw value). Properties with the `http://example.org/ru/ontology/` prefix are shortened to their local name. Results from all entities are collected as triples `(<entity>, <property>, <displayValue>)`. Per-entity failures are logged but do not abort the pipeline.

**Example:**
```
Entity "Kubernetes"  -> [(Kubernetes, тип, программное обеспечение),
                         (Kubernetes, разработчик, Google)]
Entity "scheduler"   -> [(scheduler, тип, компонент),
                         (scheduler, описание, назначает поды на ноды)]
Entity "etcd"        -> [(etcd, тип, хранилище ключей),
                         (etcd, использует, Kubernetes)]

Retrieved facts: 6 triples
```

### 3. Embedding-Based Ranking

If facts were retrieved, the service ranks them by relevance to the original question:

1. The original question is embedded via `llmClient.Embed(question)` (Ollama `/api/embed` endpoint, using the `EMBEDDING_MODEL`, e.g., `nomic-embed-text`).
2. Each fact is embedded individually using the same model.
3. Cosine similarity is computed between the question vector and each fact vector (`ranker.Similarity`, implemented in `pkg/infrastructure/ranker/cosine.go`):

   ```
   similarity(a, b) = dot(a, b) / (||a|| * ||b||)
   ```

4. Facts are sorted in descending order of similarity score. Per-fact embedding failures are skipped with a log warning.

**Example:**
```
Question embedding:  [-0.02, 0.15, -0.08, ...]  (768-dim for nomic-embed-text)

Fact                                      Cosine Score
---------------------------------------------------------
(scheduler, описание, назначает поды...)  0.847
(scheduler, тип, компонент)              0.792
(etcd, тип, хранилище ключей)            0.651
(etcd, использует, Kubernetes)           0.618
(Kubernetes, тип, программное обесп...)   0.534
(Kubernetes, разработчик, Google)        0.489
```

### 4. Prompt Augmentation

The top-K facts (controlled by `RAG_TOP_K`, default 3) are formatted as bullet points and injected into the `kg-augmented-answer` prompt template:

```
You are a knowledgeable assistant. Use the following facts from the knowledge base to answer the question.
If the are no facts or they don't contain enough information, use your general knowledge.
Answer ONLY in Russian language.

Relevant facts:
- <fact_1>
- <fact_2>
- <fact_3>

Question: <original_prompt>

Answer:
```

If no facts were retrieved, the original prompt is sent as-is (no augmentation).

**Example (topK=3):**
```
You are a knowledgeable assistant. Use the following facts from the knowledge base to answer the question.
If the are no facts or they don't contain enough information, use your general knowledge.
Answer ONLY in Russian language.

Relevant facts:
- (scheduler, тип, компонент)
- (scheduler, описание, назначает поды на ноды)
- (etcd, тип, хранилище ключей)

Question: Что делает планировщик в Kubernetes и как он взаимодействует с etcd?

Answer:
```

### 5. Answer Generation

The augmented prompt is sent to the LLM via one of two paths:

| Method                 | LLM Call                     | Response Type     |
|------------------------|------------------------------|-------------------|
| `GenerateAnswer`       | `llmClient.Generate()`       | `string`          |
| `GenerateAnswerStream` | `llmClient.GenerateStream()` | `io.Reader` (SSE) |

The Ollama client (`pkg/infrastructure/llm/client.go`) always calls `/api/generate` with `stream: true`. For `Generate`, the streaming response is consumed client-side and assembled into a single string. For `GenerateStream`, the raw Ollama stream is re-wrapped as an OpenAI-compatible Server-Sent Events stream.

## Flow Diagram

```
User Prompt
    │
    ▼
┌──────────────────────┐
│  Entity Extraction   │  LLM → JSON array of entities (Russian)
│  (llmClient)         │  Fallback: [prompt] on error
└────────┬─────────────┘
         │  ["Kubernetes", "scheduler", "etcd"]
         ▼
┌──────────────────────┐
│  Knowledge Retrieval │  SPARQL label matching per entity
│  (kgClient)          │  Returns triples (entity, property, value)
└────────┬─────────────┘
         │  6 triples
         ▼
┌──────────────────────┐
│  Embedding & Ranking │  Embed question + each fact triple
│  (llmClient + ranker)│  Cosine similarity, descending sort
└────────┬─────────────┘
         │  scored: [0.847, 0.792, 0.651, ...]
         ▼
┌──────────────────────┐
│  Prompt Augmentation │  Top-K facts → bullet list
│  (service)           │  Inject into answer prompt template (Russian)
└────────┬─────────────┘
         │  Augmented prompt string
         ▼
┌──────────────────────┐
│  Answer Generation   │  LLM completion (blocking or streaming)
│  (llmClient)         │  Response in Russian
└──────────────────────┘
```

## Error Handling

| Stage                  | Behavior on Error                              |
|------------------------|------------------------------------------------|
| Entity extraction      | Fallback: use full prompt as single entity     |
| Per-entity retrieval   | Skip entity, log warning, continue with others |
| Per-fact embedding     | Skip fact, log warning, continue with others   |
| Question embedding     | Abort: return error (cannot rank without it)   |
| Knowledge retrieval    | Abort if all entities fail                     |
| Answer generation      | Error propagated to caller                     |

## Configuration

| Variable             | Default | Used In                    |
|----------------------|---------|----------------------------|
| `RAG_TOP_K`          | `3`     | Step 4 (fact limit)        |
| `EMBEDDING_MODEL`    | —       | Step 3 (embedding model)   |
| `OLLAMA_MODEL`       | —       | Steps 1, 5 (LLM model)     |
| `GRAPHDB_ENDPOINT`   | —       | Step 2 (SPARQL endpoint)   |
