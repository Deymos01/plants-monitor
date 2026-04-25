package main

import "plants-monitor/internal/config"

func main() {
	cfg := config.Load()

	_ = cfg
}
