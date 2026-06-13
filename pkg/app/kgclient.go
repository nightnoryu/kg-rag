package app

type KGClient interface {
	RetrieveKnowledge(question string) ([]string, error)
}
