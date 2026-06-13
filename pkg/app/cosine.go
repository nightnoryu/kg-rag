package app

type Ranker interface {
	Similarity(a, b []float64) float64
}
