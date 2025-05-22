package compute

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"strings"
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

	queryId, args := tokens[0], tokens[1:]

	switch queryId {
	case QueryGetID:
		return NewQuery(QueryGetID, args), nil
	case QuerySetID:
		return NewQuery(QuerySetID, args), nil
	case QueryDeleteID:
		return NewQuery(QueryDeleteID, args), nil
	default:
		return Query{}, ErrUnknownQuery
	}
}
