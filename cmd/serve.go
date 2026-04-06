package cmd

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/healtronlabs/gofasta/app"
	"github.com/healtronlabs/gofasta/app/di"
	"github.com/healtronlabs/gofasta/app/rest/controllers"
	"github.com/healtronlabs/gofasta/app/rest/routes"
	apperrors "github.com/healtronlabs/gofasta/pkg/errors"
	"github.com/healtronlabs/gofasta/pkg/middleware"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	Long:  "Start the gofasta HTTP server with GraphQL and REST endpoints.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return startServer()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}

func startServer() error {
	container, err := di.InitializeServiceContainer()
	if err != nil {
		return err
	}
	di.WireSetterDependencies(container)

	cfg := container.Config
	logger := container.Logger

	// Build GraphQL handler
	graphqlHandler := handler.NewDefaultServer(app.NewExecutableSchema(app.Config{
		Resolvers: container.Resolver,
	}))
	graphqlHandler.SetErrorPresenter(apperrors.GraphQLErrorPresenter)

	// Build REST router
	healthController := controllers.NewHealthController(container.DB)
	apiRouter := routes.InitApiRoutes(&routes.RouteConfig{
		UserController:   container.UserController,
		HealthController: healthController,
	})

	mux := http.NewServeMux()
	mux.Handle(cfg.GraphQL.PlaygroundRoute, playground.Handler("GraphQL playground", cfg.GraphQL.GeneralRoute))
	mux.Handle(cfg.GraphQL.GeneralRoute, graphqlHandler)
	mux.Handle("/", apiRouter)

	// Apply middleware chain
	rootHandler := middleware.Chain(
		mux,
		middleware.RequestID(),
		middleware.RequestLogging(logger),
		middleware.Recovery(logger),
		middleware.CORS(cfg.Server.AllowedOrigins),
	)

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: rootHandler,
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

	// Cleanup database connection
	if sqlDB, err := container.DB.DB(); err == nil {
		sqlDB.Close()
	}

	slog.Info("server stopped")
	return nil
}
