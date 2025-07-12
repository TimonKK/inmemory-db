package fs

import (
	"io/fs"
	"iter"
	"os"
	"time"

	"github.com/stretchr/testify/mock"
)

var _ os.FileInfo = (*MockFileInfo)(nil)
var _ os.DirEntry = (*MockDirEntry)(nil)

type MockFileInfo struct {
	name string
	size int64
}

func (m MockFileInfo) Name() string       { return m.name }
func (m MockFileInfo) Size() int64        { return m.size }
func (m MockFileInfo) Mode() fs.FileMode  { return 0 }
func (m MockFileInfo) ModTime() time.Time { return time.Time{} }
func (m MockFileInfo) IsDir() bool        { return false }
func (m MockFileInfo) Sys() interface{}   { return nil }

type MockDirEntry struct {
	name string
	size int
}

func NewMockDirEntry(name string, size int) *MockDirEntry {
	return &MockDirEntry{name, size}
}

func (m *MockDirEntry) Info() (fs.FileInfo, error) {
	return MockFileInfo{name: m.name, size: int64(m.size)}, nil
}
func (m *MockDirEntry) Type() fs.FileMode { return fs.ModeDir }
func (m *MockDirEntry) Name() string      { return m.name }
func (m *MockDirEntry) IsDir() bool       { return false }

type MockFile struct {
	WriteFunc func([]byte) (int, error)
	CloseFunc func() error
}

func (m *MockFile) Write(p []byte) (int, error) {
	if m.WriteFunc != nil {
		return m.WriteFunc(p)
	}
	return len(p), nil
}

func (m *MockFile) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

type MockFileStorage struct {
	mock.Mock
}

func (m *MockFileStorage) OpenFile(filePath string) (File, error) {
	args := m.Called(filePath)
	return args.Get(0).(File), args.Error(1)
}

func (m *MockFileStorage) WriteFile(filePath string, data []byte) error {
	args := m.Called(filePath, data)
	return args.Error(1)
}

func (m *MockFileStorage) ReadDir(dir string) ([]os.DirEntry, error) {
	args := m.Called(dir)

	return args.Get(0).([]os.DirEntry), args.Error(1)
}

func (m *MockFileStorage) Lines(filePath string) iter.Seq2[string, error] {
	args := m.Called(filePath)
	return args.Get(0).(iter.Seq2[string, error])
}

func (m *MockFileStorage) Size() int {
	return 0
}

func (m *MockFileStorage) Flush() error {
	return nil
}

func (m *MockFileStorage) Reset() error {
	return nil
}
