package engine

import (
	"context"
	_ "github.com/TimonKK/inmemory-db/internal/database/storage"
	"testing"

	"github.com/stretchr/testify/assert"
)

var ctx = context.TODO()

func TestMemoryEngine(t *testing.T) {
	// Общие настройки
	tests := []struct {
		name          string
		prepare       func(e *MemoryEngine) // Подготовка данных
		key           string                // Ключ для теста
		expectedValue string                // Ожидаемое значение
		expectedError error                 // Ожидаемая ошибка
	}{
		{
			name:          "Get non-existent key",
			prepare:       func(e *MemoryEngine) {},
			key:           "missing",
			expectedValue: "",
			expectedError: ErrKeyNotFound,
		},
		{
			name: "Get existing key",
			prepare: func(e *MemoryEngine) {
				_ = e.Set(ctx, "test", "value") // Используем Set для подготовки
			},
			key:           "test",
			expectedValue: "value",
			expectedError: nil,
		},
		{
			name: "Overwrite existing key",
			prepare: func(e *MemoryEngine) {
				_ = e.Set(ctx, "key", "old")
				_ = e.Set(ctx, "key", "new")
			},
			key:           "key",
			expectedValue: "new",
			expectedError: nil,
		},
		{
			name:          "Empty key",
			prepare:       func(e *MemoryEngine) {},
			key:           "",
			expectedValue: "",
			expectedError: ErrKeyNotFound,
		},
	}

	t.Run("Get operations", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				engine := NewMemoryEngine()
				tt.prepare(engine)

				value, err := engine.Get(ctx, tt.key)

				assert.Equal(t, tt.expectedValue, value)
				if tt.expectedError != nil {
					assert.ErrorIs(t, err, tt.expectedError)
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}

func TestMemoryEngine_Set(t *testing.T) {
	engine := NewMemoryEngine()

	t.Run("Set new key", func(t *testing.T) {
		err := engine.Set(ctx, "new", "value")
		assert.NoError(t, err)

		val, err := engine.Get(ctx, "new")
		assert.Equal(t, "value", val)
		assert.NoError(t, err)
	})

	t.Run("Set empty key", func(t *testing.T) {
		err := engine.Set(ctx, "", "value")
		assert.NoError(t, err)
	})
}

func TestMemoryEngine_Delete(t *testing.T) {
	engine := NewMemoryEngine()
	_ = engine.Set(ctx, "to_delete", "value")

	t.Run("Delete existing key", func(t *testing.T) {
		err := engine.Delete(ctx, "to_delete")
		assert.NoError(t, err)

		_, err = engine.Get(ctx, "to_delete")
		assert.ErrorIs(t, err, ErrKeyNotFound)
	})

	t.Run("Delete non-existent key", func(t *testing.T) {
		err := engine.Delete(ctx, "missing")
		assert.NoError(t, err) // Обычно удаление несуществующего ключа не считается ошибкой
	})
}
