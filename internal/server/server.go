package server

import (
	"context"
	"errors"
	"github.com/TimonKK/inmemory-db/internal/config"
	"github.com/TimonKK/inmemory-db/internal/database"
	"github.com/TimonKK/inmemory-db/internal/database/compute"
	"github.com/TimonKK/inmemory-db/internal/database/fs"
	"github.com/TimonKK/inmemory-db/internal/database/network"
	"github.com/TimonKK/inmemory-db/internal/database/storage"
	"github.com/TimonKK/inmemory-db/internal/database/storage/engine"
	"github.com/TimonKK/inmemory-db/internal/database/storage/replication"
	"github.com/TimonKK/inmemory-db/internal/database/storage/wal"
	"go.uber.org/zap"
)

type Replication interface {
	Start(ctx context.Context) error
	Stop() error
	ValidateQuery(string2 string) error
}

type Server struct {
	tcpServer   *network.TCPServer
	replication Replication
	db          *database.Database
	logger      *zap.Logger
}

func NewServer(cfg *config.Config, logger *zap.Logger) (*Server, error) {
	if cfg == nil {
		return nil, errors.New("config required")
	}

	if logger == nil {
		return nil, errors.New("logger required")
	}

	computeInstance := compute.NewCompute(logger)
	engineInstance, err := engine.NewEngine(cfg.Engine.Type)
	if err != nil {
		logger.Fatal("Failed to init engine", zap.Error(err), zap.String("type", cfg.Engine.Type))
	}

	f := fs.NewFileStorage(cfg.Wal.DataDirectory, logger)
	cm := wal.NewChunkManager(f, cfg.Wal.DataDirectory, int(cfg.Wal.MaxSegmentSize), logger)
	w := wal.NewWAL(cm, &cfg.Wal, logger)
	storageInstance, err := storage.NewStorage(engineInstance, w, logger)
	if err != nil {
		logger.Fatal("Failed to init storage", zap.Error(err))
	}

	db := database.NewDatabase(computeInstance, storageInstance, logger)

	tcpServer, err := network.NewTCPServer(cfg.Network, logger)
	if err != nil {
		logger.Fatal("Failed to init server", zap.Error(err))
	}

	server := &Server{
		tcpServer: tcpServer,
		db:        db,
		logger:    logger,
	}

	if cfg.Replication != nil {
		server.replication = replication.NewReplication(cm, computeInstance, cfg.Replication, logger)
	}

	return server, nil
}

func (s *Server) Handlers(ctx context.Context) {
	s.tcpServer.HandleConnect(ctx, func(ctx context.Context, query string) (string, error) {

		// TODO вынести в middleware.
		// И в коде инициализации и просто создавать middleware или нет
		if s.replication != nil {
			if err := s.replication.ValidateQuery(query); err != nil {
				return "", err
			}
		}

		return s.db.ExecQuery(ctx, query)
	})
}

func (s *Server) Start(ctx context.Context) error {
	s.logger.Info("Starting server")

	if s.replication != nil {
		if err := s.replication.Start(ctx); err != nil {
			s.logger.Error("Failed to start replication", zap.Error(err))
		}
	}

	if err := s.db.Start(ctx); err != nil {
		return err
	}

	if err := s.tcpServer.Start(); err != nil {
		s.logger.Error("Failed to start server", zap.Error(err))
		return err
	}

	s.Handlers(ctx)

	return nil
}

func (s *Server) Shutdown() error {
	s.logger.Info("Shutting down server")

	if s.replication != nil {
		if err := s.replication.Stop(); err != nil {
			s.logger.Error("Failed to stop replication", zap.Error(err))
		}
	}

	if err := s.tcpServer.Shutdown(); err != nil {
		s.logger.Error("Failed to shutdown server", zap.Error(err))
		return err
	}

	return nil
}
