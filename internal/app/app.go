package app

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/todoflow-labs/command-service/internal/config"
	"github.com/todoflow-labs/command-service/internal/handler"
	"github.com/todoflow-labs/shared-dtos/logging"
	"github.com/todoflow-labs/shared-dtos/metrics"
)

const USER_ID = "user_id"

func Run() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	// Initialize logger and metrics
	logger := logging.New(cfg.LogLevel).With().Str("service", "command-service").Logger()
	metrics.Init(cfg.MetricsAddr)
	logger.Info().Msgf("metrics server listening on %s", cfg.MetricsAddr)

	// Connect to NATS
	nc, err := nats.Connect(cfg.NATSURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to NATS")
	}
	js, err := nc.JetStream()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init JetStream")
	}
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "todo_commands",
		Subjects: []string{"todo.commands"},
	})
	if err != nil && !strings.Contains(err.Error(), "file already in use") {
		logger.Fatal().Err(err).Msg("failed to create JetStream stream")
	}

	// Set up HTTP server and routes
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(jsonContentType)
	r.Use(authMiddleware)

	// Routes
	r.Post("/todos", handler.PublishCreate(js, logger))
	r.Put("/todos/{id}", handler.PublishUpdate(js, logger))
	r.Delete("/todos/{id}", handler.PublishDelete(js, logger))

	// Error handlers
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusNotFound, "route not found")
		logger.Warn().Str("path", r.URL.Path).Msg("404 not found")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		logger.Warn().Str("path", r.URL.Path).Msg("405 method not allowed")
	})

	logger.Info().Msgf("command-service listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, r); err != nil {
		logger.Fatal().Err(err).Msg("HTTP server failed")
	}
}

// Forces JSON Content-Type for all responses
func jsonContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}

// Writes a structured JSON error
func writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": msg,
	})
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			if os.Getenv("ENV") != "production" {
				userID = "test-user"
			} else {
				writeError(w, http.StatusUnauthorized, "Missing X-User-ID header")
				return
			}
		}
		ctx := context.WithValue(r.Context(), USER_ID, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
