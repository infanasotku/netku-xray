package caching

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrEngineHashNotFound = errors.New("engine hash does not exist")

type XrayInfo struct {
	Created  string
	XrayUUID string
	Running  bool
	GRPCAddr string
}

type RedisXrayCachingClient struct {
	client   *redis.Client
	infoTTL  time.Duration
	engineID string
}

func getHashKey(uuid string) string {
	return "xrayEngines:" + uuid
}

func (c *RedisXrayCachingClient) RefreshTTL(context context.Context) error {
	hashKey := getHashKey(c.engineID)

	keys, err := c.client.Exists(context, hashKey).Result()

	if err != nil || keys == 0 {
		return ErrEngineHashNotFound
	}
	_, err = c.client.Expire(context, hashKey, c.infoTTL).Result()
	if err != nil {
		return fmt.Errorf("engine hash expiration not set: %v", err)
	}

	return nil
}

func (c *RedisXrayCachingClient) SetXrayInfo(context context.Context, info *XrayInfo) error {
	hashKey := getHashKey(c.engineID)
	running := strconv.FormatBool(info.Running)
	payload := map[string]any{
		"running": running,
		"created": info.Created,
		"addr":    info.GRPCAddr,
	}
	if info.XrayUUID != "" {
		payload["uuid"] = info.XrayUUID
	}

	// JSON-encode payload for stream; Redis field values cannot be maps
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("payload marshal failed: %v", err)
	}

	_, err = c.client.TxPipelined(context, func(pipe redis.Pipeliner) error {
		hsetArgs := make([]any, 0, len(payload)*2)
		for k, v := range payload {
			hsetArgs = append(hsetArgs, k, v)
		}
		pipe.HSet(context, hashKey, hsetArgs...)
		pipe.XAdd(context, &redis.XAddArgs{Stream: "xray_engines_keyevent_stream", Values: map[string]any{"event": "hset", "key": hashKey, "payload": string(payloadJSON)}})
		return nil
	})

	if err != nil {
		return fmt.Errorf("xray info not set: %v", err)
	}
	_, err = c.client.Expire(context, hashKey, c.infoTTL).Result()
	if err != nil {
		return fmt.Errorf("engine hash expiration not set: %v", err)
	}

	return nil
}
