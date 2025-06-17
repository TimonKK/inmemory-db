package storage

import (
	"context"
	"go.uber.org/zap"
)

type Engine interface {
	Get(context.Context, string) (string, error)
	Set(context.Context, string, string) error
	Delete(context.Context, string) error
}

type Storage struct {
	engine Engine
	logger *zap.Logger
}

func NewStorage(engine Engine, logger *zap.Logger) *Storage {
	return &Storage{
		engine: engine,
		logger: logger,
	}
}

func (s *Storage) Get(ctx context.Context, key string) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	return s.engine.Get(ctx, key)
}

func (s *Storage) Set(ctx context.Context, key, value string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return s.engine.Set(ctx, key, value)
}
func (s *Storage) Delete(ctx context.Context, key string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	return s.engine.Delete(ctx, key)
}
