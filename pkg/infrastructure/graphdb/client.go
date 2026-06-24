package graphdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"rag-server/pkg/app"
)

func NewClient(endpoint string) app.KGClient {
	return &client{
		endpoint: endpoint,
		client:   http.DefaultClient,
	}
}

type client struct {
	endpoint string
	client   *http.Client
}

func (c *client) RetrieveKnowledge(entityName string) ([]app.Fact, error) {
	searchQuery := escapeSPARQLString(strings.ToLower(entityName))

	sparql := fmt.Sprintf(`
		PREFIX rdfs: <http://www.w3.org/2000/01/rdf-schema#>
		PREFIX : <http://example.org/ru/ontology/>
		
		SELECT ?property ?displayValue WHERE {
		    ?entity rdfs:label ?label .
		    FILTER(LCASE(STR(?label)) = "%s")
		    ?entity ?property ?value .
		    FILTER(!isBlank(?value))
			OPTIONAL { ?value rdfs:label ?valueLabel . }
			BIND(COALESCE(?valueLabel, ?value) AS ?displayValue)
		}
	`, searchQuery)

	result, err := c.query(sparql)
	if err != nil {
		return nil, fmt.Errorf("SPARQL query failed: %w", err)
	}

	results, ok := result["results"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no results found")
	}

	bindingsRaw, ok := results["bindings"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("bindings is not an array")
	}

	// nolint:prealloc
	var facts []app.Fact
	for _, b := range bindingsRaw {
		bind, ok := b.(map[string]interface{})
		if !ok {
			continue
		}

		propMap, ok := bind["property"].(map[string]interface{})
		if !ok {
			continue
		}
		property, ok := propMap["value"].(string)
		if !ok {
			continue
		}

		valMap, ok := bind["displayValue"].(map[string]interface{})
		if !ok {
			continue
		}
		displayValue, ok := valMap["value"].(string)
		if !ok {
			continue
		}

		facts = append(facts, app.Fact{
			Property: strings.ReplaceAll(property, "http://example.org/ru/ontology/", ""),
			Value:    strings.ReplaceAll(displayValue, "http://example.org/ru/ontology/", ""),
		})
	}

	return facts, nil
}

func (c *client) query(sparql string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", c.endpoint, http.NoBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/sparql-results+json")
	q := req.URL.Query()
	q.Add("query", sparql)
	req.URL.RawQuery = q.Encode()

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	return result, err
}

func escapeSPARQLString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}
