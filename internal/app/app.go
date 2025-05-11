package app

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/todoflow-labs/command-service/internal/config"
	"github.com/todoflow-labs/command-service/internal/handler"
	"github.com/todoflow-labs/shared-dtos/logging"
	"github.com/todoflow-labs/shared-dtos/metrics"
)

func Run() {
	// Load configuration

	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}
	// Initialize logger and metrics
	logger := logging.New(cfg.LogLevel)
	metrics.Init(cfg.MetricsAddr)
	logger.Info().Msgf("metrics server listening on %s", cfg.MetricsAddr)

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

	r := chi.NewRouter()
	r.Post("/todos", handler.PublishCreate(js, logger))
	r.Put("/todos/{id}", handler.PublishUpdate(js, logger))
	r.Delete("/todos/{id}", handler.PublishDelete(js, logger))

	logger.Info().Msgf("command-service listening on %s", cfg.HTTPAddr)
	http.ListenAndServe(cfg.HTTPAddr, r)
}
