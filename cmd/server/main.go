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
	if err := run(); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()

	setupLogger(cfg.LogLevel)

	repo := repository.NewMemoryGameRepository()
	rooms := service.NewRoomManager(repo, cfg.MaxRooms)
	svc := service.NewGameService(repo, rooms)
	handler := transport.NewHandler(svc, cfg)
	router := transport.NewRouter(handler)

	stopCleanup := startCleanupLoop(cfg, rooms, svc)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: router,
	}

	slog.Info("starting server", "port", cfg.Port, "log_level", cfg.LogLevel)

	serverErr := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		close(stopCleanup)
		return err
	case <-sigCh:
		slog.Info("shutting down server")
	}

	close(stopCleanup)

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	slog.Info("server stopped")
	return nil
}

func setupLogger(logLevel string) {
	level := slog.LevelInfo
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	slog.SetDefault(logger)
}

func startCleanupLoop(cfg *config.Config, rooms *service.RoomManager, svc *service.GameService) chan struct{} {
	stop := make(chan struct{})
	ticker := time.NewTicker(cfg.RoomCleanupInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				rooms.CleanupStaleRooms(cfg.ReconnectGrace, cfg.ReconnectGrace)
				slog.Debug("cleanup ran", "active_rooms", svc.ActiveRoomCount())
			case <-stop:
				return
			}
		}
	}()
	return stop
}
