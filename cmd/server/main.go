package main

import (
	"context"
	"github.com/TimonKK/inmemory-db/internal/config"
	"github.com/TimonKK/inmemory-db/internal/database"
	"github.com/TimonKK/inmemory-db/internal/database/compute"
	"github.com/TimonKK/inmemory-db/internal/database/network"
	"github.com/TimonKK/inmemory-db/internal/database/storage"
	"github.com/TimonKK/inmemory-db/internal/database/storage/engine"
	"github.com/TimonKK/inmemory-db/internal/logger"
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
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("failed to initialize config: %v", err)
	}

	l, err := logger.NewLogger(&cfg.Logging)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	l.Info("Loading config", zap.Any("config", cfg))

	computeInstance := compute.NewCompute(l)
	engineInstance, err := engine.NewEngine(cfg.Engine.Type)
	if err != nil {
		l.Fatal("Failed to init engine", zap.Error(err), zap.String("type", cfg.Engine.Type))
	}
	storageInstance := storage.NewStorage(engineInstance, l)

	db := database.NewDatabase(computeInstance, storageInstance, l)

	server, err := network.NewTCPServer(cfg.Network, l)
	if err != nil {
		l.Fatal("Failed to init server", zap.Error(err))
	}

	server.HandleConnect(ctx, func(ctx context.Context, query string) string {
		res, err := db.ExecQuery(ctx, query)

		if err != nil {
			l.Error("Failed to execute query", zap.Error(err), zap.String("query", string(query)))
			return res
		}

		return res
	})

	if err := server.Start(); err != nil {
		l.Fatal("Failed to start server", zap.Error(err))
	}

	<-ctx.Done()

	if err := server.Shutdown(ctx); err != nil {
		l.Fatal("Failed to shutdown server", zap.Error(err))
	}
}
