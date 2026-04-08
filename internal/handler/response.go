package handler

import (
	"bytes"
	"encoding/json"
	"net/http"

	"load_balancing_project_auth/internal/middleware"
	"load_balancing_project_auth/internal/model"
)

func writeJSON(w http.ResponseWriter, statusCode int, payload any) {
	var buffer bytes.Buffer

	if err := json.NewEncoder(&buffer).Encode(payload); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"code":"internal_server_error","message":"failed to encode response"}}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write(buffer.Bytes())
}

func writeError(w http.ResponseWriter, r *http.Request, statusCode int, code, message string) {
	writeJSON(w, statusCode, model.ErrorResponse{
		Error: model.APIError{
			Code:    code,
			Message: message,
		},
		RequestID: middleware.GetRequestID(r.Context()),
	})
}
