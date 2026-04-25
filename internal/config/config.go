package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPAddr         string
	DBPath           string
	TelegramBotToken string
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return Config{}, fmt.Errorf("load .env: %w", err)
	}

	httpAddr, err := requiredEnv("HTTP_ADDR")
	if err != nil {
		return Config{}, err
	}

	dbPath, err := requiredEnv("DB_PATH")
	if err != nil {
		return Config{}, err
	}

	telegramBotToken, err := requiredEnv("TELEGRAM_BOT_TOKEN")
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPAddr:         httpAddr,
		DBPath:           dbPath,
		TelegramBotToken: telegramBotToken,
	}, nil
}

func requiredEnv(key string) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return "", fmt.Errorf("environment variable %s is required", key)
	}

	return value, nil
}
