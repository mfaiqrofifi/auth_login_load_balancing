package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimitResult struct {
	Allowed   bool
	Remaining int64
	ResetIn   time.Duration
}

type RateLimitService struct {
	client *redis.Client
}

func NewRateLimitService(client *redis.Client) *RateLimitService {
	return &RateLimitService{client: client}
}

func (s *RateLimitService) Allow(ctx context.Context, scope, identifier string, limit int, window time.Duration) (RateLimitResult, error) {
	key := fmt.Sprintf("auth:rate_limit:%s:%s", scope, identifier)

	count, err := s.client.Incr(ctx, key).Result()
	if err != nil {
		return RateLimitResult{}, err
	}

	if count == 1 {
		if err := s.client.Expire(ctx, key, window).Err(); err != nil {
			return RateLimitResult{}, err
		}
	}

	ttl, err := s.client.TTL(ctx, key).Result()
	if err != nil {
		return RateLimitResult{}, err
	}
	if ttl < 0 {
		ttl = window
	}

	remaining := int64(limit) - count
	if remaining < 0 {
		remaining = 0
	}

	return RateLimitResult{
		Allowed:   count <= int64(limit),
		Remaining: remaining,
		ResetIn:   ttl,
	}, nil
}
