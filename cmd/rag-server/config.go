package main

import (
	"fmt"

	"github.com/kelseyhightower/envconfig"
)

func parseEnv() (*config, error) {
	c := new(config)
	if err := envconfig.Process("", c); err != nil {
		return nil, fmt.Errorf("failed to parse env: %w", err)
	}
	return c, nil
}

type config struct {
	ServeRESTAddress string `envconfig:"serve_rest_address" default:":8080"`

	GraphDBEndpoint string `envconfig:"graphdb_endpoint"`

	OllamaURL      string `envconfig:"ollama_url"`
	OllamaModel    string `envconfig:"ollama_model"`
	EmbeddingModel string `envconfig:"embedding_model"`
	RagTopK        int    `envconfig:"rag_top_k" default:"3"`
}
