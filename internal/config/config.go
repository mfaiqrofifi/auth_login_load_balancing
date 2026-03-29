package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

const (
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
	defaultRedisHost                  = "localhost"
	defaultRedisPort                  = "6379"
	defaultRedisDB                    = 0
	defaultCookieSecure               = false
	defaultCookieSameSite             = "Lax"
	defaultLoginRateLimitRequests     = 5
	defaultLoginRateLimitWindowSec    = 60
	defaultRegisterRateLimitRequests  = 3
	defaultRegisterRateLimitWindowSec = 60
	defaultRefreshRateLimitRequests   = 10
	defaultRefreshRateLimitWindowSec  = 60
	defaultInstanceName               = "app-instance"
)

type Config struct {
	Port                       string
	AppName                    string
	DBHost                     string
	DBPort                     string
	DBUser                     string
	DBPassword                 string
	DBName                     string
	DBSSLMode                  string
	JWTAccessSecret            string
	JWTAccessTTLMinutes        int
	RefreshTokenTTLHours       int
	RedisHost                  string
	RedisPort                  string
	RedisPassword              string
	RedisDB                    int
	CookieDomain               string
	CookieSecure               bool
	CookieSameSite             string
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

	return Config{
		Port:                       getEnv("PORT", defaultPort),
		AppName:                    getEnv("APP_NAME", defaultAppName),
		DBHost:                     getEnv("DB_HOST", defaultDBHost),
		DBPort:                     getEnv("DB_PORT", defaultDBPort),
		DBUser:                     getEnv("DB_USER", defaultDBUser),
		DBPassword:                 getEnv("DB_PASSWORD", defaultDBPassword),
		DBName:                     getEnv("DB_NAME", defaultDBName),
		DBSSLMode:                  getEnv("DB_SSLMODE", defaultDBSSLMode),
		JWTAccessSecret:            getEnv("JWT_ACCESS_SECRET", defaultJWTSecret),
		JWTAccessTTLMinutes:        getEnvAsInt("JWT_ACCESS_TTL_MINUTES", defaultJWTTTL),
		RefreshTokenTTLHours:       getEnvAsInt("REFRESH_TOKEN_TTL_HOURS", defaultRefreshTTL),
		RedisHost:                  getEnv("REDIS_HOST", defaultRedisHost),
		RedisPort:                  getEnv("REDIS_PORT", defaultRedisPort),
		RedisPassword:              getEnv("REDIS_PASSWORD", ""),
		RedisDB:                    getEnvAsInt("REDIS_DB", defaultRedisDB),
		CookieDomain:               getEnv("COOKIE_DOMAIN", ""),
		CookieSecure:               getEnvAsBool("COOKIE_SECURE", defaultCookieSecure),
		CookieSameSite:             getEnv("COOKIE_SAMESITE", defaultCookieSameSite),
		LoginRateLimitRequests:     getEnvAsInt("LOGIN_RATE_LIMIT_REQUESTS", defaultLoginRateLimitRequests),
		LoginRateLimitWindowSec:    getEnvAsInt("LOGIN_RATE_LIMIT_WINDOW_SECONDS", defaultLoginRateLimitWindowSec),
		RegisterRateLimitRequests:  getEnvAsInt("REGISTER_RATE_LIMIT_REQUESTS", defaultRegisterRateLimitRequests),
		RegisterRateLimitWindowSec: getEnvAsInt("REGISTER_RATE_LIMIT_WINDOW_SECONDS", defaultRegisterRateLimitWindowSec),
		RefreshRateLimitRequests:   getEnvAsInt("REFRESH_RATE_LIMIT_REQUESTS", defaultRefreshRateLimitRequests),
		RefreshRateLimitWindowSec:  getEnvAsInt("REFRESH_RATE_LIMIT_WINDOW_SECONDS", defaultRefreshRateLimitWindowSec),
		InstanceName:               getEnv("APP_INSTANCE_NAME", defaultInstanceName),
	}
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
