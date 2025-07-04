package network

import (
	"bufio"
	"context"
	"fmt"
	"github.com/TimonKK/inmemory-db/internal/config"
	"github.com/TimonKK/inmemory-db/internal/utils"
	"go.uber.org/zap"
	"io"
	"net"
	"time"
)

type RequestHandler = func(context.Context, string) (string, error)

type TCPServer struct {
	// TODO указатель?!
	listener  net.Listener
	semaphore *utils.Semaphore

	config config.NetworkConfig
	logger *zap.Logger
}

func NewTCPServer(config config.NetworkConfig, logger *zap.Logger) (*TCPServer, error) {
	listener, err := net.Listen("tcp", config.Address)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to listen %s", err, config.Address)
	}

	server := &TCPServer{
		listener: listener,

		config: config,
		logger: logger,
	}

	if config.MaxConnections != 0 {
		server.semaphore = utils.NewSemaphore(config.MaxConnections)
	}

	return server, nil
}

func (s *TCPServer) Start() error {
	return nil
}

func (s *TCPServer) Shutdown() error {
	err := s.listener.Close()
	if err != nil {
		return err
	}

	return nil
}

func (s *TCPServer) HandleConnect(ctx context.Context, handler RequestHandler) {
	for {
		if ctx.Err() != nil {
			return
		}

		conn, err := s.listener.Accept()
		if err != nil {
			s.logger.Error("failed to accept connection", zap.Error(err))
			break
		}

		// TODO добавить методв TryAcquire, чтобы если нельзя - ответить клиенту ошибкой что нельзя
		s.tryAcquire()

		s.logger.Info("handleConnect: handling new connection", zap.String("remote", conn.RemoteAddr().String()))

		go func() {
			defer s.Release()

			err := s.handleConnect(ctx, conn, handler)
			if err != nil {
				// TODO добавить контексту, а что за коннект: ip, какие данные может успели прочитать
				s.logger.Error("failed to handle connect", zap.Error(err))
				return
			}
		}()
	}
}

func (s *TCPServer) handleConnect(ctx context.Context, conn net.Conn, handler RequestHandler) error {
	defer func() {
		if v := recover(); v != nil {
			s.logger.Error("handleConnect: captured panic", zap.Any("panic", v))
		}

		if err := conn.Close(); err != nil {
			s.logger.Warn("handleConnect: failed to close connection", zap.Error(err))
		}
	}()

	for {
		if s.config.IdleTimeout != 0 {
			if err := conn.SetReadDeadline(time.Now().Add(s.config.IdleTimeout)); err != nil {
				s.logger.Error("failed to set read deadline", zap.Duration("IdleTimeout", s.config.IdleTimeout), zap.Error(err))
				return err
			}
		}

		query, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			if err != io.EOF {
				s.logger.Error("handleConnect: failed to read data", zap.Error(err))
			} else if len(query) >= int(s.config.MaxMessageSize) {
				s.logger.Error("handleConnect: too much data", zap.Int("data", len(query)))
			}
			return err
		}

		s.logger.Info("handleConnect: request", zap.String("request", query))
		res, err := handler(ctx, query)
		if err != nil {
			s.logger.Error("handleConnect: failed to handle request", zap.Error(err))
			return err
		}
		s.logger.Info("handleConnect: response", zap.String("response", res))

		if _, err := conn.Write([]byte(res + "\n")); err != nil {
			s.logger.Error(
				"handleConnect: failed to write data",
				zap.String("address", conn.RemoteAddr().String()),
				zap.String("response", string(res)),
				zap.Error(err),
			)

			return err
		}
	}
}

func (s *TCPServer) tryAcquire() {
	if s.semaphore != nil {
		s.semaphore.Acquire()
	}
}

func (s *TCPServer) Release() {
	if s.semaphore != nil {
		s.semaphore.Release()
	}
}
