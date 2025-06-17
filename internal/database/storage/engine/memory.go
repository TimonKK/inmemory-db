package engine

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrKeyNotFound = errors.New("key not found")
)

type MemoryEngine struct {
	m    sync.RWMutex
	data map[string]string
}

func NewMemoryEngine() *MemoryEngine {
	return &MemoryEngine{
		data: make(map[string]string),
	}
}

func (e *MemoryEngine) Get(_ context.Context, key string) (string, error) {
	e.m.RLock()
	defer e.m.RUnlock()

	value, ok := e.data[key]
	if ok {
		return value, nil
	}

	return "", ErrKeyNotFound
}

func (e *MemoryEngine) Set(_ context.Context, key string, value string) error {
	e.m.Lock()
	defer e.m.Unlock()

	e.data[key] = value

	return nil
}

func (e *MemoryEngine) Delete(_ context.Context, key string) error {
	e.m.Lock()
	defer e.m.Unlock()

	delete(e.data, key)

	return nil
}
