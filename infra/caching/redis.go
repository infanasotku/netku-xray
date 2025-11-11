package caching

import (
	"time"

	"github.com/redis/go-redis/v9"
)

func CreateRedisXrayCachingClient(addr string, password string, infoTTL time.Duration, engineID string) (*RedisXrayCachingClient, error) {
	client := redis.NewClient(&redis.Options{Addr: addr, Password: password})
	return &RedisXrayCachingClient{client: client, engineID: engineID, infoTTL: infoTTL}, nil
}
