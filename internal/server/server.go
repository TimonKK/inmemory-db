package server

import (
	"context"
	"errors"
	"github.com/TimonKK/inmemory-db/internal/config"
	"github.com/TimonKK/inmemory-db/internal/database"
	"github.com/TimonKK/inmemory-db/internal/database/compute"
	"github.com/TimonKK/inmemory-db/internal/database/network"
	"github.com/TimonKK/inmemory-db/internal/database/storage"
	"github.com/TimonKK/inmemory-db/internal/database/storage/engine"
	"github.com/TimonKK/inmemory-db/internal/database/storage/wal"
	"go.uber.org/zap"
)

type Server struct {
	tcpServer *network.TCPServer
	db        *database.Database
	logger    *zap.Logger
}

func NewServer(config *config.Config, logger *zap.Logger) (*Server, error) {
	if config == nil {
		return nil, errors.New("config required")
	}

	if logger == nil {
		return nil, errors.New("logger required")
	}

	computeInstance := compute.NewCompute(logger)
	engineInstance, err := engine.NewEngine(config.Engine.Type)
	if err != nil {
		logger.Fatal("Failed to init engine", zap.Error(err), zap.String("type", config.Engine.Type))
	}

	w := wal.NewWAL(&config.Wal, logger)
	storageInstance, err := storage.NewStorage(engineInstance, w, logger)
	if err != nil {
		logger.Fatal("Failed to init storage", zap.Error(err))
	}

	db := database.NewDatabase(computeInstance, storageInstance, logger)

	tcpServer, err := network.NewTCPServer(config.Network, logger)
	if err != nil {
		logger.Fatal("Failed to init server", zap.Error(err))
	}

	server := &Server{
		tcpServer: tcpServer,
		db:        db,
		logger:    logger,
	}

	return server, nil
}

func (s *Server) Handlers(ctx context.Context) {
	s.tcpServer.HandleConnect(ctx, func(ctx context.Context, query string) (string, error) {
		res, err := s.db.ExecQuery(ctx, query)

		if err != nil {
			return "", err
		}

		return res, nil
	})
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting server")

	err := s.db.Start(ctx)
	if err != nil {
		return err
	}

	err = s.tcpServer.Start()
	if err != nil {
		s.logger.Error("Failed to start server", zap.Error(err))
		return err
	}

	s.Handlers(ctx)

	return nil
}

func (s *Server) Shutdown() error {
	s.logger.Info("Shutting down server")
	err := s.tcpServer.Shutdown()
	if err != nil {
		s.logger.Error("Failed to shutdown server", zap.Error(err))
		return err
	}

	return nil
}
