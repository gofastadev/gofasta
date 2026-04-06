package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/healtronlabs/gofasta/configs"
	"github.com/healtronlabs/gofasta/pkg/logger"
)

func serve() {
	cfg, err := configs.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	logger.NewLogger(&cfg.Log)

	graphqlResolver, apiRouter := setupAndInitializeDb(cfg)

	mux := http.NewServeMux()
	mux.Handle(cfg.GraphQL.PlaygroundRoute, playground.Handler("GraphQL playground", cfg.GraphQL.GeneralRoute))
	mux.Handle(cfg.GraphQL.GeneralRoute, graphqlResolver)
	mux.Handle("/", apiRouter)

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: mux,
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("server starting", "port", cfg.Server.Port, "playground", cfg.GraphQL.PlaygroundRoute)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down gracefully")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}

	slog.Info("server stopped")
}
