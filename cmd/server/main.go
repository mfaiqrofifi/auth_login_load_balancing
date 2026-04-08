package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"load_balancing_project_auth/internal/config"
	"load_balancing_project_auth/internal/database"
	"load_balancing_project_auth/internal/handler"
	"load_balancing_project_auth/internal/repository"
	"load_balancing_project_auth/internal/service"
)

func main() {
	cfg := config.Load()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: cfg.LogLevel,
	}))

	db, err := database.NewPostgresConnection(cfg)
	if err != nil {
		logger.Error("database connection failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer db.Close()

	redisClient, err := database.NewRedisClient(cfg)
	if err != nil {
		logger.Error("redis connection failed", slog.Any("error", err))
		os.Exit(1)
	}
	defer redisClient.Close()

	systemRepository := repository.NewSystemRepository()
	userRepository := repository.NewPostgresUserRepository(db)
	sessionRepository := repository.NewPostgresSessionRepository(db)
	auditLogRepository := repository.NewPostgresAuditLogRepository(db)
	refreshTokenRepository := repository.NewRedisRefreshTokenRepository(redisClient)
	healthService := service.NewHealthService(systemRepository)
	tokenService := service.NewTokenService(cfg.JWTAccessSecret, cfg.JWTAccessTTLMinutes, cfg.RefreshTokenTTLHours, cfg.JWTIssuer, cfg.JWTAudience)
	rateLimitService := service.NewRateLimitService(redisClient)
	auditService := service.NewAuditService(auditLogRepository)
	authService := service.NewAuthService(userRepository, sessionRepository, refreshTokenRepository, tokenService, auditService)
	httpHandler := handler.NewHandler(cfg, logger, healthService, authService, tokenService, rateLimitService)

	server := &http.Server{
		Addr:              "0.0.0.0:" + cfg.Port,
		Handler:           httpHandler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("server starting",
			slog.String("app_name", cfg.AppName),
			slog.String("app_env", cfg.AppEnv),
			slog.String("port", cfg.Port),
			slog.String("instance_name", cfg.InstanceName),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownTimeoutSec)*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("server stopped cleanly", slog.String("app_name", cfg.AppName))
}
