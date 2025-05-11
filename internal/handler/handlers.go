package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"

	"github.com/todoflow-labs/shared-dtos/dto"
	"github.com/todoflow-labs/shared-dtos/logging"
	"github.com/todoflow-labs/shared-dtos/metrics"
)

func getUserID(r *http.Request) string {
	// Replace with actual auth logic (JWT/session/header)
	return r.Header.Get("X-User-ID")
}

func PublishCreate(js nats.JetStreamContext, logger *logging.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debug().Msg("handling create todo command")

		var cmd dto.CreateTodoCommand
		if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
			logger.Error().Err(err).Msg("invalid create payload")
			http.Error(w, "invalid payload", http.StatusBadRequest)
			metrics.TodoCreateCounter.WithLabelValues("invalid").Inc()
			return
		}

		cmd.Type = dto.CreateTodoCmd
		cmd.UserID = getUserID(r)

		data, _ := json.Marshal(cmd)
		if _, err := js.Publish("todo.commands", data); err != nil {
			logger.Error().Err(err).Msg("failed to publish create command")
			http.Error(w, "publish failed", http.StatusInternalServerError)
			metrics.TodoCreateCounter.WithLabelValues("error").Inc()
			return
		}

		logger.Debug().Msg("create command published successfully")
		metrics.TodoCreateCounter.WithLabelValues("success").Inc()
		w.WriteHeader(http.StatusAccepted)
	}
}

func PublishUpdate(js nats.JetStreamContext, logger *logging.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debug().Msg("handling update todo command")
		id := chi.URLParam(r, "id")

		var cmd dto.UpdateTodoCommand
		if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
			logger.Error().Err(err).Msg("invalid update payload")
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}

		cmd.Type = dto.UpdateTodoCmd
		cmd.ID = id
		cmd.UserID = getUserID(r)

		data, _ := json.Marshal(cmd)
		if _, err := js.Publish("todo.commands", data); err != nil {
			logger.Error().Err(err).Msg("failed to publish update command")
			http.Error(w, "publish failed", http.StatusInternalServerError)
			return
		}

		logger.Debug().Msg("update command published successfully")
		w.WriteHeader(http.StatusAccepted)
	}
}

func PublishDelete(js nats.JetStreamContext, logger *logging.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Debug().Msg("handling delete todo command")
		id := chi.URLParam(r, "id")

		cmd := dto.DeleteTodoCommand{
			BaseCommand: dto.BaseCommand{
				Type:   dto.DeleteTodoCmd,
				ID:     id,
				UserID: getUserID(r),
			},
		}

		data, _ := json.Marshal(cmd)
		if _, err := js.Publish("todo.commands", data); err != nil {
			logger.Error().Err(err).Msg("failed to publish delete command")
			http.Error(w, "publish failed", http.StatusInternalServerError)
			return
		}

		logger.Debug().Msg("delete command published successfully")
		w.WriteHeader(http.StatusAccepted)
	}
}
