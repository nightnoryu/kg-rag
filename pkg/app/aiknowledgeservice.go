package app

type AIKnowledgeService interface {
	GenerateAnswer(prompt string) (string, error)
}

func NewAIKnowledgeService(
	kgClient KGClient,
	llmClient LLMClient,
) AIKnowledgeService {
	return &service{
		kgClient:  kgClient,
		llmClient: llmClient,
	}
}

type service struct {
	kgClient  KGClient
	llmClient LLMClient
}

func (s *service) GenerateAnswer(prompt string) (string, error) {
	// TODO implement
	panic("implement me")
}
