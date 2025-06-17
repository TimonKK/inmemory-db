package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/TimonKK/inmemory-db/internal/database/compute"
	"github.com/TimonKK/inmemory-db/internal/database/storage/engine"
	"go.uber.org/zap"
)

var (
	ErrUnknownQuery = errors.New("unknown query type")
)

// TODO вынести интерфейс Query куда-то
type Compute interface {
	ParseQuery(string) (compute.Query, error)
}

type Storage interface {
	Set(context.Context, string, string) error
	Get(context.Context, string) (string, error)
	Delete(context.Context, string) error
}

type Database struct {
	compute Compute
	storage Storage
	logger  *zap.Logger
}

func NewDatabase(compute Compute, storage Storage, logger *zap.Logger) *Database {
	return &Database{
		compute: compute,
		storage: storage,
		logger:  logger,
	}
}

func (db *Database) ExecQuery(ctx context.Context, queryStr string) (result string, err error) {
	db.logger.Debug("ExecQuery start", zap.String("query", queryStr))
	defer db.logger.Debug("ExecQuery", zap.String("result", result))

	query, err := db.compute.ParseQuery(queryStr)
	if err != nil {
		return "", err
	}

	db.logger.Info("ExecQuery parsed", zap.String("query", query.String()))

	switch query.CommandId() {
	case compute.GetCommandId:
		return db.ExecGet(ctx, query)
	case compute.SetCommandId:
		return db.ExecSet(ctx, query)
	case compute.DeleteCommandId:
		return db.ExecDelete(ctx, query)
	default:
		return "", fmt.Errorf("%w: %s", ErrUnknownQuery, queryStr)
	}
}

func (db *Database) ExecGet(ctx context.Context, query compute.Query) (string, error) {
	value, err := db.storage.Get(ctx, query.Key())
	if errors.Is(err, engine.ErrKeyNotFound) {
		return "no data", nil
	}

	if value == "" {
		return "empty", nil
	}

	return fmt.Sprintf("result: %s", value), nil
}

func (db *Database) ExecSet(ctx context.Context, query compute.Query) (string, error) {
	err := db.storage.Set(ctx, query.Key(), query.Value())
	if err != nil {
		return "", err
	}

	return "ok", nil
}
func (db *Database) ExecDelete(ctx context.Context, query compute.Query) (string, error) {
	err := db.storage.Delete(ctx, query.Key())
	if err != nil {
		return "", err
	}

	return "ok", nil
}
