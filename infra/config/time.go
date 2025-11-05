package config

import (
	"errors"
	"os"
)

func checkTimeEnvs() error {
	_, ok := os.LookupEnv("TIMEZONE")
	if !ok {
		return errors.New("TIMEZONE not specified")
	}

	return nil
}
