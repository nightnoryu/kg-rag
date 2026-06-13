package cosine

import (
	"math"

	"rag-server/pkg/app"
)

type cosineRanker struct{}

func NewCosineRanker() app.CosineRanker {
	return &cosineRanker{}
}

func (r *cosineRanker) Similarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	denominator := math.Sqrt(normA) * math.Sqrt(normB)
	if denominator == 0 {
		return 0
	}
	return dot / denominator
}
