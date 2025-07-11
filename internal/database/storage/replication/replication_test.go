package replication

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TimonKK/inmemory-db/internal/config"
	"github.com/TimonKK/inmemory-db/internal/database/compute"
	"github.com/TimonKK/inmemory-db/internal/database/fs"
	"github.com/TimonKK/inmemory-db/internal/database/storage/wal"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// mockTCPServer мок сервера
type mockTCPServer struct {
	listener net.Listener
	wg       sync.WaitGroup
	handler  func(conn net.Conn)
	address  string
}

// newMockTCPServer - создает мок сервера и слушает коннекты
func newMockTCPServer(t *testing.T, handler func(conn net.Conn)) *mockTCPServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0") // Listen on a random available port
	require.NoError(t, err, "Failed to start mock TCP server listener")

	server := &mockTCPServer{
		listener: listener,
		handler:  handler,
		address:  listener.Addr().String(),
	}

	server.wg.Add(1)
	go func() {
		for {
			defer server.wg.Done()
			conn, err := listener.Accept()
			if err != nil {
				if !strings.Contains(err.Error(), "use of closed network connection") {
					t.Logf("Mock server failed to accept connection: %v", err)
				}
				return
			}

			server.wg.Add(1)
			go func() {

				defer func() {
					_ = conn.Close()
				}()

				for {
					server.handler(conn)
				}
			}()
		}
	}()
	return server
}

func (s *mockTCPServer) Close() {
	if s.listener != nil {
		_ = s.listener.Close()
	}
	s.wg.Wait()
}

func TestReplication_ReplicationList(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	mockFileStorage := new(fs.MockFileStorage)

	// Мастер возвращает список файлов
	masterWals := []string{"wal.0.log", "wal.1.log"}

	files := make([]os.DirEntry, 0, len(masterWals))
	for _, walFile := range masterWals {
		files = append(files, fs.NewMockDirEntry(walFile, 1000))
	}

	mockFileStorage.On("ReadDir", "wal").Return(files, nil)

	c := compute.NewCompute(logger)
	cm := wal.NewChunkManager(mockFileStorage, "wal", 1000, logger)

	done := make(chan struct{})
	serverHandler := func(conn net.Conn) {
		fmt.Println("REQUEST1")
		query, err := bufio.NewReader(conn).ReadString('\n')
		require.NoError(t, err)

		assert.Equal(t, string(compute.ReplicationCommandId)+"\n", query)

		_, err = conn.Write([]byte(strings.Join(masterWals, ",")))
		require.NoError(t, err)

		close(done)
	}
	mockServer := newMockTCPServer(t, serverHandler)
	defer mockServer.Close()

	cfg := config.ReplicationConfig{
		ReplicaType:       ReplicaTypeSlave,
		MasterAddress:     mockServer.address,
		SyncInterval:      1 * time.Millisecond,
		MaxReplicasNumber: 10,
	}
	r := NewReplication(cm, c, &cfg, logger)

	err := r.Start(ctx)
	assert.NoError(t, err)

	// Ждём сигнал с таймаутом, чтобы тест не завис
	select {
	case <-done:
		// Успешно получили сигнал
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for replication request")
	}
}

func TestReplication_ReplicationFile(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()
	mockFileStorage := new(fs.MockFileStorage)
	c := compute.NewCompute(logger)
	cm := wal.NewChunkManager(mockFileStorage, "wal", 1000, logger)

	// Мастер возвращает список файлов
	masterWals := []string{"wal.0.log", "wal.1.log"}

	files := make([]os.DirEntry, 0, len(masterWals))
	for _, walFile := range masterWals {
		files = append(files, fs.NewMockDirEntry(walFile, 1000))
	}

	// локально только один wal-файл, а у мастер в wals - 2 файла
	mockFileStorage.On("ReadDir", "wal").Return(files[:1], nil).Once()
	mockFileStorage.On("ReadDir", "wal").Return(files, nil)
	mockFileStorage.On("OpenFile", "wal/wal.1.log").Return(&fs.MockFile{}, nil)

	done := make(chan struct{})
	requestCount := 0
	serverHandler := func(conn net.Conn) {
		query, err := bufio.NewReader(conn).ReadString('\n')
		require.NoError(t, err)

		if requestCount == 0 {
			assert.Equal(t, string(compute.ReplicationCommandId)+"\n", query)

			_, err = conn.Write([]byte(strings.Join(masterWals, ",") + "\n"))
			require.NoError(t, err)
		} else {
			assert.Equal(t, string(compute.ReplicationCommandId)+" wal.1.log\n", query)

			_, err = conn.Write([]byte("SET a 0,SET b 1\n"))
			require.NoError(t, err)
		}

		requestCount++

		if requestCount == 2 {
			close(done)
		}
	}
	mockServer := newMockTCPServer(t, serverHandler)
	defer mockServer.Close()

	cfg := config.ReplicationConfig{
		ReplicaType:       ReplicaTypeSlave,
		MasterAddress:     mockServer.address,
		SyncInterval:      1 * time.Second,
		MaxReplicasNumber: 1,
	}
	r := NewReplication(cm, c, &cfg, logger)

	err := r.Start(ctx)
	assert.NoError(t, err)

	// Ждём сигнал с таймаутом, чтобы тест не завис
	select {
	case <-done:
		// Успешно получили сигнал
		_ = r.Stop()
	case <-time.After(15 * time.Second):
		t.Fatal("timeout waiting for replication request")
	}
}

func TestReplication_ReplicationFile_NotFound(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	mockFileStorage := new(fs.MockFileStorage)

	// Мастер возвращает список файлов
	masterWals := []string{"wal.0.log", "wal.1.log"}

	mockFileStorage.On("ReadDir", "wal").Return([]os.DirEntry{fs.NewMockDirEntry("wal.3.log", 1000)}, nil)

	c := compute.NewCompute(logger)
	cm := wal.NewChunkManager(mockFileStorage, "wal", 1000, logger)

	done := make(chan struct{})
	serverHandler := func(conn net.Conn) {
		query, err := bufio.NewReader(conn).ReadString('\n')
		require.NoError(t, err)

		assert.Equal(t, string(compute.ReplicationCommandId)+"\n", query)

		_, err = conn.Write([]byte(strings.Join(masterWals, ",")))
		require.NoError(t, err)

		close(done)
	}
	mockServer := newMockTCPServer(t, serverHandler)
	defer mockServer.Close()

	cfg := config.ReplicationConfig{
		ReplicaType:       ReplicaTypeSlave,
		MasterAddress:     mockServer.address,
		SyncInterval:      1 * time.Millisecond,
		MaxReplicasNumber: 1,
	}
	r := NewReplication(cm, c, &cfg, logger)

	err := r.Start(ctx)
	assert.NoError(t, err)

	// Ждём сигнал с таймаутом, чтобы тест не завис
	select {
	case <-done:
		// Успешно получили сигнал
		_ = r.Stop()
	case <-time.After(15 * time.Second):
		t.Fatal("timeout waiting for replication request")
	}
}
