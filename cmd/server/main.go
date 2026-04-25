package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"plants-monitor/internal/alerts"
	"plants-monitor/internal/bot"
	"plants-monitor/internal/config"
	"plants-monitor/internal/httpapi"
	"plants-monitor/internal/logger"
	"plants-monitor/internal/storage"
	"syscall"
	"time"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logg := logger.New(cfg.AppEnv)

	logg.Info(
		"config loaded",
		slog.String("app_env", cfg.AppEnv),
		slog.String("http_addr", cfg.HTTPAddr),
		slog.String("db_path", cfg.DBPath),
		slog.Int("soil_low_percent", cfg.SoilLowPercent),
		slog.Float64("light_low_voltage", cfg.LightLowVoltage),
	)

	db, err := storage.Open(cfg.DBPath)
	if err != nil {
		logg.Error("open storage failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	botService := bot.NewService(db, logg)

	alertEngine := alerts.NewEngine(
		db,
		botService,
		cfg.SoilLowPercent,
		cfg.LightLowVoltage,
		logg,
	)

	handler := httpapi.NewHandler(db, alertEngine, logg)
	router := httpapi.NewRouter(handler, logg)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("server listening on %s", cfg.HTTPAddr)

		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen and serve: %v", err)
		}
	}()

	go func() {
		if err := botService.Start(ctx, cfg.TelegramBotToken); err != nil {
			log.Printf("telegram bot error: %v", err)
			stop()
		}
	}()

	<-ctx.Done()

	log.Println("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown error: %v", err)
	}

	log.Println("server stopped")
}
