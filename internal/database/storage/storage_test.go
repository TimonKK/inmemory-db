package storage

import (
	"context"
	"errors"
	"github.com/stretchr/testify/require"
	"testing"

	"iter"

	"github.com/TimonKK/inmemory-db/internal/database/compute"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestStorage_RecoveryData(t *testing.T) {
	ctx := context.Background()
	engine := new(mockEngine)
	wal := new(mockWAL)
	logger := zap.NewNop()

	// WAL возвращает две команды: SET и DELETE
	wal.On("All").Return(iter.Seq2[string, error](func(yield func(string, error) bool) {
		yield("SET key1 value1", nil)
		yield("DEL key2", nil)
	}))

	engine.On("Set", ctx, "key1", "value1").Return(nil)
	engine.On("Delete", ctx, "key2").Return(nil)

	st := &Storage{engine: engine, wal: wal, logger: logger}
	err := st.recoveryData(ctx)
	assert.NoError(t, err)
	engine.AssertExpectations(t)
}

func TestStorage_Get_Set_Delete(t *testing.T) {
	ctx := context.Background()
	engine := new(mockEngine)
	wal := new(mockWAL)
	logger := zap.NewNop()

	s, err := NewStorage(engine, wal, logger)
	require.NoError(t, err)

	// --- SET ---
	querySet := compute.NewQuery(compute.SetCommandId, []string{"k", "v"})
	wal.On("Push", querySet.String()).Return(nil)
	engine.On("Set", ctx, "k", "v").Return(nil)

	err = s.Set(ctx, querySet)
	assert.NoError(t, err)
	wal.AssertExpectations(t)
	engine.AssertExpectations(t)

	// --- GET ---
	queryGet := compute.NewQuery(compute.GetCommandId, []string{"k"})
	engine.On("Get", ctx, "k").Return("v", nil)

	val, err := s.Get(ctx, queryGet)
	assert.NoError(t, err)
	assert.Equal(t, "v", val)
	engine.AssertExpectations(t)

	// --- DELETE ---
	queryDel := compute.NewQuery(compute.DeleteCommandId, []string{"k"})
	wal.On("Push", queryDel.String()).Return(nil)
	engine.On("Delete", ctx, "k").Return(nil)

	err = s.Delete(ctx, queryDel)
	assert.NoError(t, err)
	wal.AssertExpectations(t)
	engine.AssertExpectations(t)
}

func TestStorage_Start_RecoveryError(t *testing.T) {
	ctx := context.Background()
	engine := new(mockEngine)
	wal := new(mockWAL)
	logger := zap.NewNop()

	s, err := NewStorage(engine, wal, logger)
	require.NoError(t, err)

	// recoveryData вернёт ошибку
	wal.On("All").Return(iter.Seq2[string, error](func(yield func(string, error) bool) {
		yield("", errors.New("fail"))
	}))

	err = s.Start(ctx)
	assert.Error(t, err)
}
