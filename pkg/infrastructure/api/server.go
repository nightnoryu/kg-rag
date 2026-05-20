package api

import (
	"context"
	"io"
	"net/http"
	"sync"

	"rag-server/api/server/ragapi"

	"github.com/ogen-go/ogen/middleware"
)

func NewAPIServer(
	middlewares []middleware.Middleware,
) (http.Handler, error) {
	apiHandler := newRESTHandler()
	return ragapi.NewServer(
		apiHandler,
		ragapi.WithMiddleware(middlewares...),
	)
}

func newRESTHandler() ragapi.Handler {
	return &restHandler{}
}

type restHandler struct {
}

func (h *restHandler) ListModels(_ context.Context) (*ragapi.ModelsResponse, error) {
	return &ragapi.ModelsResponse{
		Object: ragapi.NewOptModelsResponseObject(ragapi.ModelsResponseObjectList),
		Data: []ragapi.Model{
			{ID: ragapi.NewOptString("llama3"), Object: ragapi.NewOptModelObject(ragapi.ModelObjectModel)},
		},
	}, nil
}

func (h *restHandler) CreateChatCompletion(ctx context.Context, req *ragapi.ChatCompletionRequest) (ragapi.CreateChatCompletionRes, error) {
	var userMessage string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			userMessage = req.Messages[i].Content
			break
		}
	}
	if userMessage == "" {
		return &ragapi.ChatCompletionResponse{
			ID:      ragapi.NewOptString("rag-empty"),
			Object:  ragapi.NewOptChatCompletionResponseObject(ragapi.ChatCompletionResponseObjectChatCompletion),
			Choices: []ragapi.Choice{},
		}, nil
	}
}

// sseStream wraps the Ollama response and formats it as SSE chunks.
type sseStream struct {
	resp *http.Response
	once sync.Once
	done chan struct{}
	err  error
	body string
}

func (s *sseStream) Read(p []byte) (n int, err error) {
	// Build SSE data on‑the‑fly from the Ollama stream
	// ... (implementation details omitted for brevity)
	return 0, io.EOF
}

func (s *sseStream) Close() error {
	s.resp.Body.Close()
	return nil
}
