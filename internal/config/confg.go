package config

import (
	"errors"
	"fmt"
	"github.com/TimonKK/inmemory-db/internal/utils"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	ErrEngineType           = errors.New("invalid engine type")
	ErrEmptyAddressConfig   = errors.New("network address cannot be empty")
	ErrInvalidAddressFormat = errors.New("network address must valid host:port")
	ErrInvalidParamRange    = errors.New("must be in range")
	ErrEmptyFilePath        = errors.New("file path cannot be empty")
)

// EngineConfig - настройки движка
type EngineConfig struct {
	Type string `yaml:"type"`
}

type SizeInBytes int64

type ClientNetworkConfig struct {
	Address     string
	IdleTimeout time.Duration
}

// NetworkConfig - сетевые настройки сервера
// TODO Встраиваие ClientNetworkConfig в NetworkConfig ломает парсинг yaml
type NetworkConfig struct {
	Address        string        `yaml:"address" default:"127.0.0.1:3223"`
	MaxConnections int           `yaml:"max_connections" default:"100"`
	MaxMessageSize SizeInBytes   `yaml:"max_message_size" default:"4KB"` // "4KB", "1MB" и т.д.
	IdleTimeout    time.Duration `yaml:"idle_timeout" default:"5m"`      // "5m", "10s" и т.д.
}

// LoggingConfig - настройки логирования
type LoggingConfig struct {
	Level  string `yaml:"level" default:"info"`        // "debug", "info", "warn", "error"
	Output string `yaml:"output" default:"output.log"` // путь к файлу или "stdout"
}

type WALConfig struct {
	FlushingBatchSize    int           `yaml:"flushing_batch_size" default:"100"`
	FlushingBatchTimeout time.Duration `yaml:"flushing_batch_timeout" default:"10ms"`
	MaxSegmentSize       SizeInBytes   `yaml:"max_segment_size" default:"10MB"`
	DataDirectory        string        `yaml:"data_directory" default:"wal"`
}

// Config - основная структура конфигурации
type Config struct {
	Engine  EngineConfig  `yaml:"engine"`
	Network NetworkConfig `yaml:"network"`
	Wal     WALConfig     `yaml:"wal"`
	Logging LoggingConfig `yaml:"logging"`
}

// UnmarshalYAML SizeInBytes - кастомное правило десериализации для MaxMessageSize
func (s *SizeInBytes) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw interface{}
	if err := unmarshal(&raw); err != nil {
		return err
	}

	switch v := raw.(type) {
	case int:
		*s = SizeInBytes(v)
		return nil
	case string:
		bytes, err := utils.ParseSizeString(v)
		if err != nil {
			return fmt.Errorf("invalid size format: %w", err)
		}
		*s = SizeInBytes(bytes)
		return nil
	default:
		return fmt.Errorf("size must be either string or integer, got %T", v)
	}
}

// LoadConfig загружает конфиг из файла и валидирует его
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		configPath = "config.yaml"
	}

	data, err := os.ReadFile(configPath)

	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// Validate проверяет корректность всех параметров конфига
func (c *Config) Validate() error {
	if err := c.validateEngine(); err != nil {
		return err
	}

	if err := c.validateNetwork(); err != nil {
		return err
	}

	if err := c.validateLogging(); err != nil {
		return err
	}

	if err := c.validateWAL(); err != nil {
		return err
	}

	return nil
}

func (c *Config) validateWAL() error {
	if c.Wal.FlushingBatchSize <= 0 || c.Wal.FlushingBatchSize > 1<<30 {
		return fmt.Errorf("flushing_batch_size %w [1, 1^30], but got %d", ErrInvalidParamRange, c.Wal.FlushingBatchSize)
	}

	if c.Wal.FlushingBatchTimeout < 0 || c.Wal.FlushingBatchTimeout > 5*time.Minute {
		return fmt.Errorf("flushing_batch_timeout %w [1s, 5m], but got %d", ErrInvalidParamRange, c.Wal.FlushingBatchTimeout)
	}

	if c.Wal.MaxSegmentSize <= 0 || c.Wal.MaxSegmentSize > 1<<30 {
		return fmt.Errorf("max_segment_size %w [1, 1^30] byte, but got %d", ErrInvalidParamRange, c.Wal.MaxSegmentSize)
	}

	if c.Wal.DataDirectory == "" {
		return fmt.Errorf("config empty wal path %w", ErrEmptyFilePath)
	}

	return nil
}

func (c *Config) validateEngine() error {
	validTypes := map[string]bool{
		"in_memory": true,
	}

	if !validTypes[c.Engine.Type] {
		return ErrEngineType
	}
	return nil
}

func (c *Config) validateNetwork() error {
	address := c.Network.Address
	if address == "" {
		return ErrEmptyAddressConfig
	}

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return ErrInvalidAddressFormat
	}

	// Проверка IP или DNS имени
	ip := net.ParseIP(host)
	_, hostErr := net.LookupHost(host)
	p, portErr := strconv.Atoi(port)
	if ip == nil || hostErr != nil || portErr != nil || p < 1 || p > 65535 {
		return ErrInvalidAddressFormat
	}

	if c.Network.MaxConnections <= 0 || c.Network.MaxConnections > 100 {
		return fmt.Errorf("max_connections %w [1, 100]", ErrInvalidParamRange)
	}

	if c.Network.MaxMessageSize <= 0 || c.Network.MaxMessageSize > 1<<30 {
		return fmt.Errorf("max_message_size %w [1, 1^30] byte", ErrInvalidParamRange)
	}

	if c.Network.IdleTimeout < 0 || c.Network.IdleTimeout > 5*time.Minute {
		return fmt.Errorf("idle_timeout %w [1s, 5m]", ErrInvalidParamRange)
	}

	return nil
}

func (c *Config) validateLogging() error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}

	if !validLevels[strings.ToLower(c.Logging.Level)] {
		return fmt.Errorf("config invalid log level: %s", c.Logging.Level)
	}

	if c.Logging.Output == "" {
		return fmt.Errorf("config empty output path %w", ErrEmptyFilePath)
	}

	return nil
}
