package prompts

import (
	"embed"
	"encoding/json"
	"fmt"
)

const (
	PromptKGAugmentedAnswer = "kg-augmented-answer"
	PromptEntityRetrieval   = "entity-retrieval"
)

//go:embed prompts.json
var fs embed.FS

func LoadPrompts() (map[string]string, error) {
	promptsData, err := fs.ReadFile("prompts.json")
	if err != nil {
		return nil, err
	}
	var configs map[string]string
	err = json.Unmarshal(promptsData, &configs)
	if err != nil {
		return nil, fmt.Errorf("error loading prompts: %v", err)
	}
	return configs, nil
}
