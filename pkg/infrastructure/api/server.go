package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ogen-go/ogen/middleware"

	"rag-server/api/server/ragapi"
	"rag-server/pkg/app"
)

func NewAPIServer(
	middlewares []middleware.Middleware,
	aiKnowledgeService app.AIKnowledgeService,
	llmClient app.LLMClient,
	ollamaURL string,
	ollamaModel string,
) (http.Handler, error) {
	apiHandler := newRESTHandler(aiKnowledgeService, llmClient, ollamaURL, ollamaModel)
	return ragapi.NewServer(
		apiHandler,
		ragapi.WithMiddleware(middlewares...),
	)
}

func newRESTHandler(
	aiKnowledgeService app.AIKnowledgeService,
	llmClient app.LLMClient,
	ollamaURL string,
	ollamaModel string,
) ragapi.Handler {
	return &restHandler{
		aiKnowledgeService: aiKnowledgeService,
		llmClient:          llmClient,
		ollamaURL:          strings.TrimRight(ollamaURL, "/"),
		ollamaModel:        ollamaModel,
	}
}

type restHandler struct {
	aiKnowledgeService app.AIKnowledgeService
	llmClient          app.LLMClient
	ollamaURL          string
	ollamaModel        string
}

func (h *restHandler) ListModels(_ context.Context) (*ragapi.ModelsResponse, error) {
	ragModel := ragapi.Model{
		ID:     ragapi.NewOptString("rag-knowledge-graph"),
		Object: ragapi.NewOptModelObject(ragapi.ModelObjectModel),
	}

	return &ragapi.ModelsResponse{
		Object: ragapi.NewOptModelsResponseObject(ragapi.ModelsResponseObjectList),
		Data:   []ragapi.Model{ragModel},
	}, nil
}

func (h *restHandler) CreateChatCompletion(_ context.Context, req *ragapi.ChatCompletionRequest) (ragapi.CreateChatCompletionRes, error) {
	prompt := extractUserMessage(req.Messages)
	if prompt == "" {
		return nil, fmt.Errorf("no user message found")
	}

	if req.Stream.Or(false) {
		return h.handleStreaming(prompt), nil
	}

	answer, err := h.aiKnowledgeService.GenerateAnswer(prompt)
	if err != nil {
		return nil, err
	}

	id, _ := generateID()
	return &ragapi.ChatCompletionResponse{
		ID:     ragapi.NewOptString(id),
		Object: ragapi.NewOptChatCompletionResponseObject(ragapi.ChatCompletionResponseObjectChatCompletion),
		Choices: []ragapi.Choice{
			{
				Index:        ragapi.NewOptInt(0),
				Message:      ragapi.NewOptMessage(ragapi.Message{Role: ragapi.MessageRoleAssistant, Content: answer}),
				FinishReason: ragapi.NewOptString("stop"),
			},
		},
	}, nil
}

func (h *restHandler) handleStreaming(prompt string) ragapi.CreateChatCompletionRes {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		stream := h.aiKnowledgeService.GenerateAnswerStream(prompt)
		if _, err := io.Copy(pw, stream); err != nil {
			pw.CloseWithError(err)
		}
	}()

	return &ragapi.CreateChatCompletionOKTextEventStream{
		Data: pr,
	}
}

func extractUserMessage(messages []ragapi.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == ragapi.MessageRoleUser {
			return messages[i].Content
		}
	}
	return ""
}

func generateID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "chatcmpl-" + hex.EncodeToString(b), nil
}
