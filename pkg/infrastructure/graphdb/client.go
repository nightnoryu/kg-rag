package graphdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
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

func (c *client) RetrieveKnowledge(question string) ([]string, error) {
	searchQuery := strings.ReplaceAll(question, `"`, `\"`)

	sparql := fmt.Sprintf(`
		PREFIX luc: <http://www.ontotext.com/connectors/lucene#>
		PREFIX luc-index: <http://www.ontotext.com/connectors/lucene/instance#kg_index>
		SELECT ?text WHERE {
			?search a luc-index:kg_index ;
			        luc:query "%s" ;
			        luc:entities ?entity .
			?entity <http://example.org/text> ?text .
		} LIMIT 5
	`, searchQuery)

	result, err := c.query(sparql)
	if err != nil {
		return nil, fmt.Errorf("SPARQL query failed: %w", err)
	}

	// Extract ?text bindings from the JSON result.
	// For brevity, a simplified extraction; in production use proper JSON parsing.
	var texts []string
	if bindings, ok := result["results"].(map[string]interface{})["bindings"].([]interface{}); ok {
		for _, b := range bindings {
			bind := b.(map[string]interface{})
			if t, ok := bind["text"].(map[string]interface{})["value"].(string); ok {
				texts = append(texts, t)
			}
		}
	}
	return texts, nil
}

func (c *client) query(sparql string) (map[string]interface{}, error) {
	queryURL := c.endpoint + "?query=" + url.QueryEscape(sparql)
	req, err := http.NewRequest("GET", queryURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/sparql-results+json")
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
