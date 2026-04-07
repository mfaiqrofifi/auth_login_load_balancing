package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/google/uuid"

	"load_balancing_project_auth/internal/model"
)

type Middleware func(http.Handler) http.Handler

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

type timeoutResponseWriter struct {
	header     http.Header
	body       bytes.Buffer
	statusCode int
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (w *timeoutResponseWriter) Header() http.Header {
	return w.header
}

func (w *timeoutResponseWriter) WriteHeader(statusCode int) {
	if w.statusCode != 0 {
		return
	}
	w.statusCode = statusCode
}

func (w *timeoutResponseWriter) Write(data []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}

	return w.body.Write(data)
}

func Chain(handler http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}

	return handler
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = uuid.NewString()
		}

		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDContextKey, requestID)))
	})
}

func RequestLogger(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recorder := &statusRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			start := time.Now()
			next.ServeHTTP(recorder, r)

			logger.Info("request completed",
				slog.String("request_id", GetRequestID(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("remote_addr", r.RemoteAddr),
				slog.Int("status_code", recorder.statusCode),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}

func Recoverer(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("panic recovered",
						slog.String("request_id", GetRequestID(r.Context())),
						slog.Any("panic", recovered),
						slog.String("stacktrace", string(debug.Stack())),
					)
					writeError(w, r, http.StatusInternalServerError, "internal_server_error", "internal server error")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

func Timeout(timeout time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			bufferedWriter := &timeoutResponseWriter{
				header: make(http.Header),
			}

			done := make(chan struct{})

			go func() {
				defer close(done)
				next.ServeHTTP(bufferedWriter, r.WithContext(ctx))
			}()

			select {
			case <-done:
				copyHeader(w.Header(), bufferedWriter.header)
				if bufferedWriter.statusCode == 0 {
					bufferedWriter.statusCode = http.StatusOK
				}
				w.WriteHeader(bufferedWriter.statusCode)
				_, _ = w.Write(bufferedWriter.body.Bytes())
			case <-ctx.Done():
				writeError(w, r, http.StatusGatewayTimeout, "request_timeout", "request timed out")
			}
		})
	}
}

func CORS(allowedOrigins, allowedMethods, allowedHeaders []string, allowCredentials bool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin == "" || len(allowedOrigins) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			if !isOriginAllowed(origin, allowedOrigins) {
				if r.Method == http.MethodOptions {
					writeError(w, r, http.StatusForbidden, "cors_origin_denied", "origin is not allowed")
					return
				}

				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", strings.Join(allowedMethods, ", "))
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
			if allowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-XSS-Protection", "0")

		if strings.HasPrefix(r.URL.Path, "/auth/") {
			w.Header().Set("Cache-Control", "no-store")
		}

		next.ServeHTTP(w, r)
	})
}

func InstanceHeader(instanceName string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-App-Instance", instanceName)
			next.ServeHTTP(w, r)
		})
	}
}

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
		RequestID: GetRequestID(r.Context()),
	})
}

func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowedOrigin := range allowedOrigins {
		if allowedOrigin == "*" || strings.EqualFold(origin, allowedOrigin) {
			return true
		}
	}

	return false
}

func copyHeader(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}
