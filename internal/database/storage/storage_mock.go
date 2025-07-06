package storage

import (
	"context"
	"iter"

	"github.com/stretchr/testify/mock"
)

type mockEngine struct {
	mock.Mock
}

func (m *mockEngine) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}
func (m *mockEngine) Set(ctx context.Context, key, value string) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}
func (m *mockEngine) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

type mockWAL struct {
	mock.Mock
}

func (m *mockWAL) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
func (m *mockWAL) All() iter.Seq2[string, error] {
	args := m.Called()
	return args.Get(0).(iter.Seq2[string, error])
}
func (m *mockWAL) Push(data string) error {
	args := m.Called(data)
	return args.Error(0)
}
