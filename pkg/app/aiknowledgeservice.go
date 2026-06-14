package app

import (
	"fmt"
	"io"
	"log"
	"sort"
	"strings"
	"time"
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
	logger *log.Logger,
) AIKnowledgeService {
	return &service{
		kgClient:     kgClient,
		llmClient:    llmClient,
		ranker:       ranker,
		topK:         topK,
		answerPrompt: answerPrompt,
		logger:       logger,
	}
}

type service struct {
	kgClient     KGClient
	llmClient    LLMClient
	ranker       Ranker
	topK         int
	answerPrompt string
	logger       *log.Logger
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
	start := time.Now()
	s.logger.Printf("rag: generating answer")
	answer, err := s.llmClient.Generate(augmented)
	s.logger.Printf("rag: answer generated in %v (error: %v)", time.Since(start), err)
	return answer, err
}

func (s *service) GenerateAnswerStream(prompt string) io.Reader {
	augmented, err := s.buildAugmentedPrompt(prompt)
	if err != nil {
		pr, pw := io.Pipe()
		pw.CloseWithError(err)
		return pr
	}
	s.logger.Printf("rag: generating streaming answer")
	return s.llmClient.GenerateStream(augmented)
}

func (s *service) buildAugmentedPrompt(prompt string) (string, error) {
	start := time.Now()
	s.logger.Printf("rag: start building augmented prompt")

	entities, err := s.llmClient.ExtractEntities(prompt)
	if err != nil {
		s.logger.Printf("rag: entity extraction failed (%v), falling back to full prompt", err)
		entities = []string{prompt}
	}
	s.logger.Printf("rag: extracted %d entities: %v (%v)", len(entities), entities, time.Since(start))

	facts, err := s.retrieveFacts(entities)
	if err != nil {
		return "", fmt.Errorf("retrieve facts: %w", err)
	}
	s.logger.Printf("rag: retrieved %d unique facts (%v)", len(facts), time.Since(start))

	if len(facts) == 0 {
		s.logger.Printf("rag: no facts found, returning original prompt (%v)", time.Since(start))
		return prompt, nil
	}

	scored, err := s.rankFacts(prompt, facts)
	if err != nil {
		return "", fmt.Errorf("rank facts: %w", err)
	}
	s.logger.Printf("rag: ranked %d facts, top score: %.4f (%v)", len(scored), scored[0].score, time.Since(start))

	limit := s.topK
	if limit > len(scored) {
		limit = len(scored)
	}

	var factLines []string
	for i := 0; i < limit; i++ {
		factLines = append(factLines, fmt.Sprintf("- %s", scored[i].text))
	}

	context := strings.Join(factLines, "\n")
	s.logger.Printf("rag: augmented prompt ready, %d facts included (%v)", limit, time.Since(start))
	return fmt.Sprintf(s.answerPrompt, context, prompt), nil
}

func (s *service) retrieveFacts(entities []string) ([]string, error) {
	var facts []string

	for _, entity := range entities {
		results, err := s.kgClient.RetrieveKnowledge(entity)
		if err != nil {
			s.logger.Printf("rag: failed to retrieve facts for entity %q: %v", entity, err)
			continue
		}
		s.logger.Printf("rag: entity %q -> %d facts", entity, len(results))
		for _, fact := range results {
			facts = append(facts, fmt.Sprintf("(%s, %s)", fact.Property, fact.Value))
		}
	}

	return facts, nil
}

func (s *service) rankFacts(question string, facts []string) ([]scoredFact, error) {
	s.logger.Printf("rag: embedding question for ranking")
	qEmbed, err := s.llmClient.Embed(question)
	if err != nil {
		return nil, fmt.Errorf("embed question: %w", err)
	}

	scored := make([]scoredFact, 0, len(facts))
	for _, fact := range facts {
		fEmbed, err := s.llmClient.Embed(fact)
		if err != nil {
			s.logger.Printf("rag: failed to embed fact: %v", err)
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

	s.logger.Printf("rag: ranked %d facts, scores: %v", len(scored), func() []float64 {
		scores := make([]float64, len(scored))
		for i, sf := range scored {
			scores[i] = sf.score
		}
		return scores
	}())

	return scored, nil
}
