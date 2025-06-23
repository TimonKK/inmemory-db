package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseSizeString(t *testing.T) {
	tests := []struct {
		name          string
		str           string
		expectedValue int64
		expectedError error
	}{
		{
			name:          "empty",
			str:           "",
			expectedError: ErrEmptySizeStr,
		},

		{
			name:          "non numeric value",
			str:           "a1",
			expectedError: ErrNonNumericSize,
		},

		{
			name:          "valid size 5KB",
			str:           "5KB",
			expectedValue: 5 * 1024,
			expectedError: nil,
		},

		{
			name:          "valid size 15MB",
			str:           "15MB",
			expectedValue: 15 * 1024 * 1024,
			expectedError: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size, err := ParseSizeString(tt.str)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedValue, size)
			}
		})
	}
}
