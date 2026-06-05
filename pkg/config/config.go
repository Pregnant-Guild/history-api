package config

import (
	"fmt"
	"history-api/assets"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

func LoadEnv() error {
	envData, err := assets.GetFileContent("resources/.env")
	if err != nil {
		return nil
	}
	envMap, err := godotenv.Parse(strings.NewReader(envData))
	if err != nil {
		return fmt.Errorf("error parsing .env content: %w", err)
	}

	for key, value := range envMap {
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
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

func GetIntConfigWithDefault(config string, defaultValue int) int {
	data := strings.TrimSpace(os.Getenv(config))
	if data == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(data)
	if err != nil {
		return defaultValue
	}
	return value
}

func GetBoolConfigWithDefault(config string, defaultValue bool) bool {
	data := strings.TrimSpace(os.Getenv(config))
	if data == "" {
		return defaultValue
	}

	value, err := strconv.ParseBool(data)
	if err != nil {
		return defaultValue
	}
	return value
}
