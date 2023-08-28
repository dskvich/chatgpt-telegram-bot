package response

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/sushkevichd/chatgpt-telegram-bot/pkg/logger"
)

type JSONResponseWriter struct{}

func (j *JSONResponseWriter) WriteSuccessResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("encoding success response", logger.Err(err))
	}
}

func (j *JSONResponseWriter) WriteErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(ErrorResponse{Error: message}); err != nil {
		slog.Error("encoding error response", logger.Err(err))
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}
