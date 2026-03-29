package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"load_balancing_project_auth/internal/config"
	"load_balancing_project_auth/internal/middleware"
	"load_balancing_project_auth/internal/model"
	"load_balancing_project_auth/internal/service"
)

const refreshTokenCookieName = "refresh_token"

type Handler struct {
	config           config.Config
	healthService    *service.HealthService
	authService      *service.AuthService
	tokenService     *service.TokenService
	rateLimitService *service.RateLimitService
}

func NewHandler(cfg config.Config, healthService *service.HealthService, authService *service.AuthService, tokenService *service.TokenService, rateLimitService *service.RateLimitService) *Handler {
	return &Handler{
		config:           cfg,
		healthService:    healthService,
		authService:      authService,
		tokenService:     tokenService,
		rateLimitService: rateLimitService,
	}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", h.handleRoot)
	mux.HandleFunc("/health", h.handleHealth)
	mux.Handle("/auth/register", middleware.RateLimit(
		h.rateLimitService,
		"register",
		h.config.RegisterRateLimitRequests,
		time.Duration(h.config.RegisterRateLimitWindowSec)*time.Second,
	)(http.HandlerFunc(h.handleRegister)))
	mux.Handle("/auth/login", middleware.RateLimit(
		h.rateLimitService,
		"login",
		h.config.LoginRateLimitRequests,
		time.Duration(h.config.LoginRateLimitWindowSec)*time.Second,
	)(http.HandlerFunc(h.handleLogin)))
	mux.Handle("/auth/refresh", middleware.RateLimit(
		h.rateLimitService,
		"refresh",
		h.config.RefreshRateLimitRequests,
		time.Duration(h.config.RefreshRateLimitWindowSec)*time.Second,
	)(http.HandlerFunc(h.handleRefresh)))
	mux.HandleFunc("/auth/logout", h.handleLogout)
	mux.Handle("/auth/logout-all", middleware.RequireAuth(h.tokenService)(http.HandlerFunc(h.handleLogoutAll)))
	mux.Handle("/auth/sessions", middleware.RequireAuth(h.tokenService)(http.HandlerFunc(h.handleSessions)))
	mux.Handle("/auth/sessions/", middleware.RequireAuth(h.tokenService)(http.HandlerFunc(h.handleSessionByID)))
	mux.Handle("/auth/me", middleware.RequireAuth(h.tokenService)(http.HandlerFunc(h.handleMe)))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/auth/sessions/") && r.URL.Path != "/" && r.URL.Path != "/health" && r.URL.Path != "/auth/login" && r.URL.Path != "/auth/register" && r.URL.Path != "/auth/refresh" && r.URL.Path != "/auth/logout" && r.URL.Path != "/auth/logout-all" && r.URL.Path != "/auth/sessions" && r.URL.Path != "/auth/me" {
			writeJSON(w, http.StatusNotFound, model.ErrorResponse{
				Error: "route not found",
			})
			return
		}

		mux.ServeHTTP(w, r)
	})

	return middleware.Chain(
		handler,
		middleware.InstanceHeader(h.config.InstanceName),
		middleware.Recoverer,
		middleware.RequestLogger,
	)
}

func (h *Handler) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorResponse{
			Error: "method not allowed",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"message":       "authentication service base is running",
		"app_name":      h.config.AppName,
		"instance_name": h.config.InstanceName,
	})
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorResponse{
			Error: "method not allowed",
		})
		return
	}

	status := h.healthService.Check()
	writeJSON(w, http.StatusOK, map[string]any{
		"status":        status.Status,
		"timestamp":     status.Timestamp,
		"instance_name": h.config.InstanceName,
	})
}

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorResponse{
			Error: "method not allowed",
		})
		return
	}

	identity, ok := middleware.GetAuthIdentity(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, model.ErrorResponse{
			Error: "unauthorized",
		})
		return
	}

	writeJSON(w, http.StatusOK, model.MeResponse{
		UserID: identity.UserID,
		Email:  identity.Email,
	})
}

func (h *Handler) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorResponse{
			Error: "method not allowed",
		})
		return
	}

	identity, ok := middleware.GetAuthIdentity(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, model.ErrorResponse{
			Error: "unauthorized",
		})
		return
	}

	response, err := h.authService.ListSessions(r.Context(), identity.UserID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) handleSessionByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorResponse{
			Error: "method not allowed",
		})
		return
	}

	identity, ok := middleware.GetAuthIdentity(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, model.ErrorResponse{
			Error: "unauthorized",
		})
		return
	}

	sessionID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/auth/sessions/"))
	if sessionID == "" {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error: "session id is required",
		})
		return
	}

	response, err := h.authService.RevokeSession(r.Context(), identity.UserID, sessionID)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, service.ErrInvalidRefreshToken) {
			statusCode = http.StatusUnauthorized
		}

		writeJSON(w, statusCode, model.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorResponse{
			Error: "method not allowed",
		})
		return
	}

	defer r.Body.Close()

	var request model.RegisterRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		if errors.Is(err, io.EOF) {
			writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
				Error: "request body is required",
			})
			return
		}

		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error: "invalid JSON request body",
		})
		return
	}

	request.Email = strings.TrimSpace(request.Email)
	request.Password = strings.TrimSpace(request.Password)

	if request.Email == "" || request.Password == "" {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error: "email and password are required",
		})
		return
	}

	response, err := h.authService.Register(r.Context(), request.Email, request.Password, h.requestMetadata(r))
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, service.ErrInvalidLoginInput) {
			statusCode = http.StatusBadRequest
		}
		if errors.Is(err, service.ErrEmailAlreadyExists) {
			statusCode = http.StatusConflict
		}

		writeJSON(w, statusCode, model.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusCreated, response)
}

func (h *Handler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorResponse{
			Error: "method not allowed",
		})
		return
	}

	refreshToken, err := h.readRefreshTokenCookie(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, model.ErrorResponse{
			Error: "missing or invalid refresh token cookie",
		})
		return
	}

	response, err := h.authService.RefreshAccessToken(r.Context(), refreshToken, h.requestMetadata(r))
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, service.ErrInvalidRefreshToken) {
			statusCode = http.StatusUnauthorized
		}
		if errors.Is(err, service.ErrRefreshTokenReuseDetected) {
			statusCode = http.StatusUnauthorized
		}

		writeJSON(w, statusCode, model.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	h.setRefreshTokenCookie(w, response.RefreshToken)
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorResponse{
			Error: "method not allowed",
		})
		return
	}

	refreshToken, _ := h.readRefreshTokenCookie(r)

	response, err := h.authService.Logout(r.Context(), refreshToken, h.requestMetadata(r))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	h.clearRefreshTokenCookie(w)
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) handleLogoutAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorResponse{
			Error: "method not allowed",
		})
		return
	}

	identity, ok := middleware.GetAuthIdentity(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, model.ErrorResponse{
			Error: "unauthorized",
		})
		return
	}

	response, err := h.authService.LogoutAll(r.Context(), identity.UserID, h.requestMetadata(r))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, model.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	h.clearRefreshTokenCookie(w)
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, model.ErrorResponse{
			Error: "method not allowed",
		})
		return
	}

	defer r.Body.Close()

	var request model.LoginRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&request); err != nil {
		if errors.Is(err, io.EOF) {
			writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
				Error: "request body is required",
			})
			return
		}

		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error: "invalid JSON request body",
		})
		return
	}

	request.Email = strings.TrimSpace(request.Email)
	request.Password = strings.TrimSpace(request.Password)

	if request.Email == "" || request.Password == "" {
		writeJSON(w, http.StatusBadRequest, model.ErrorResponse{
			Error: "email and password are required",
		})
		return
	}

	response, err := h.authService.Login(r.Context(), request.Email, request.Password, h.requestMetadata(r))
	if err != nil {
		statusCode := http.StatusInternalServerError
		if errors.Is(err, service.ErrInvalidLoginInput) {
			statusCode = http.StatusBadRequest
		}
		if errors.Is(err, service.ErrInvalidCredentials) {
			statusCode = http.StatusUnauthorized
		}

		writeJSON(w, statusCode, model.ErrorResponse{
			Error: err.Error(),
		})
		return
	}

	h.setRefreshTokenCookie(w, response.RefreshToken)
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) setRefreshTokenCookie(w http.ResponseWriter, refreshToken string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    refreshToken,
		Path:     "/",
		Domain:   h.config.CookieDomain,
		HttpOnly: true,
		Secure:   h.config.CookieSecure,
		SameSite: parseSameSite(h.config.CookieSameSite),
		MaxAge:   h.config.RefreshTokenTTLHours * 60 * 60,
	})
}

func (h *Handler) clearRefreshTokenCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    "",
		Path:     "/",
		Domain:   h.config.CookieDomain,
		HttpOnly: true,
		Secure:   h.config.CookieSecure,
		SameSite: parseSameSite(h.config.CookieSameSite),
		MaxAge:   -1,
	})
}

func (h *Handler) readRefreshTokenCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie(refreshTokenCookieName)
	if err != nil {
		return "", err
	}

	value := strings.TrimSpace(cookie.Value)
	if value == "" {
		return "", http.ErrNoCookie
	}

	return value, nil
}

func parseSameSite(value string) http.SameSite {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}

func (h *Handler) requestMetadata(r *http.Request) model.RequestMetadata {
	userAgent := strings.TrimSpace(r.UserAgent())
	ipAddress := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if ipAddress == "" {
		ipAddress = strings.TrimSpace(r.RemoteAddr)
	}

	return model.RequestMetadata{
		UserAgent:  userAgent,
		DeviceName: userAgent,
		IPAddress:  ipAddress,
	}
}
