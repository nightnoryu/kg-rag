package app

import "io"

type LLMClient interface {
	ExtractEntities(text string) ([]string, error)
	Generate(prompt string) (string, error)
	GenerateStream(prompt string) io.Reader
	Embed(text string) ([]float64, error)
}
