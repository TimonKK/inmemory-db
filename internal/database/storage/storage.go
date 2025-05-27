package storage

import (
	"go.uber.org/zap"
)

type Engine interface {
	Get(string) (string, error)
	Set(string, string) error
	Delete(string) error
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

func (s *Storage) Get(key string) (string, error) {
	return s.engine.Get(key)
}

func (s *Storage) Set(key, value string) error {
	return s.engine.Set(key, value)
}
func (s *Storage) Delete(key string) error {
	return s.engine.Delete(key)
}
