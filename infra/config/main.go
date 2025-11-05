package config

import (
	"fmt"

	"github.com/joho/godotenv"
)

func checkEnvs() error {
	err := checkXrayEnvs()
	if err != nil {
		return err
	}
	err = checkRedisEnvs()
	if err != nil {
		return err
	}
	err = checkTimeEnvs()
	if err != nil {
		return err
	}
	return nil
}

func ConfigureEnvs() error {
	err := checkEnvs()

	if err != nil {
		err = godotenv.Overload()
		if err != nil {
			return fmt.Errorf("error while loading .env file: %v", err)
		}

		err = checkEnvs()
		if err != nil {
			return fmt.Errorf("error while reading env: %v", err)
		}
	}
	return nil
}
