package network

import (
	"bufio"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TimonKK/inmemory-db/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		defer server.wg.Done()
		conn, err := listener.Accept()
		if err != nil {
			// Check if the error is due to the listener being closed, which is expected during shutdown.
			if !strings.Contains(err.Error(), "use of closed network connection") {
				t.Logf("Mock server failed to accept connection: %v", err)
			}
			return
		}
		defer func() {
			_ = conn.Close()
		}()
		if server.handler != nil {
			server.handler(conn)
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

func TestNewTCPClient(t *testing.T) {
	serverHandler := func(conn net.Conn) {
		_ = conn.Close()
	}
	mockServer := newMockTCPServer(t, serverHandler)
	defer mockServer.Close()

	cfg := &config.ClientNetworkConfig{Address: "localhost:1234"}
	logger := zap.NewNop()
	client, err := NewTCPClient(cfg, logger)
	require.Error(t, err, " connect: connection refused")
	assert.Nil(t, client)
}

func TestTCPClient_Connect_Success(t *testing.T) {
	serverHandler := func(conn net.Conn) {
		_, err := conn.Write([]byte("ok"))
		require.NoError(t, err)

		for {
			select {
			case <-time.After(100 * time.Millisecond):
				return
			default:
				_, err := bufio.NewReader(conn).ReadString('\n')
				if err != nil {
					return
				}
				_, err = conn.Write([]byte("OK\n"))
				require.NoError(t, err)
			}
		}
	}
	mockServer := newMockTCPServer(t, serverHandler)
	defer mockServer.Close()

	cfg := &config.ClientNetworkConfig{Address: mockServer.address}
	logger := zap.NewNop()
	client, err := NewTCPClient(cfg, logger)
	require.NoError(t, err)

	defer func() {
		_ = client.Close()
	}()

	assert.NotNil(t, client.conn)

}

func TestTCPClient_Connect_Failure(t *testing.T) {
	cfg := &config.ClientNetworkConfig{Address: "127.0.0.1:1"} // Use a port that's unlikely to be in use
	logger := zap.NewNop()
	_, err := NewTCPClient(cfg, logger)
	require.Error(t, err, " connect: connection refused")
}

func TestTCPClient_Send_Success(t *testing.T) {
	request := "GET a"
	expectedResponse := "no data\n"
	serverHandler := func(conn net.Conn) {
		query, err := bufio.NewReader(conn).ReadString('\n')
		require.NoError(t, err)
		assert.Equal(t, request+"\n", query)
		_, err = conn.Write([]byte(expectedResponse))
		require.NoError(t, err)
	}
	mockServer := newMockTCPServer(t, serverHandler)
	defer mockServer.Close()

	cfg := &config.ClientNetworkConfig{Address: mockServer.address}
	logger := zap.NewNop()
	client, err := NewTCPClient(cfg, logger)
	require.NoError(t, err)

	defer func() {
		_ = client.Close()
	}()

	response, err := client.Send(request)
	require.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
}

func TestTCPClient_Send_NotConnected_AutoConnects(t *testing.T) {
	request := "SET a 1"
	expectedResponse := "ok\n"
	serverHandler := func(conn net.Conn) {
		query, err := bufio.NewReader(conn).ReadString('\n')
		require.NoError(t, err)
		assert.Equal(t, request+"\n", query)

		_, err = conn.Write([]byte(expectedResponse))
		require.NoError(t, err)
	}
	mockServer := newMockTCPServer(t, serverHandler)
	defer mockServer.Close()

	cfg := &config.ClientNetworkConfig{Address: mockServer.address}
	logger := zap.NewNop()
	client, err := NewTCPClient(cfg, logger)
	require.NoError(t, err)
	defer func() {
		_ = client.Close()
	}()

	response, err := client.Send(request)
	require.NoError(t, err)
	assert.Equal(t, expectedResponse, response)
}

func TestTCPClient_Send_ConnectionErrorOnWrite(t *testing.T) {
	serverHandler := func(conn net.Conn) {
		_ = conn.Close()
	}
	mockServer := newMockTCPServer(t, serverHandler)
	defer mockServer.Close()

	cfg := &config.ClientNetworkConfig{Address: mockServer.address}
	logger := zap.NewNop()
	client, err := NewTCPClient(cfg, logger)
	require.NoError(t, err)

	defer func() {
		_ = client.Close()
	}()

	time.Sleep(50 * time.Millisecond)

	_, err = client.Send("GET aa")
	require.Error(t, err, "Send should fail due to connection error on write")
}

func TestTCPClient_Send_ErrorOnRead(t *testing.T) {
	serverHandler := func(conn net.Conn) {
		_, err := bufio.NewReader(conn).ReadString('\n')
		require.NoError(t, err)
		_ = conn.Close()
	}
	mockServer := newMockTCPServer(t, serverHandler)
	defer mockServer.Close()

	cfg := &config.ClientNetworkConfig{Address: mockServer.address}
	logger := zap.NewNop()
	client, err := NewTCPClient(cfg, logger)
	require.NoError(t, err)

	defer func() {
		_ = client.Close()
	}()

	_, err = client.Send("GET aaa")
	assert.ErrorIs(t, err, io.EOF, "Expected EOF error")
}

func TestTCPClient_Close_Success(t *testing.T) {
	serverHandler := func(conn net.Conn) {
		_, err := bufio.NewReader(conn).ReadString('\n')
		require.NoError(t, err)
		_, err = conn.Write([]byte("no data\n"))
		require.NoError(t, err)
	}
	mockServer := newMockTCPServer(t, serverHandler)
	defer mockServer.Close()

	cfg := &config.ClientNetworkConfig{Address: mockServer.address}
	logger := zap.NewNop()
	client, err := NewTCPClient(cfg, logger)
	require.NoError(t, err)

	_, err = client.Send("GET aaaa")
	require.NoError(t, err)

	err = client.Close()
	require.NoError(t, err)
}

func TestTCPClient_Close_NotConnected(t *testing.T) {
	cfg := &config.ClientNetworkConfig{Address: "localhost:1234"}
	logger := zap.NewNop()
	_, err := NewTCPClient(cfg, logger)
	require.Error(t, err, " connect: connection refused")
}
