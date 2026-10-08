package items

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandlerCreateItem(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 8, 23, 13, 30, 0, 0, time.UTC)
	repository := &handlerFakeRepository{}
	service := NewService(repository)
	service.now = func() time.Time { return now }
	service.newID = func() (string, error) { return "018fb3b2-0f9d-4f59-8a63-7ef4bb812345", nil }

	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/items", strings.NewReader(`{"name":"Example item"}`))
	recorder := httptest.NewRecorder()

	NewHandler(service, nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	var response itemResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if response.ID != "018fb3b2-0f9d-4f59-8a63-7ef4bb812345" {
		t.Fatalf("expected created id, got %q", response.ID)
	}

	if response.Name != "Example item" {
		t.Fatalf("expected created name, got %q", response.Name)
	}

	if repository.created.Name != "Example item" {
		t.Fatalf("expected item to be persisted, got %+v", repository.created)
	}
}

func TestHandlerCreateRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/items", strings.NewReader(`{"name":"Example item","unknown":true}`))
	recorder := httptest.NewRecorder()

	NewHandler(NewService(&handlerFakeRepository{}), nil).ServeHTTP(recorder, request)

	assertHandlerError(t, recorder, http.StatusBadRequest, "invalid_request")
}

func TestHandlerCreateRejectsInvalidName(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/items", strings.NewReader(`{"name":"   "}`))
	recorder := httptest.NewRecorder()

	NewHandler(NewService(&handlerFakeRepository{}), nil).ServeHTTP(recorder, request)

	assertHandlerError(t, recorder, http.StatusBadRequest, "invalid_request")
}

func TestHandlerGetItem(t *testing.T) {
	t.Parallel()

	expected := Item{
		ID:        "018fb3b2-0f9d-4f59-8a63-7ef4bb812345",
		Name:      "Example item",
		CreatedAt: time.Date(2026, 8, 23, 13, 30, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 8, 23, 13, 30, 0, 0, time.UTC),
	}
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/items/018fb3b2-0f9d-4f59-8a63-7ef4bb812345", nil)
	recorder := httptest.NewRecorder()

	NewHandler(NewService(&handlerFakeRepository{found: expected}), nil).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
	}

	var response itemResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if response.ID != expected.ID {
		t.Fatalf("expected item id %q, got %q", expected.ID, response.ID)
	}
}

func TestHandlerGetReturnsNotFound(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/items/018fb3b2-0f9d-4f59-8a63-7ef4bb812345", nil)
	recorder := httptest.NewRecorder()

	NewHandler(NewService(&handlerFakeRepository{findErr: ErrNotFound}), nil).ServeHTTP(recorder, request)

	assertHandlerError(t, recorder, http.StatusNotFound, "not_found")
}

func TestHandlerGetRejectsInvalidID(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/items/not-a-uuid", nil)
	recorder := httptest.NewRecorder()

	NewHandler(NewService(&handlerFakeRepository{}), nil).ServeHTTP(recorder, request)

	assertHandlerError(t, recorder, http.StatusBadRequest, "invalid_request")
}

func TestHandlerDoesNotLeakInternalErrors(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/items", strings.NewReader(`{"name":"Example item"}`))
	recorder := httptest.NewRecorder()
	var logs bytes.Buffer

	NewHandler(
		NewService(&handlerFakeRepository{createErr: errors.New("password=super-secret-password")}),
		slog.New(slog.NewTextHandler(&logs, nil)),
	).ServeHTTP(recorder, request)

	assertHandlerError(t, recorder, http.StatusInternalServerError, "internal_server_error")

	if strings.Contains(recorder.Body.String(), "super-secret-password") {
		t.Fatalf("expected response not to leak internal error, got %q", recorder.Body.String())
	}

	if strings.Contains(logs.String(), "super-secret-password") {
		t.Fatalf("expected logs not to leak internal error, got %q", logs.String())
	}
}

func TestHandlerRejectsUnsupportedMethods(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		method    string
		path      string
		wantAllow string
	}{
		{name: "items collection", method: http.MethodGet, path: "/items", wantAllow: http.MethodPost},
		{name: "item resource", method: http.MethodPost, path: "/items/018fb3b2-0f9d-4f59-8a63-7ef4bb812345", wantAllow: http.MethodGet},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequestWithContext(context.Background(), tt.method, tt.path, nil)
			recorder := httptest.NewRecorder()

			NewHandler(NewService(&handlerFakeRepository{}), nil).ServeHTTP(recorder, request)

			assertHandlerError(t, recorder, http.StatusMethodNotAllowed, "method_not_allowed")

			if allow := recorder.Header().Get("Allow"); allow != tt.wantAllow {
				t.Fatalf("expected Allow header %q, got %q", tt.wantAllow, allow)
			}
		})
	}
}

func assertHandlerError(t *testing.T, recorder *httptest.ResponseRecorder, wantStatus int, wantCode string) {
	t.Helper()

	if recorder.Code != wantStatus {
		t.Fatalf("expected status %d, got %d", wantStatus, recorder.Code)
	}

	var response handlerErrorResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if response.Error.Code != wantCode {
		t.Fatalf("expected error code %q, got %q", wantCode, response.Error.Code)
	}
}

type handlerFakeRepository struct {
	created   Item
	found     Item
	createErr error
	findErr   error
}

func (r *handlerFakeRepository) Create(_ context.Context, item Item) error {
	r.created = item
	return r.createErr
}

func (r *handlerFakeRepository) FindByID(_ context.Context, _ string) (Item, error) {
	if r.findErr != nil {
		return Item{}, r.findErr
	}

	return r.found, nil
}
