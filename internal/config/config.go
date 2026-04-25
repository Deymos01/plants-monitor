package config

import "os"

type Config struct {
	HTTPAddr string
	DBPath   string
}

func Load() Config {
	httpAddr := os.Getenv("HTTP_ADDR")
	if httpAddr == "" {
		httpAddr = ":8080"
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "data/plant_monitor.db"
	}

	return Config{
		HTTPAddr: httpAddr,
		DBPath:   dbPath,
	}
}
