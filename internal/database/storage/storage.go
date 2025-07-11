package storage

import (
	"context"
	"github.com/TimonKK/inmemory-db/internal/database/compute"
	"go.uber.org/zap"
	"iter"
)

type Engine interface {
	Get(context.Context, string) (string, error)
	Set(context.Context, string, string) error
	Delete(context.Context, string) error
}

type WAL interface {
	Start(context.Context) error
	All() iter.Seq2[string, error]
	Push(string) error
}

type Storage struct {
	engine Engine
	wal    WAL
	logger *zap.Logger
}

func NewStorage(engine Engine, wal WAL, logger *zap.Logger) (*Storage, error) {
	storage := Storage{
		engine: engine,
		wal:    wal,
		logger: logger,
	}

	return &storage, nil
}

func (s *Storage) pushToWal(query compute.Query) error {
	if s.wal == nil {
		return nil
	}

	return s.wal.Push(query.String())
}

func (s *Storage) recoveryData(ctx context.Context) error {
	for record, err := range s.wal.All() {
		if err != nil {
			return err
		}

		query, err := compute.NewQueryFromString(record)
		if err != nil {
			return err
		}

		switch query.CommandId() {
		case compute.SetCommandId:
			err = s.engine.Set(ctx, query.Key(), query.Value())
		case compute.DeleteCommandId:
			err = s.engine.Delete(ctx, query.Key())
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Storage) Start(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if s.wal == nil {
		return nil
	}

	err := s.recoveryData(ctx)
	if err != nil {
		return err
	}

	return s.wal.Start(ctx)
}

func (s *Storage) Get(ctx context.Context, query compute.Query) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	return s.engine.Get(ctx, query.Key())
}

func (s *Storage) Set(ctx context.Context, query compute.Query) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	err := s.pushToWal(query)
	if err != nil {
		return err
	}

	return s.engine.Set(ctx, query.Key(), query.Value())
}

func (s *Storage) Delete(ctx context.Context, query compute.Query) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	err := s.pushToWal(query)
	if err != nil {
		return err
	}

	return s.engine.Delete(ctx, query.Key())
}
