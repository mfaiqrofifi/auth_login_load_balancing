package middleware

import (
	"context"
	"net/http"
	"strings"

	"load_balancing_project_auth/internal/model"
	"load_balancing_project_auth/internal/service"
)

type contextKey string

const authIdentityContextKey contextKey = "auth_identity"

func RequireAuth(tokenService *service.TokenService) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorization := strings.TrimSpace(r.Header.Get("Authorization"))
			if authorization == "" {
				writeJSON(w, http.StatusUnauthorized, model.ErrorResponse{
					Error: "missing authorization header",
				})
				return
			}

			parts := strings.SplitN(authorization, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
				writeJSON(w, http.StatusUnauthorized, model.ErrorResponse{
					Error: "invalid authorization header",
				})
				return
			}

			identity, err := tokenService.ValidateAccessToken(strings.TrimSpace(parts[1]))
			if err != nil {
				writeJSON(w, http.StatusUnauthorized, model.ErrorResponse{
					Error: err.Error(),
				})
				return
			}

			ctx := context.WithValue(r.Context(), authIdentityContextKey, identity)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetAuthIdentity(ctx context.Context) (model.AuthIdentity, bool) {
	identity, ok := ctx.Value(authIdentityContextKey).(model.AuthIdentity)
	return identity, ok
}
