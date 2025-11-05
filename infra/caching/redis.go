package caching

import (
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func CreateRedisXrayCachingClient(addr string, password string, infoTTL time.Duration) (*RedisXrayCachingClient, error) {
	client := redis.NewClient(&redis.Options{Addr: addr, Password: password})
	return &RedisXrayCachingClient{client: client, engineID: uuid.New().String(), infoTTL: infoTTL}, nil
}
