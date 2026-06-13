# Algorithm

1. Entities are extracted from the prompt using LLM
2. Facts regarding these entities are retrieved from GraphDB using a SPARQL query
3. The most relevant facts are selected using comparison of vector embeddings with the original prompt
4. Facts are put before the original prompt in the final prompt to the LLM
5. Final answer generation with references
