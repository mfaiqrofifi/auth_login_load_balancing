package middleware

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"load_balancing_project_auth/internal/service"
)

func RateLimit(rateLimitService *service.RateLimitService, scope string, limit int, window time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identifier := rateLimitIdentifier(r)
			result, err := rateLimitService.Allow(r.Context(), scope, identifier, limit, window)
			if err != nil {
				writeError(w, r, http.StatusInternalServerError, "rate_limit_check_failed", "rate limit check failed")
				return
			}

			if !result.Allowed {
				w.Header().Set("Retry-After", formatRetryAfter(result.ResetIn))
				writeError(w, r, http.StatusTooManyRequests, "rate_limit_exceeded", "too many requests, please try again later")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func rateLimitIdentifier(r *http.Request) string {
	forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			return strings.TrimSpace(parts[0])
		}
	}

	remoteAddr := strings.TrimSpace(r.RemoteAddr)
	if remoteAddr == "" {
		return "unknown"
	}

	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil && host != "" {
		return host
	}

	return remoteAddr
}

func formatRetryAfter(duration time.Duration) string {
	seconds := int(duration.Seconds())
	if seconds < 1 {
		seconds = 1
	}

	return strconv.Itoa(seconds)
}
