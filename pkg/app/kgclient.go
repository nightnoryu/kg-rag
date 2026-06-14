package app

type Fact struct {
	Property string
	Value    string
}

type KGClient interface {
	RetrieveKnowledge(question string) ([]Fact, error)
}
