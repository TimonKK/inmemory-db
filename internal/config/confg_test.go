package config

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func NewDefaultConfig() Config {
	return Config{
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
	}
}

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
  address: "127.0.0.1:8080"
  max_connections: 100
  max_message_size: "4KB"
  idle_timeout: "5m"
logging:
  level: "info"
  output: "stdout"
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

	t.Run("valid config file", func(t *testing.T) {
		cfg, err := LoadConfig(tmpFile.Name())
		if err != nil {
			t.Errorf("LoadConfig() unexpected error = %v", err)
		}

		if cfg.Engine.Type != "in_memory" {
			t.Errorf("Expected engine type 'in_memory', got %q", cfg.Engine.Type)
		}

		if cfg.Network.MaxMessageSize != 4096 {
			t.Errorf("Expected max message size 4096, got %d", cfg.Network.MaxMessageSize)
		}
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

func TestNetworkConfig_Validate(t *testing.T) {
	tests := []struct {
		name          string
		cfg           Config
		expectedError error
	}{
		{
			name: "empty engine type",
			cfg: Config{
				Engine: EngineConfig{Type: ""},
			},
			expectedError: ErrEngineType,
		},
		{
			name: "invalid engine type",
			cfg: Config{
				Engine: EngineConfig{Type: ""},
			},
			expectedError: ErrEngineType,
		},

		{
			name: "empty address",
			cfg: Config{
				Engine: EngineConfig{Type: "in_memory"},
				Network: NetworkConfig{
					Address:        "",
					MaxConnections: 100,
					MaxMessageSize: 1024,
					IdleTimeout:    5 * time.Minute,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Output: "stdout",
				},
			},
			expectedError: ErrEmptyAddressConfig,
		},
		{
			name: "invalid address",
			cfg: Config{
				Engine: EngineConfig{Type: "in_memory"},
				Network: NetworkConfig{
					Address:        "123",
					MaxConnections: 100,
					MaxMessageSize: 1024,
					IdleTimeout:    5 * time.Minute,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Output: "stdout",
				},
			},
			expectedError: ErrInvalidAddressFormat,
		},
		{
			name: "invalid max connections",
			cfg: Config{
				Engine: EngineConfig{Type: "in_memory"},
				Network: NetworkConfig{
					Address:        "127.0.0.1:8080",
					MaxConnections: -1,
					MaxMessageSize: 1024,
					IdleTimeout:    5 * time.Minute,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Output: "stdout",
				},
			},
			expectedError: ErrInvalidParamRange,
		},
		{
			name: "invalid max message size",
			cfg: Config{
				Engine: EngineConfig{Type: "in_memory"},
				Network: NetworkConfig{
					Address:        "127.0.0.1:8080",
					MaxConnections: 100,
					MaxMessageSize: -1,
					IdleTimeout:    5 * time.Minute,
				},
				Logging: LoggingConfig{
					Level:  "info",
					Output: "stdout",
				},
			},
			expectedError: ErrInvalidParamRange,
		},
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
func TestNetworkConfig_Load(t *testing.T) {
	// Создаем временный файл конфигурации
	validConfig := `
engine:
  type: "in_memory"
network:
  address: "127.0.0.1:8080"
  max_connections: 100
  max_message_size: "4KB"
  idle_timeout: "5m"
logging:
  level: "info"
  output: "stdout"
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

	t.Run("valid config file", func(t *testing.T) {
		cfg, err := LoadConfig(tmpFile.Name())
		if err != nil {
			t.Errorf("LoadConfig() unexpected error = %v", err)
		}

		if cfg.Engine.Type != "in_memory" {
			t.Errorf("Expected engine type 'in_memory', got %q", cfg.Engine.Type)
		}

		if cfg.Network.MaxMessageSize != 4096 {
			t.Errorf("Expected max message size 4096, got %d", cfg.Network.MaxMessageSize)
		}
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
