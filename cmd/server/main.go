package main

import (
	"context"
	"log"
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

	db, err := database.NewPostgresConnection(cfg)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	redisClient, err := database.NewRedisClient(cfg)
	if err != nil {
		log.Fatalf("redis connection failed: %v", err)
	}
	defer redisClient.Close()

	systemRepository := repository.NewSystemRepository()
	userRepository := repository.NewPostgresUserRepository(db)
	sessionRepository := repository.NewPostgresSessionRepository(db)
	auditLogRepository := repository.NewPostgresAuditLogRepository(db)
	refreshTokenRepository := repository.NewRedisRefreshTokenRepository(redisClient)
	healthService := service.NewHealthService(systemRepository)
	tokenService := service.NewTokenService(cfg.JWTAccessSecret, cfg.JWTAccessTTLMinutes, cfg.RefreshTokenTTLHours)
	rateLimitService := service.NewRateLimitService(redisClient)
	auditService := service.NewAuditService(auditLogRepository)
	authService := service.NewAuthService(userRepository, sessionRepository, refreshTokenRepository, tokenService, auditService)
	httpHandler := handler.NewHandler(cfg, healthService, authService, tokenService, rateLimitService)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpHandler.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("%s is running on port %s", cfg.AppName, cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("graceful shutdown failed: %v", err)
	}

	log.Printf("%s stopped cleanly", cfg.AppName)
}
