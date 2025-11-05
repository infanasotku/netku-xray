package config

import (
	"errors"
	"os"
)

func checkRedisEnvs() error {
	_, ok := os.LookupEnv("REDIS_ADDR")
	if !ok {
		return errors.New("REDIS_ADDR not specified")
	}

	_, ok = os.LookupEnv("REDIS_PASS")
	if !ok {
		return errors.New("REDIS_PASS not specified")
	}

	_, ok = os.LookupEnv("ENGINE_TTL")
	if !ok {
		return errors.New("ENGINE_TTL not specified")
	}

	return nil
}
