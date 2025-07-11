package engine

import (
	"context"
	"errors"
	"hash/fnv"
	"sync"
)

const dataShartNum = 1024

var (
	ErrKeyNotFound = errors.New("key not found")
)

type dataShard struct {
	m    sync.RWMutex
	data map[string]string
}

type MemoryEngine struct {
	shards []*dataShard
}

func NewMemoryEngine() *MemoryEngine {
	shards := make([]*dataShard, dataShartNum)
	for i := range shards {
		shards[i] = &dataShard{
			data: make(map[string]string),
		}
	}

	return &MemoryEngine{
		shards: shards,
	}
}

func (e *MemoryEngine) getShardId(key string) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return int(h.Sum32() % dataShartNum)
}

func (e *MemoryEngine) Get(_ context.Context, key string) (string, error) {
	shard := e.shards[e.getShardId(key)]
	shard.m.RLock()
	defer shard.m.RUnlock()

	value, ok := shard.data[key]
	if ok {
		return value, nil
	}

	return "", ErrKeyNotFound
}

func (e *MemoryEngine) Set(_ context.Context, key string, value string) error {
	shard := e.shards[e.getShardId(key)]
	shard.m.Lock()
	defer shard.m.Unlock()

	shard.data[key] = value

	return nil
}

func (e *MemoryEngine) Delete(_ context.Context, key string) error {
	shard := e.shards[e.getShardId(key)]
	shard.m.Lock()
	defer shard.m.Unlock()

	delete(shard.data, key)

	return nil
}
