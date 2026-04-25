package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPAddr         string
	DBPath           string
	TelegramBotToken string

	SoilLowPercent  int
	LightLowVoltage float64
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

	soilLowPercent, err := optionalIntEnv("SOIL_LOW_PERCENT", 30)
	if err != nil {
		return Config{}, err
	}

	lightLowVoltage, err := optionalFloatEnv("LIGHT_LOW_VOLTAGE", 0.8)
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPAddr:         httpAddr,
		DBPath:           dbPath,
		TelegramBotToken: telegramBotToken,
		SoilLowPercent:   soilLowPercent,
		LightLowVoltage:  lightLowVoltage,
	}, nil
}

func requiredEnv(key string) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return "", fmt.Errorf("environment variable %s is required", key)
	}

	return value, nil
}

func optionalIntEnv(key string, defaultValue int) (int, error) {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s as int: %w", key, err)
	}

	return parsed, nil
}

func optionalFloatEnv(key string, defaultValue float64) (float64, error) {
	value, ok := os.LookupEnv(key)
	if !ok || value == "" {
		return defaultValue, nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("parse %s as float: %w", key, err)
	}

	return parsed, nil
}
