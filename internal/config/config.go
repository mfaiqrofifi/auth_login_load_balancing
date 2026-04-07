package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const (
	defaultAppEnv                     = "development"
	defaultPort                       = "8080"
	defaultAppName                    = "auth-service"
	defaultDBHost                     = "localhost"
	defaultDBPort                     = "5432"
	defaultDBUser                     = "postgres"
	defaultDBPassword                 = "postgres"
	defaultDBName                     = "auth_service"
	defaultDBSSLMode                  = "disable"
	defaultJWTSecret                  = "change-this-in-production"
	defaultJWTTTL                     = 15
	defaultRefreshTTL                 = 168
	defaultJWTIssuer                  = "auth-service"
	defaultJWTAudience                = "auth-clients"
	defaultRedisHost                  = "localhost"
	defaultRedisPort                  = "6379"
	defaultRedisDB                    = 0
	defaultCookieSecure               = false
	defaultCookieSameSite             = "Lax"
	defaultRequestTimeoutSec          = 15
	defaultShutdownTimeoutSec         = 10
	defaultTrustProxyHeaders          = true
	defaultLogLevel                   = "INFO"
	defaultLoginRateLimitRequests     = 5
	defaultLoginRateLimitWindowSec    = 60
	defaultRegisterRateLimitRequests  = 3
	defaultRegisterRateLimitWindowSec = 60
	defaultRefreshRateLimitRequests   = 10
	defaultRefreshRateLimitWindowSec  = 60
	defaultInstanceName               = "app-instance"
)

type Config struct {
	AppEnv                     string
	Port                       string
	AppName                    string
	DBHost                     string
	DBPort                     string
	DBUser                     string
	DBPassword                 string
	DBName                     string
	DBSSLMode                  string
	DatabaseURL                string
	JWTAccessSecret            string
	JWTAccessTTLMinutes        int
	RefreshTokenTTLHours       int
	JWTIssuer                  string
	JWTAudience                string
	RedisHost                  string
	RedisPort                  string
	RedisPassword              string
	RedisDB                    int
	RedisURL                   string
	CookieDomain               string
	CookieSecure               bool
	CookieSameSite             string
	RequestTimeoutSec          int
	ShutdownTimeoutSec         int
	TrustProxyHeaders          bool
	LogLevel                   slog.Level
	CORSAllowedOrigins         []string
	CORSAllowedMethods         []string
	CORSAllowedHeaders         []string
	CORSAllowCredentials       bool
	LoginRateLimitRequests     int
	LoginRateLimitWindowSec    int
	RegisterRateLimitRequests  int
	RegisterRateLimitWindowSec int
	RefreshRateLimitRequests   int
	RefreshRateLimitWindowSec  int
	InstanceName               string
}

func Load() Config {
	loadEnvFile()

	cfg := Config{
		AppEnv:                     normalizeAppEnv(getEnv("APP_ENV", defaultAppEnv)),
		Port:                       getEnv("PORT", defaultPort),
		AppName:                    getEnv("APP_NAME", defaultAppName),
		DBHost:                     getEnv("DB_HOST", defaultDBHost),
		DBPort:                     getEnv("DB_PORT", defaultDBPort),
		DBUser:                     getEnv("DB_USER", defaultDBUser),
		DBPassword:                 getEnv("DB_PASSWORD", defaultDBPassword),
		DBName:                     getEnv("DB_NAME", defaultDBName),
		DBSSLMode:                  getEnv("DB_SSLMODE", defaultDBSSLMode),
		DatabaseURL:                getEnv("DATABASE_URL", ""),
		JWTAccessSecret:            getEnv("JWT_ACCESS_SECRET", defaultJWTSecret),
		JWTAccessTTLMinutes:        getEnvAsInt("JWT_ACCESS_TTL_MINUTES", defaultJWTTTL),
		RefreshTokenTTLHours:       getEnvAsInt("REFRESH_TOKEN_TTL_HOURS", defaultRefreshTTL),
		JWTIssuer:                  getEnv("JWT_ISSUER", defaultJWTIssuer),
		JWTAudience:                getEnv("JWT_AUDIENCE", defaultJWTAudience),
		RedisHost:                  getEnv("REDIS_HOST", defaultRedisHost),
		RedisPort:                  getEnv("REDIS_PORT", defaultRedisPort),
		RedisPassword:              getEnv("REDIS_PASSWORD", ""),
		RedisDB:                    getEnvAsInt("REDIS_DB", defaultRedisDB),
		RedisURL:                   getEnv("REDIS_URL", ""),
		CookieDomain:               getEnv("COOKIE_DOMAIN", ""),
		CookieSecure:               getEnvAsBool("COOKIE_SECURE", defaultCookieSecure),
		CookieSameSite:             getEnv("COOKIE_SAMESITE", defaultCookieSameSite),
		RequestTimeoutSec:          getEnvAsInt("REQUEST_TIMEOUT_SECONDS", defaultRequestTimeoutSec),
		ShutdownTimeoutSec:         getEnvAsInt("SHUTDOWN_TIMEOUT_SECONDS", defaultShutdownTimeoutSec),
		TrustProxyHeaders:          getEnvAsBool("TRUST_PROXY_HEADERS", defaultTrustProxyHeaders),
		LogLevel:                   getEnvAsLogLevel("LOG_LEVEL", defaultLogLevel),
		CORSAllowedOrigins:         getEnvAsCSV("CORS_ALLOWED_ORIGINS"),
		CORSAllowedMethods:         getEnvAsCSVWithDefault("CORS_ALLOWED_METHODS", []string{"GET", "POST", "DELETE", "OPTIONS"}),
		CORSAllowedHeaders:         getEnvAsCSVWithDefault("CORS_ALLOWED_HEADERS", []string{"Authorization", "Content-Type"}),
		CORSAllowCredentials:       getEnvAsBool("CORS_ALLOW_CREDENTIALS", false),
		LoginRateLimitRequests:     getEnvAsInt("LOGIN_RATE_LIMIT_REQUESTS", defaultLoginRateLimitRequests),
		LoginRateLimitWindowSec:    getEnvAsInt("LOGIN_RATE_LIMIT_WINDOW_SECONDS", defaultLoginRateLimitWindowSec),
		RegisterRateLimitRequests:  getEnvAsInt("REGISTER_RATE_LIMIT_REQUESTS", defaultRegisterRateLimitRequests),
		RegisterRateLimitWindowSec: getEnvAsInt("REGISTER_RATE_LIMIT_WINDOW_SECONDS", defaultRegisterRateLimitWindowSec),
		RefreshRateLimitRequests:   getEnvAsInt("REFRESH_RATE_LIMIT_REQUESTS", defaultRefreshRateLimitRequests),
		RefreshRateLimitWindowSec:  getEnvAsInt("REFRESH_RATE_LIMIT_WINDOW_SECONDS", defaultRefreshRateLimitWindowSec),
		InstanceName:               getEnv("APP_INSTANCE_NAME", defaultInstanceName),
	}

	applySecureDefaults(&cfg)
	validate(cfg)

	return cfg
}

func loadEnvFile() {
	_ = godotenv.Load()
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}

func getEnvAsInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsedValue, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsedValue
}

func getEnvAsBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsedValue, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsedValue
}

func getEnvAsCSV(key string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

func getEnvAsCSVWithDefault(key string, fallback []string) []string {
	values := getEnvAsCSV(key)
	if len(values) == 0 {
		return fallback
	}

	return values
}

func getEnvAsLogLevel(key, fallback string) slog.Level {
	switch strings.ToUpper(strings.TrimSpace(getEnv(key, fallback))) {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func normalizeAppEnv(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "prod":
		return "production"
	case "stage":
		return "staging"
	default:
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			return defaultAppEnv
		}
		return normalized
	}
}

func applySecureDefaults(cfg *Config) {
	if cfg.AppEnv == "production" {
		cfg.CookieSecure = true
		if strings.EqualFold(strings.TrimSpace(cfg.DBSSLMode), "disable") {
			cfg.DBSSLMode = "require"
		}
	}
}

func validate(cfg Config) {
	if strings.TrimSpace(cfg.JWTAccessSecret) == "" {
		panic("JWT_ACCESS_SECRET must not be empty")
	}

	if cfg.JWTAccessTTLMinutes <= 0 {
		panic("JWT_ACCESS_TTL_MINUTES must be greater than zero")
	}

	if cfg.RefreshTokenTTLHours <= 0 {
		panic("REFRESH_TOKEN_TTL_HOURS must be greater than zero")
	}

	if cfg.RequestTimeoutSec <= 0 {
		panic("REQUEST_TIMEOUT_SECONDS must be greater than zero")
	}

	if cfg.ShutdownTimeoutSec <= 0 {
		panic("SHUTDOWN_TIMEOUT_SECONDS must be greater than zero")
	}

	if strings.EqualFold(strings.TrimSpace(cfg.CookieSameSite), "none") && !cfg.CookieSecure {
		panic("COOKIE_SAMESITE=None requires COOKIE_SECURE=true")
	}

	if cfg.CORSAllowCredentials && len(cfg.CORSAllowedOrigins) == 0 {
		panic("CORS_ALLOW_CREDENTIALS requires explicit CORS_ALLOWED_ORIGINS")
	}

	if cfg.AppEnv == "production" {
		if cfg.JWTAccessSecret == defaultJWTSecret || len(cfg.JWTAccessSecret) < 32 {
			panic("production requires a strong JWT_ACCESS_SECRET with at least 32 characters")
		}
	}
}
