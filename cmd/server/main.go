package main

import (
	"context"
	"github.com/TimonKK/inmemory-db/internal/config"
	"github.com/TimonKK/inmemory-db/internal/logger"
	"github.com/TimonKK/inmemory-db/internal/server"
	"go.uber.org/zap"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	configPath := os.Getenv("CONFIG_PATH")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("failed to initialize config: %v", err)
	}

	l, err := logger.NewLogger(&cfg.Logging)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	l.Info("Loading config", zap.Any("config", cfg))

	srv, err := server.NewServer(cfg, l)
	if err != nil {
		l.Fatal("failed to initialize server: %v", zap.Error(err))
	}

	if err := srv.Start(ctx); err != nil {
		l.Fatal("Failed to start server", zap.Error(err))
	}

	<-ctx.Done()

	if err := srv.Shutdown(); err != nil {
		l.Fatal("Failed to shutdown server", zap.Error(err))
	}
}
