package wal

import (
	"context"
	"fmt"
	"github.com/TimonKK/inmemory-db/internal/database/fs"
	"os"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/TimonKK/inmemory-db/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const MaxSegmentSize = 5 * 1024

func WalPathGenerator(dir string) func(int) string {
	return func(walId int) string {
		return path.Join(dir, fmt.Sprintf(FormatWalFilename, walId))
	}
}

func TestWal_Push(t *testing.T) {
	logger := zap.NewNop()

	walPathFn := WalPathGenerator("wal")

	ids := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	expectedFiles := make([]string, len(ids))
	for i := range ids {
		expectedFiles[i] = fmt.Sprintf(FormatWalFilename, i)
	}

	tests := []struct {
		name            string
		pushCount       int
		config          *config.WALConfig
		mockFileStorage func(*fs.MockFileStorage)
	}{
		{
			name:      "should resolve promise after timeout flush",
			pushCount: 5,
			config: &config.WALConfig{
				FlushingBatchSize:    10,
				FlushingBatchTimeout: 1 * time.Millisecond,
				MaxSegmentSize:       10,
				DataDirectory:        "wal",
			},
			mockFileStorage: func(m *fs.MockFileStorage) {
				files := make([]os.DirEntry, 1)
				files[0] = fs.NewMockDirEntry("wal/wal.0.log", 0)

				m.On("ReadDir", "wal").Return(files, nil)
				m.On("OpenFile", walPathFn(len(files)-1)).Return(&fs.MockFile{}, nil)
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
			mockFileStorage: func(m *fs.MockFileStorage) {
				files := make([]os.DirEntry, len(expectedFiles))
				for i := range files {
					files[i] = fs.NewMockDirEntry(expectedFiles[i], 1000)
				}

				m.On("ReadDir", "wal").Return(files, nil)
				m.On("OpenFile", walPathFn(len(files)-1)).Return(&fs.MockFile{}, nil)
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
			mockFileStorage: func(m *fs.MockFileStorage) {
				files := make([]os.DirEntry, len(expectedFiles))
				for i := range files {
					files[i] = fs.NewMockDirEntry(expectedFiles[i], 1000)
				}

				m.On("ReadDir", "wal").Return(files, nil)
				m.On("OpenFile", walPathFn(len(files)-1)).Return(&fs.MockFile{}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, ctx := errgroup.WithContext(context.Background())

			mockFileStorage := new(fs.MockFileStorage)
			tt.mockFileStorage(mockFileStorage)

			s := NewChunkManager(mockFileStorage, "wal", MaxSegmentSize, logger)
			wal := NewWAL(s, tt.config, logger)
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
