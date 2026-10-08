package items

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Handler struct {
	service *Service
	log     *slog.Logger
}

type createItemRequest struct {
	Name string `json:"name"`
}

type itemResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type handlerErrorResponse struct {
	Error handlerErrorDetails `json:"error"`
}

type handlerErrorDetails struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewHandler(service *Service, log *slog.Logger) http.Handler {
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	return &Handler{
		service: service,
		log:     log,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/items":
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			writeHandlerError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		h.create(w, r)
	case strings.HasPrefix(r.URL.Path, "/items/"):
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			writeHandlerError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}

		h.get(w, r)
	default:
		writeHandlerError(w, http.StatusNotFound, "not_found", "route not found")
	}
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var request createItemRequest
	if err := decodeJSONBody(r, &request); err != nil {
		writeHandlerError(w, http.StatusBadRequest, "invalid_request", "request body must be a valid JSON object")
		return
	}

	item, err := h.service.Create(r.Context(), request.Name)
	if err != nil {
		h.handleServiceError(w, r, err, "create_item")
		return
	}

	writeHandlerJSON(w, http.StatusCreated, toItemResponse(item))
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/items/")
	if id == "" || strings.Contains(id, "/") {
		writeHandlerError(w, http.StatusNotFound, "not_found", "route not found")
		return
	}

	item, err := h.service.Get(r.Context(), id)
	if err != nil {
		h.handleServiceError(w, r, err, "get_item")
		return
	}

	writeHandlerJSON(w, http.StatusOK, toItemResponse(item))
}

func (h *Handler) handleServiceError(w http.ResponseWriter, r *http.Request, err error, operation string) {
	var validationErr ValidationError
	if errors.As(err, &validationErr) {
		writeHandlerError(w, http.StatusBadRequest, "invalid_request", validationErr.Message)
		return
	}

	if errors.Is(err, ErrNotFound) {
		writeHandlerError(w, http.StatusNotFound, "not_found", "item not found")
		return
	}

	h.log.ErrorContext(r.Context(), "items request failed", "operation", operation)
	writeHandlerError(w, http.StatusInternalServerError, "internal_server_error", "internal server error")
}

func decodeJSONBody(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(target); err != nil {
		return err
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return err
	}

	return nil
}

func toItemResponse(item Item) itemResponse {
	return itemResponse(item)
}

func writeHandlerError(w http.ResponseWriter, statusCode int, code, message string) {
	writeHandlerJSON(w, statusCode, handlerErrorResponse{
		Error: handlerErrorDetails{
			Code:    code,
			Message: message,
		},
	})
}

func writeHandlerJSON(w http.ResponseWriter, statusCode int, response any) {
	body, err := json.Marshal(response)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"code":"internal_server_error","message":"internal server error"}}` + "\n"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write(append(body, '\n'))
}
