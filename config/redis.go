package config

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	ctx    context.Context
	client *redis.Client
}

func NewRedisClient(ctx context.Context, url string) (*RedisClient, error) {
	opt, err := redis.ParseURL(url)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opt)
	return &RedisClient{
		ctx:    ctx,
		client: client,
	}, nil
}

func (r *RedisClient) SetVal(key string, val string) error {
	return r.client.Set(r.ctx, key, val, 0).Err()
}

func (r *RedisClient) GetVal(key string) (string, error) {
	val, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		return "Unable to fetch", err
	}
	return val, nil
}
