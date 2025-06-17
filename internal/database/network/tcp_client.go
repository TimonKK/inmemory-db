package network

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/TimonKK/inmemory-db/internal/config"
	"go.uber.org/zap"
	"io"
	"net"
)

type TCPClient struct {
	config *config.ClientNetworkConfig
	logger *zap.Logger
	conn   net.Conn
}

func NewTCPClient(config *config.ClientNetworkConfig, logger *zap.Logger) (*TCPClient, error) {
	client := &TCPClient{
		config: config,
		logger: logger,
	}

	err := client.сonnect()
	if err != nil {
		return nil, err
	}

	return client, nil
}

// Connect - подключает клиента к базе. Потоконебезопастнный
func (c *TCPClient) сonnect() error {
	conn, err := net.Dial("tcp", c.config.Address)
	if err != nil {
		return err
	}

	c.conn = conn

	c.logger.Info("Connected to server", zap.String("address", c.config.Address))

	return nil
}

func (c *TCPClient) Send(query string) (string, error) {
	c.logger.Info("Sending query", zap.String("query", query))

	// TODO подумать что делать если запись упала:
	// 1) реконнект от греха подальше
	// 2) подождать писечку и опробовать еще раз
	// 3) ничего не делать (выбрано)
	bytesWritten, err := c.conn.Write([]byte(query + "\n"))
	if err != nil {
		switch {
		case errors.Is(err, net.ErrClosed):
			c.logger.Error("Connection already closed")
			return "", err
		case errors.Is(err, io.ErrShortWrite):
			c.logger.Error("Partial write occurred")
			return "", err
		default:
			c.logger.Error("Network write error: %v", zap.Error(err))
			return "", fmt.Errorf("network error: %w", err)
		}
	}

	if bytesWritten != (len(query) + 1) {
		c.logger.Warn("Partial write: %d of %d bytes", zap.Int("bytesWritten", bytesWritten), zap.Int("query", len(query)))
		return "", fmt.Errorf("partial write")
	}

	response, err := bufio.NewReader(c.conn).ReadString('\n')
	if err != nil {
		return "", err
	}

	return response, nil
}

func (c *TCPClient) Close() error {
	if c.conn != nil {
		err := c.conn.Close()
		if err != nil {
			return err
		}
	}

	return nil
}
