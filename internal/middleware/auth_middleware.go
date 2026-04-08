package middleware

import (
	"context"
	"net/http"
	"strings"

	"load_balancing_project_auth/internal/model"
	"load_balancing_project_auth/internal/service"
)

func RequireAuth(tokenService *service.TokenService) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorization := strings.TrimSpace(r.Header.Get("Authorization"))
			if authorization == "" {
				writeError(w, r, http.StatusUnauthorized, "missing_authorization_header", "missing authorization header")
				return
			}

			parts := strings.SplitN(authorization, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
				writeError(w, r, http.StatusUnauthorized, "invalid_authorization_header", "invalid authorization header")
				return
			}

			identity, err := tokenService.ValidateAccessToken(strings.TrimSpace(parts[1]))
			if err != nil {
				writeError(w, r, http.StatusUnauthorized, "invalid_access_token", err.Error())
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
