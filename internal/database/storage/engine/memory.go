package engine

import "errors"

var (
	ErrKeyNotFound = errors.New("key not found")
)

type MemoryEngine struct {
	data map[string]string
}

func NewMemoryEngine() *MemoryEngine {
	return &MemoryEngine{
		data: make(map[string]string),
	}
}

func (memoryEngine *MemoryEngine) Get(key string) (string, error) {
	value, ok := memoryEngine.data[key]
	if ok {
		return value, nil
	}

	return "", ErrKeyNotFound
}

func (memoryEngine *MemoryEngine) Set(key string, value string) error {
	memoryEngine.data[key] = value

	return nil
}

func (memoryEngine *MemoryEngine) Delete(key string) error {
	delete(memoryEngine.data, key)

	return nil
}
