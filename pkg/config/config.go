package config

import (
	"errors"
	"fmt"
	"history-api/assets"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

func LoadEnv() error {
	envData, err := assets.GetFileContent("resources/.env")
	if err != nil {
		return errors.New("error read .env file")
	}
	envMap, err := godotenv.Parse(strings.NewReader(envData))
	if err != nil {
		return errors.New("error parsing .env content")
	}

	for key, value := range envMap {
		os.Setenv(key, value)
	}
	return nil
}

func GetConfig(config string) (string, error) {
	var data string = os.Getenv(config)
	if data == "" {
		return "", fmt.Errorf("config (%s) dose not exit", config)
	}

	return data, nil
}

func GetConfigWithDefault(config, defaultValue string) string {
	var data string = os.Getenv(config)
	if data == "" {
		return defaultValue
	}
	return data
}