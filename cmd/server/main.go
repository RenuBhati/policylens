package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"policylens/internal/server"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	budget := 40
	if value := os.Getenv("AI_DAILY_CALL_LIMIT"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			slog.Error("AI_DAILY_CALL_LIMIT must be between 1 and 100")
			os.Exit(1)
		}
		budget = parsed
	}
	s, err := server.New(server.Config{EnginePath: os.Getenv("KYVERNO_BIN"), OllamaURL: os.Getenv("OLLAMA_URL"), OllamaModel: os.Getenv("OLLAMA_MODEL"), AIProvider: os.Getenv("AI_PROVIDER"), CFModel: os.Getenv("CF_AI_MODEL"), CFPath: os.Getenv("CF_BIN"), ModelBudget: budget})
	if err != nil {
		slog.Error("startup failed", "error", err)
		os.Exit(1)
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	srv := &http.Server{Addr: addr, Handler: s.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 75 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdown)
	}()
	slog.Info("PolicyLens started", "address", addr, "engine_available", s.Engine.Path != "", "engine_version", s.Engine.Version, "llm_enabled", s.Provider != nil)
	if err = srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
