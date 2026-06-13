package app

type LLMClient interface {
	Generate(prompt string) (string, error)
}
