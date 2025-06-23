package wal

import (
	"context"
	"github.com/TimonKK/inmemory-db/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"os"
	"strconv"
	"testing"
	"time"
)

func TestWal_Push(t *testing.T) {
	logger := zap.NewNop()
	tests := []struct {
		name      string
		pushCount int
		config    *config.WALConfig
	}{
		{
			name:      "should resolve promise after timeout flush",
			pushCount: 5,
			config: &config.WALConfig{
				FlushingBatchSize:    10,
				FlushingBatchTimeout: 1 * time.Millisecond,
				MaxSegmentSize:       1000000,
				DataDirectory:        os.TempDir(),
			},
		},

		{
			name:      "should resolve promise after batch len flush",
			pushCount: 15,
			config: &config.WALConfig{
				FlushingBatchSize:    10,
				FlushingBatchTimeout: 1 * time.Millisecond,
				MaxSegmentSize:       1000000,
				DataDirectory:        os.TempDir(),
			},
		},

		{
			name:      "should resolve promise after batch size flush",
			pushCount: 5,
			config: &config.WALConfig{
				FlushingBatchSize:    10,
				FlushingBatchTimeout: 1 * time.Millisecond,
				MaxSegmentSize:       10,
				DataDirectory:        os.TempDir(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, ctx := errgroup.WithContext(context.Background())

			wal := NewWAL(tt.config, logger)
			err := wal.Start(ctx)
			require.NoError(t, err)

			for i := range tt.pushCount {
				g.Go(func() error {
					return wal.Push("promise" + strconv.Itoa(i))
				})
			}

			err = g.Wait()
			assert.NoError(t, err)
		})
	}
}
