package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tkozakas/sea-battle-server/internal/config"
	"github.com/tkozakas/sea-battle-server/internal/repository"
	"github.com/tkozakas/sea-battle-server/internal/service"
	"github.com/tkozakas/sea-battle-server/internal/transport"
)

func main() {
	cfg := config.Load()

	logLevel := slog.LevelInfo
	if cfg.LogLevel == "debug" {
		logLevel = slog.LevelDebug
	} else if cfg.LogLevel == "warn" {
		logLevel = slog.LevelWarn
	} else if cfg.LogLevel == "error" {
		logLevel = slog.LevelError
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	repo := repository.NewMemoryGameRepository()
	svc := service.NewGameService(repo)
	handler := transport.NewHandler(svc)
	router := transport.NewRouter(handler)

	rooms := service.NewRoomManager(repo)

	cleanupTicker := time.NewTicker(cfg.RoomCleanupInterval)
	stopCleanup := make(chan struct{})
	go func() {
		for {
			select {
			case <-cleanupTicker.C:
				rooms.CleanupStaleRooms(cfg.ReconnectGrace, cfg.ReconnectGrace)
				slog.Debug("cleanup ran", "active_rooms", svc.ActiveRoomCount())
			case <-stopCleanup:
				return
			}
		}
	}()

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}

	slog.Info("starting server", "port", cfg.Port, "log_level", cfg.LogLevel)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-sigCh
	slog.Info("shutting down server")

	cleanupTicker.Stop()
	close(stopCleanup)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "error", err)
	}

	slog.Info("server stopped")
}
