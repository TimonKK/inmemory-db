package wal

import (
	"fmt"
	"github.com/TimonKK/inmemory-db/internal/database/fs"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestChunkManager_GetWalFiles(t *testing.T) {
	logger := zap.NewNop()

	ids := []int{0, 1, 11, 13, 4, 5, 6, 7, 8, 9, 10, 2, 12, 3, 14, 15}

	files := make([]os.DirEntry, 0, len(ids))
	expectedFiles := make([]string, len(ids))
	for i, walId := range ids {
		expectedFiles[i] = fmt.Sprintf(FormatWalFilename, i)

		files = append(files, fs.NewMockDirEntry(fmt.Sprintf(FormatWalFilename, walId), 1000))
	}

	tests := []struct {
		name            string
		mockFileStorage func(*fs.MockFileStorage)
	}{
		{
			name: "should get sorted list of wal files",
			mockFileStorage: func(m *fs.MockFileStorage) {
				m.On("ReadDir", "wal").Return(files, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockFileStorage := new(fs.MockFileStorage)
			tt.mockFileStorage(mockFileStorage)

			cm := NewChunkManager(mockFileStorage, "wal", 1000, logger)

			f, err := cm.GetWalFiles()
			require.NoError(t, err)
			assert.Equal(t, expectedFiles, f)
		})
	}
}
