package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"

	"github.com/todoflow-labs/command-service/internal/handler"
	"github.com/todoflow-labs/shared-dtos/dto"
	"github.com/todoflow-labs/shared-dtos/logging"
)

func setupEmbeddedNATSServer(t *testing.T) (*server.Server, nats.JetStreamContext, *nats.Conn) {
	opts := &server.Options{
		JetStream: true,
		StoreDir:  t.TempDir(),
		Port:      -1,
		NoLog:     true,
		NoSigs:    true,
	}
	srv, err := server.NewServer(opts)
	assert.NoError(t, err)

	go srv.Start()
	if !srv.ReadyForConnections(10 * time.Second) {
		t.Fatal("NATS server not ready in time")
	}

	nc, err := nats.Connect(srv.ClientURL())
	assert.NoError(t, err)

	js, err := nc.JetStream()
	assert.NoError(t, err)

	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "todo_commands",
		Subjects: []string{"todo.commands"},
	})
	assert.NoError(t, err)

	return srv, js, nc
}

func withRouteParam(r *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	return r.WithContext(ctx)
}

func TestPublishCreate(t *testing.T) {
	srv, js, nc := setupEmbeddedNATSServer(t)
	defer srv.Shutdown()
	defer nc.Close()
	logger := logging.New("debug")

	handlerFunc := handler.PublishCreate(js, logger)
	cmd := dto.CreateTodoCommand{
		BaseCommand: dto.BaseCommand{Type: dto.CreateTodoCmd},
		Title:       "Test todo",
	}
	body, _ := json.Marshal(cmd)
	req := httptest.NewRequest(http.MethodPost, "/todos", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "test-user-1") // 👈 Add test user ID
	resp := httptest.NewRecorder()

	handlerFunc(resp, req)
	assert.Equal(t, http.StatusAccepted, resp.Code)

	sub, err := js.PullSubscribe("todo.commands", "test-durable")
	assert.NoError(t, err)
	msgs, err := sub.Fetch(1, nats.MaxWait(time.Second))
	assert.NoError(t, err)
	assert.Len(t, msgs, 1)

	var received dto.CreateTodoCommand
	err = json.Unmarshal(msgs[0].Data, &received)
	assert.NoError(t, err)
	assert.Equal(t, cmd.Title, received.Title)
	assert.Equal(t, dto.CreateTodoCmd, received.Type)
	assert.Equal(t, "test-user-1", received.UserID)
}

func TestPublishUpdate(t *testing.T) {
	srv, js, nc := setupEmbeddedNATSServer(t)
	defer srv.Shutdown()
	defer nc.Close()
	logger := logging.New("debug")

	handlerFunc := handler.PublishUpdate(js, logger)
	cmd := dto.UpdateTodoCommand{
		Title:     ptrString("Updated Title"),
		Completed: ptrBool(true),
	}
	body, _ := json.Marshal(cmd)
	req := httptest.NewRequest(http.MethodPut, "/todos/123", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-ID", "test-user-2") // 👈 Add test user ID
	req = withRouteParam(req, "id", "123")
	resp := httptest.NewRecorder()

	handlerFunc(resp, req)
	assert.Equal(t, http.StatusAccepted, resp.Code)

	sub, err := js.PullSubscribe("todo.commands", "test-durable-update")
	assert.NoError(t, err)
	msgs, err := sub.Fetch(1, nats.MaxWait(time.Second))
	assert.NoError(t, err)
	assert.Len(t, msgs, 1)

	var received dto.UpdateTodoCommand
	err = json.Unmarshal(msgs[0].Data, &received)
	assert.NoError(t, err)
	assert.Equal(t, "123", received.ID)
	assert.Equal(t, *cmd.Title, *received.Title)
	assert.Equal(t, *cmd.Completed, *received.Completed)
	assert.Equal(t, dto.UpdateTodoCmd, received.Type)
	assert.Equal(t, "test-user-2", received.UserID)
}

func TestPublishDelete(t *testing.T) {
	srv, js, nc := setupEmbeddedNATSServer(t)
	defer srv.Shutdown()
	defer nc.Close()
	logger := logging.New("debug")

	handlerFunc := handler.PublishDelete(js, logger)
	req := httptest.NewRequest(http.MethodDelete, "/todos/123", nil)
	req.Header.Set("X-User-ID", "test-user-3") // 👈 Add test user ID
	req = withRouteParam(req, "id", "123")
	resp := httptest.NewRecorder()

	handlerFunc(resp, req)
	assert.Equal(t, http.StatusAccepted, resp.Code)

	sub, err := js.PullSubscribe("todo.commands", "test-durable-delete")
	assert.NoError(t, err)
	msgs, err := sub.Fetch(1, nats.MaxWait(time.Second))
	assert.NoError(t, err)
	assert.Len(t, msgs, 1)

	var received dto.DeleteTodoCommand
	err = json.Unmarshal(msgs[0].Data, &received)
	assert.NoError(t, err)
	assert.Equal(t, "123", received.ID)
	assert.Equal(t, dto.DeleteTodoCmd, received.Type)
	assert.Equal(t, "test-user-3", received.UserID)
}

func ptrString(s string) *string {
	return &s
}

func ptrBool(b bool) *bool {
	return &b
}
