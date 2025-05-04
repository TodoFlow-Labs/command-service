package main

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"
	"github.com/rs/zerolog/log"

	"github.com/todoflow-labs/shared-dtos/dto"
	"github.com/todoflow-labs/command-service/internal/config"
	"github.com/todoflow-labs/command-service/internal/logging"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	// Initialize logger
	logger := logging.New(cfg.LogLevel)

	// Connect to NATS and JetStream
	nc, err := nats.Connect(cfg.NATSURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to NATS")
	}
	js, err := nc.JetStream()
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to init JetStream")
	}

	// Ensure JetStream stream exists
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "todo_commands",
		Subjects: []string{"todo.commands"},
	})
	if err != nil && !strings.Contains(err.Error(), "file already in use") {
		logger.Fatal().Err(err).Msg("failed to create JetStream stream")
	}

	// Setup HTTP router
	r := chi.NewRouter()
	r.Post("/todos", publishCreate(js, logger))
	r.Put("/todos/{id}", publishUpdate(js, logger))
	r.Delete("/todos/{id}", publishDelete(js, logger))

	// Start HTTP server
	logger.Info().Msgf("command-service listening on %s", cfg.HTTPAddr)
	http.ListenAndServe(cfg.HTTPAddr, r)
}

// publishCreate handles CreateTodoCommand
func publishCreate(js nats.JetStreamContext, log *logging.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var cmd dto.CreateTodoCommand
		if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		cmd.Type = dto.CreateTodoCmd
		data, _ := json.Marshal(cmd)
		if _, err := js.Publish("todo.commands", data); err != nil {
			log.Error().Err(err).Msg("failed to publish create command")
			http.Error(w, "publish failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

// publishUpdate handles UpdateTodoCommand
func publishUpdate(js nats.JetStreamContext, log *logging.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		var cmd dto.UpdateTodoCommand
		if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		cmd.Type = dto.UpdateTodoCmd
		cmd.ID = id
		data, _ := json.Marshal(cmd)
		if _, err := js.Publish("todo.commands", data); err != nil {
			log.Error().Err(err).Msg("failed to publish update command")
			http.Error(w, "publish failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}

// publishDelete handles DeleteTodoCommand
func publishDelete(js nats.JetStreamContext, log *logging.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		cmd := dto.DeleteTodoCommand{BaseCommand: dto.BaseCommand{Type: dto.DeleteTodoCmd, ID: id}}
		data, _ := json.Marshal(cmd)
		if _, err := js.Publish("todo.commands", data); err != nil {
			log.Error().Err(err).Msg("failed to publish delete command")
			http.Error(w, "publish failed", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}
}
