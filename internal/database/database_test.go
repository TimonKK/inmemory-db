package database

import (
	"errors"
	"github.com/TimonKK/inmemory-db/internal/database/compute"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockCompute struct {
	mock.Mock
}

func (m *MockCompute) ParseQuery(query string) (compute.Query, error) {
	args := m.Called(query)
	return args.Get(0).(compute.Query), args.Error(1)
}

type MockStorage struct {
	mock.Mock
}

func (m *MockStorage) Set(key, value string) error {
	args := m.Called(key, value)
	return args.Error(0)
}

func (m *MockStorage) Get(key string) (string, error) {
	args := m.Called(key)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) Delete(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

func TestDatabase_Execute(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		name          string
		query         string
		mockParse     func(*MockCompute)
		mockStorage   func(*MockStorage)
		expectedError error
	}{
		{
			name:  "successful GET",
			query: "GET aaa",
			mockParse: func(m *MockCompute) {
				m.On("ParseQuery", "GET aaa").
					Return(compute.NewQuery(compute.QueryGetID, []string{"aaa"}), nil)
			},
			mockStorage: func(m *MockStorage) {
				m.On("Get", "aaa").Return("aaa", nil)
			},
		},
		{
			name:  "successful SET",
			query: "SET bbb 123",
			mockParse: func(m *MockCompute) {
				m.On("ParseQuery", "SET bbb 123").
					Return(compute.NewQuery(compute.QuerySetID, []string{"bbb", "123"}), nil)
			},
			mockStorage: func(m *MockStorage) {
				m.On("Set", "bbb", "123").Return(nil)
			},
		},
		{
			name:  "successful DELETE",
			query: "DELETE ccc",
			mockParse: func(m *MockCompute) {
				m.On("ParseQuery", "DELETE ccc").
					Return(compute.NewQuery(compute.QueryDeleteID, []string{"ccc"}), nil)
			},
			mockStorage: func(m *MockStorage) {
				m.On("Delete", "ccc").Return(nil)
			},
		},
		{
			name:  "parse error",
			query: "ГЕТ",
			mockParse: func(m *MockCompute) {
				m.On("ParseQuery", "ГЕТ").
					Return(compute.Query{}, errors.New("parse error"))
			},
			mockStorage:   func(m *MockStorage) {},
			expectedError: errors.New("parse error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCompute := new(MockCompute)
			mockStorage := new(MockStorage)

			tt.mockParse(mockCompute)
			tt.mockStorage(mockStorage)

			db := NewDatabase(mockCompute, mockStorage, logger)
			_, err := db.ExecQuery(tt.query)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			mockCompute.AssertExpectations(t)
			mockStorage.AssertExpectations(t)
		})
	}
}
