package app

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

type AIKnowledgeService interface {
	GenerateAnswer(prompt string) (string, error)
	GenerateAnswerStream(prompt string) io.Reader
}

func NewAIKnowledgeService(
	kgClient KGClient,
	llmClient LLMClient,
	ranker Ranker,
	topK int,
	answerPrompt string,
) AIKnowledgeService {
	return &service{
		kgClient:     kgClient,
		llmClient:    llmClient,
		ranker:       ranker,
		topK:         topK,
		answerPrompt: answerPrompt,
	}
}

type service struct {
	kgClient     KGClient
	llmClient    LLMClient
	ranker       Ranker
	topK         int
	answerPrompt string
}

type scoredFact struct {
	text  string
	score float64
}

func (s *service) GenerateAnswer(prompt string) (string, error) {
	augmented, err := s.buildAugmentedPrompt(prompt)
	if err != nil {
		return "", err
	}
	return s.llmClient.Generate(augmented)
}

func (s *service) GenerateAnswerStream(prompt string) io.Reader {
	augmented, err := s.buildAugmentedPrompt(prompt)
	if err != nil {
		pr, pw := io.Pipe()
		pw.CloseWithError(err)
		return pr
	}
	return s.llmClient.GenerateStream(augmented)
}

func (s *service) buildAugmentedPrompt(prompt string) (string, error) {
	entities, err := s.llmClient.ExtractEntities(prompt)
	if err != nil {
		entities = []string{prompt}
	}

	facts, err := s.retrieveFacts(entities)
	if err != nil {
		return "", fmt.Errorf("retrieve facts: %w", err)
	}

	if len(facts) == 0 {
		return prompt, nil
	}

	scored, err := s.rankFacts(prompt, facts)
	if err != nil {
		return "", fmt.Errorf("rank facts: %w", err)
	}

	limit := s.topK
	if limit > len(scored) {
		limit = len(scored)
	}

	var factLines []string
	for i := 0; i < limit; i++ {
		factLines = append(factLines, fmt.Sprintf("- %s", scored[i].text))
	}

	context := strings.Join(factLines, "\n")
	return fmt.Sprintf(s.answerPrompt, context, prompt), nil
}

func (s *service) retrieveFacts(entities []string) ([]string, error) {
	seen := make(map[string]bool)
	var facts []string

	for _, entity := range entities {
		results, err := s.kgClient.RetrieveKnowledge(entity)
		if err != nil {
			continue
		}
		for _, fact := range results {
			if !seen[fact] {
				seen[fact] = true
				facts = append(facts, fact)
			}
		}
	}

	return facts, nil
}

func (s *service) rankFacts(question string, facts []string) ([]scoredFact, error) {
	qEmbed, err := s.llmClient.Embed(question)
	if err != nil {
		return nil, fmt.Errorf("embed question: %w", err)
	}

	scored := make([]scoredFact, 0, len(facts))
	for _, fact := range facts {
		fEmbed, err := s.llmClient.Embed(fact)
		if err != nil {
			continue
		}
		scored = append(scored, scoredFact{
			text:  fact,
			score: s.ranker.Similarity(qEmbed, fEmbed),
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	return scored, nil
}
