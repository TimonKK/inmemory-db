package config

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				Engine: EngineConfig{Type: "in_memory"},
				Network: NetworkConfig{
					Address:        "127.0.0.1:8080",
					MaxConnections: 100,
					MaxMessageSize: 1024,
					IdleTimeout:    5 * time.Minute,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Output: "stdout",
				},
				Wal: WALConfig{
					FlushingBatchSize:    100,
					FlushingBatchTimeout: 10 * time.Millisecond,
					MaxSegmentSize:       10 << 20,
					DataDirectory:        "wal",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid engine type",
			cfg: Config{
				Engine: EngineConfig{Type: "invalid"},
				Network: NetworkConfig{
					Address: "127.0.0.1:8080",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid address",
			cfg: Config{
				Engine: EngineConfig{Type: "in_memory"},
				Network: NetworkConfig{
					Address: "",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid max connections",
			cfg: Config{
				Engine: EngineConfig{Type: "in_memory"},
				Network: NetworkConfig{
					Address:        "127.0.0.1:8080",
					MaxConnections: -1,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			cfg: Config{
				Engine: EngineConfig{Type: "in_memory"},
				Network: NetworkConfig{
					Address: "127.0.0.1:8080",
				},
				Logging: LoggingConfig{
					Level: "invalid",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid flushing batch size",
			cfg: Config{
				Engine: EngineConfig{Type: "in_memory"},
				Network: NetworkConfig{
					Address:        "127.0.0.1:8080",
					MaxConnections: 100,
					MaxMessageSize: 1024,
					IdleTimeout:    5 * time.Minute,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Output: "stdout",
				},
				Wal: WALConfig{
					FlushingBatchSize: 0,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid flushing timeout",
			cfg: Config{
				Engine: EngineConfig{Type: "in_memory"},
				Network: NetworkConfig{
					Address:        "127.0.0.1:8080",
					MaxConnections: 100,
					MaxMessageSize: 1024,
					IdleTimeout:    5 * time.Minute,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Output: "stdout",
				},
				Wal: WALConfig{
					FlushingBatchSize:    100,
					FlushingBatchTimeout: -1,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Создаем временный файл конфигурации
	validConfig := `
engine:
  type: "in_memory"
network:
  address: "127.0.0.1:8081"
  max_connections: 99
  max_message_size: "4KB"
  idle_timeout: "4m"
logging:
  level: "debug"
  output: "stdout"
wal:
  flushing_batch_size: 15
  flushing_batch_timeout: "1s"
  max_segment_size: "1MB"
  data_directory: "wal111"
`

	tmpFile, err := os.CreateTemp("", "config_test_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			t.Fatal(err)
		}
	}(tmpFile.Name())

	if _, err := tmpFile.WriteString(validConfig); err != nil {
		t.Fatal(err)
	}

	err = tmpFile.Close()
	if err != nil {
		t.Fatal(err)
	}

	expectedConfig := Config{
		Engine: EngineConfig{Type: "in_memory"},
		Network: NetworkConfig{
			Address:        "127.0.0.1:8081",
			MaxConnections: 99,
			MaxMessageSize: 4 * 1024,
			IdleTimeout:    4 * time.Minute,
		},
		Logging: LoggingConfig{
			Level:  "debug",
			Output: "stdout",
		},
		Wal: WALConfig{
			FlushingBatchSize:    15,
			FlushingBatchTimeout: 1 * time.Second,
			MaxSegmentSize:       1 << 20,
			DataDirectory:        "wal111",
		},
	}

	t.Run("valid config file", func(t *testing.T) {
		cfg, err := LoadConfig(tmpFile.Name())
		assert.NoError(t, err)
		assert.Equal(t, expectedConfig, *cfg)
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := LoadConfig("nonexistent.yaml")
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}
	})

	// Тест с невалидным YAML
	invalidConfig := `
engine:
  type: "invalid"
network:
  address: "invalid_address"
`

	tmpInvalidFile, err := os.CreateTemp("", "config_invalid_test_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			t.Fatal(err)
		}
	}(tmpInvalidFile.Name())

	if _, err := tmpInvalidFile.WriteString(invalidConfig); err != nil {
		t.Fatal(err)
	}
	err = tmpInvalidFile.Close()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("invalid config file", func(t *testing.T) {
		_, err := LoadConfig(tmpInvalidFile.Name())
		if err == nil {
			t.Error("Expected error for invalid config, got nil")
		}
	})
}
