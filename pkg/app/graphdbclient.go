package app

type GraphDBClient interface {
	Query(sparql string) (map[string]interface{}, error)
}
