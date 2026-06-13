package app

type CosineRanker interface {
	Similarity(a, b []float64) float64
}
