package database

import (
	"context"
	"errors"
	"testing"

	"github.com/TimonKK/inmemory-db/internal/database/compute"

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

func (m *MockStorage) Start(_ context.Context) error {
	return nil
}

func (m *MockStorage) Set(_ context.Context, query compute.Query) error {
	args := m.Called(query)
	return args.Error(0)
}

func (m *MockStorage) Get(_ context.Context, query compute.Query) (string, error) {
	args := m.Called(query)
	return args.String(0), args.Error(1)
}

func (m *MockStorage) Delete(_ context.Context, query compute.Query) error {
	args := m.Called(query)
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
					Return(compute.NewQuery(compute.GetCommandId, []string{"aaa"}), nil)
			},
			mockStorage: func(m *MockStorage) {
				m.On("Get", compute.NewQuery(compute.GetCommandId, []string{"aaa"})).Return("aaa", nil)
			},
		},
		{
			name:  "successful SET",
			query: "SET bbb 123",
			mockParse: func(m *MockCompute) {
				m.On("ParseQuery", "SET bbb 123").
					Return(compute.NewQuery(compute.SetCommandId, []string{"bbb", "123"}), nil)
			},
			mockStorage: func(m *MockStorage) {
				m.On("Set", compute.NewQuery(compute.SetCommandId, []string{"bbb", "123"})).Return(nil)
			},
		},
		{
			name:  "successful DELETE",
			query: "DELETE ccc",
			mockParse: func(m *MockCompute) {
				m.On("ParseQuery", "DELETE ccc").
					Return(compute.NewQuery(compute.DeleteCommandId, []string{"ccc"}), nil)
			},
			mockStorage: func(m *MockStorage) {
				m.On("Delete", compute.NewQuery(compute.DeleteCommandId, []string{"ccc"})).Return(nil)
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
			_, err := db.ExecQuery(context.TODO(), tt.query)

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
