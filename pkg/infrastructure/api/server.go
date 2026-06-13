package api

import (
	"context"
	"net/http"

	"rag-server/api/server/ragapi"
	"rag-server/pkg/app"

	"github.com/ogen-go/ogen/middleware"
)

func NewAPIServer(
	middlewares []middleware.Middleware,
	aiKnowledgeService app.AIKnowledgeService,
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
	// TODO implement
	panic("implement me")
}

func (h *restHandler) CreateChatCompletion(ctx context.Context, req *ragapi.ChatCompletionRequest) (ragapi.CreateChatCompletionRes, error) {
	// TODO implement
	panic("implement me")
}
