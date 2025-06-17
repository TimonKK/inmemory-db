package engine

import (
	"errors"
	"fmt"
	"github.com/TimonKK/inmemory-db/internal/database/storage"
)

var (
	ErrUnknowEngine = errors.New("unknow engine")
)

func NewEngine(t string) (storage.Engine, error) {
	if t == "in_memory" {
		return NewMemoryEngine(), nil
	}

	return nil, fmt.Errorf("%w: type %s", ErrUnknowEngine, t)
}
