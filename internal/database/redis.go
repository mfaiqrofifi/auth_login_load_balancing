package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"load_balancing_project_auth/internal/config"
)

func NewRedisClient(cfg config.Config) (*redis.Client, error) {
	options := &redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	}

	if cfg.RedisURL != "" {
		parsedOptions, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			return nil, err
		}
		options = parsedOptions
	}

	client := redis.NewClient(options)

	if err := client.Ping(context.Background()).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	return client, nil
}
