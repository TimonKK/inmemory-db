package database

import (
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
	Set(string, string) error
	Get(string) (string, error)
	Delete(string) error
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

func (db *Database) ExecQuery(queryStr string) (result string, err error) {
	db.logger.Debug("ExecQuery start", zap.String("query", queryStr))
	defer db.logger.Debug("ExecQuery", zap.String("result", result))

	query, err := db.compute.ParseQuery(queryStr)
	if err != nil {
		return "", err
	}

	db.logger.Info("ExecQuery parsed", zap.String("query", query.String()))

	switch query.Id() {
	case compute.QueryGetID:
		return db.ExecGet(query)
	case compute.QuerySetID:
		return db.ExecSet(query)
	case compute.QueryDeleteID:
		return db.HandeDelete(query)
	default:
		return "", fmt.Errorf("%w: %s", ErrUnknownQuery, queryStr)
	}
}

func (db *Database) ExecGet(query compute.Query) (string, error) {
	key := query.Args()[0]
	value, err := db.storage.Get(key)
	if errors.Is(err, engine.ErrKeyNotFound) {
		return "no data", nil
	}

	if value == "" {
		return "empty", nil
	}

	return fmt.Sprintf("result: %s", value), nil
}

func (db *Database) ExecSet(query compute.Query) (string, error) {
	args := query.Args()
	key, value := args[0], args[1]
	err := db.storage.Set(key, value)
	if err != nil {
		return "", err
	}

	return "ok", nil
}
func (db *Database) HandeDelete(query compute.Query) (string, error) {
	key := query.Args()[0]
	err := db.storage.Delete(key)
	if err != nil {
		return "", err
	}

	return "ok", nil
}
