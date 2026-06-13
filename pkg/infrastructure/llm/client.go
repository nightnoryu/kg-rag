package llm

import (
	"bufio"
	"encoding/json"
	"net/http"
	"strings"
)

func NewClient(baseURL, model string) *client {
	return &client{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  http.DefaultClient,
	}
}

type client struct {
	baseURL string
	model   string
	client  *http.Client
}

func (c *client) Generate(prompt string) (*http.Response, error) {
	body := map[string]interface{}{
		"model":  c.model,
		"prompt": prompt,
		"stream": true, // always stream
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", c.baseURL+"/api/generate", strings.NewReader(string(b)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.client.Do(req)
}

// ReadStream reads Ollama's stream, yielding each generated piece of text.
func ReadStream(resp *http.Response, out chan<- string) {
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var data struct {
			Response string `json:"response"`
			Done     bool   `json:"done"`
		}
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			continue
		}
		if data.Response != "" {
			out <- data.Response
		}
		if data.Done {
			close(out)
			return
		}
	}
	close(out)
}
