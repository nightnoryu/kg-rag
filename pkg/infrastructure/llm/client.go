package llm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"rag-server/pkg/app"
)

func NewClient(baseURL, model, embeddingModel string) app.LLMClient {
	return &client{
		baseURL:        strings.TrimRight(baseURL, "/"),
		model:          model,
		embeddingModel: embeddingModel,
		client:         http.DefaultClient,
	}
}

type client struct {
	baseURL        string
	model          string
	embeddingModel string
	client         *http.Client
}

func (c *client) ExtractEntities(text string) ([]string, error) {
	prompt := fmt.Sprintf(`Extract the key entities (people, organizations, locations, dates, technical terms, concepts) from the following text.
Return ONLY a JSON array of strings, nothing else. No markdown, no explanation.

Text: %s

Entities:`, text)

	out, err := c.GenerateString(prompt)
	if err != nil {
		return nil, err
	}

	out = strings.TrimSpace(out)
	if strings.HasPrefix(out, "```") {
		if idx := strings.Index(out, "["); idx != -1 {
			out = out[idx:]
		}
		if idx := strings.LastIndex(out, "]"); idx != -1 {
			out = out[:idx+1]
		}
	}

	var entities []string
	if err := json.Unmarshal([]byte(out), &entities); err != nil {
		return nil, fmt.Errorf("parse entities: %w", err)
	}
	return entities, nil
}

func (c *client) Generate(prompt string) (string, error) {
	return c.GenerateString(prompt)
}

func (c *client) GenerateString(prompt string) (string, error) {
	resp, err := c.doGenerate(prompt)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var sb strings.Builder
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
			sb.WriteString(data.Response)
		}
		if data.Done {
			break
		}
	}
	return sb.String(), nil
}

func (c *client) GenerateStream(prompt string) io.Reader {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		resp, err := c.doGenerate(prompt)
		if err != nil {
			pw.CloseWithError(err)
			return
		}
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
				fmt.Fprintf(pw, "data: %s\n\n", data.Response)
			}
			if data.Done {
				break
			}
		}
	}()

	return pr
}

func (c *client) Embed(text string) ([]float64, error) {
	body := map[string]interface{}{
		"model": c.embeddingModel,
		"input": text,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal embed request: %w", err)
	}
	req, err := http.NewRequest("POST", c.baseURL+"/api/embeddings", strings.NewReader(string(b)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Embedding []float64 `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode embed response: %w", err)
	}
	return result.Embedding, nil
}

//nolint:bodyclose // caller is responsible for closing resp.Body
func (c *client) doGenerate(prompt string) (*http.Response, error) {
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
