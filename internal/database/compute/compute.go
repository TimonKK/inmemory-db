package compute

import (
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

var (
	ErrUnknownQuery = errors.New("unknown query")
)

type Compute struct {
	logger *zap.Logger
}

func NewCompute(logger *zap.Logger) *Compute {
	return &Compute{
		logger: logger,
	}
}

func (c *Compute) ParseQuery(queryStr string) (Query, error) {
	query, err := c.parseQuery(queryStr)
	if err != nil {
		return Query{}, err
	}

	err = query.Validate()
	if err != nil {
		return Query{}, err
	}

	return query, nil
}

func (c *Compute) parseQuery(query string) (Query, error) {
	tokens := strings.Fields(query)
	if len(tokens) == 0 {
		return Query{}, fmt.Errorf("%w: %s", ErrEmptyQuery, query)
	}

	queryId, args := QueryType(tokens[0]), tokens[1:]

	switch queryId {
	case QueryTypeGet, QueryTypeSet, QueryTypeDelete:
		return NewQuery(queryId, args), nil
	default:
		return Query{}, ErrUnknownQuery
	}
}
