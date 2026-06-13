package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"rag-server/data/prompts"
	"rag-server/pkg/app"
	"rag-server/pkg/infrastructure/api"
	"rag-server/pkg/infrastructure/graphdb"
	"rag-server/pkg/infrastructure/llm"
	"rag-server/pkg/infrastructure/ranker"
)

func main() {
	ctx := context.Background()

	logger := initLogger()
	cnf, err := parseEnv()
	if err != nil {
		logger.Fatal(err)
	}

	serveHTTP(ctx, cnf, logger)
}

func serveHTTP(ctx context.Context, cnf *config, logger *log.Logger) {
	ctx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	ctx = listenOSKillSignals(ctx)

	kgClient := graphdb.NewClient(cnf.GraphDBEndpoint)
	cosineRanker := ranker.NewCosineRanker()

	promptsMap, err := prompts.LoadPrompts()
	if err != nil {
		logger.Fatal(fmt.Errorf("load prompts: %w", err))
	}

	entityPrompt, ok := promptsMap[prompts.PromptEntityRetrieval]
	if !ok {
		logger.Fatal(fmt.Errorf("missing prompt: %s", prompts.PromptEntityRetrieval))
	}

	answerPrompt, ok := promptsMap[prompts.PromptKGAugmentedAnswer]
	if !ok {
		logger.Fatal(fmt.Errorf("missing prompt: %s", prompts.PromptKGAugmentedAnswer))
	}

	llmClient := llm.NewClient(cnf.OllamaURL, cnf.OllamaModel, cnf.EmbeddingModel, entityPrompt)
	aiKnowledgeService := app.NewAIKnowledgeService(kgClient, llmClient, cosineRanker, cnf.RagTopK, answerPrompt)

	router := mux.NewRouter()
	router.HandleFunc("/resilience/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, http.StatusText(http.StatusOK))
	})

	apiServer, err := api.NewAPIServer(nil, aiKnowledgeService, llmClient, cnf.OllamaURL, cnf.OllamaModel)
	if err != nil {
		logger.Fatal(err)
	}

	router.PathPrefix("/v1").Handler(apiServer)

	httpServer := &http.Server{
		Handler:           router,
		Addr:              cnf.ServeRESTAddress,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       time.Hour,
		WriteTimeout:      time.Hour,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Printf("server shutdown error: %v", err)
		}
	}()

	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Fatal(err)
	}
}

func initLogger() *log.Logger {
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lmicroseconds)
	return logger
}

func listenOSKillSignals(ctx context.Context) context.Context {
	var cancelFunc context.CancelFunc
	ctx, cancelFunc = context.WithCancel(ctx)
	go func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
		select {
		case <-ch:
			cancelFunc()
		case <-ctx.Done():
			signal.Reset()
			return
		}
	}()
	return ctx
}
