package response

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/anupam-chopra/prism/internal/model"
)

type Envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrDetail  `json:"error,omitempty"`
}

type ErrDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

const (
	ErrCodeValidation   = "VALIDATION_ERROR"
	ErrCodeNotFound     = "NOT_FOUND"
	ErrCodeUnauthorized = "UNAUTHORIZED"
	ErrCodeDRMError     = "DRM_ERROR"
	ErrCodeTokenError   = "TOKEN_ERROR"
	ErrCodeInternal     = "INTERNAL_ERROR"
	ErrCodeBadGateway   = "BAD_GATEWAY"
	ErrCodeUnavailable  = "SERVICE_UNAVAILABLE"
)

func Success(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Envelope{
		Success: true,
		Data:    data,
	})
}

func Error(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Envelope{
		Success: false,
		Error: &ErrDetail{
			Code:    code,
			Message: message,
		},
	})
}

func ValidationError(w http.ResponseWriter, err error) {
	var ve *model.ValidationError
	if errors.As(err, &ve) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Envelope{
			Success: false,
			Error: &ErrDetail{
				Code:    ErrCodeValidation,
				Message: ve.Message,
				Field:   ve.Field,
			},
		})
		return
	}
	Error(w, http.StatusBadRequest, ErrCodeValidation, err.Error())
}

func Raw(w http.ResponseWriter, status int, contentType string, data []byte) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	w.Write(data)
}

func InternalError(w http.ResponseWriter, logger *slog.Logger, err error, requestID string) {
	logger.Error("internal error", "request_id", requestID, "error", err)
	Error(w, http.StatusInternalServerError, ErrCodeInternal, "an internal error occurred")
}
