package compute

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	ErrEmptyQuery      = errors.New("query is empty")
	ErrInvalidQueryArg = errors.New("query contains invalid argument")
	ErrQueryArgsCount  = errors.New("query contains invalid arguments count")
)

var argRegex = regexp.MustCompile(`^[a-zA-Z0-9*/_]+$`)

type QueryType string

const (
	QueryTypeGet    QueryType = "GET"
	QueryTypeSet    QueryType = "SET"
	QueryTypeDelete QueryType = "DEL"
)

type Query struct {
	id   QueryType
	args []string
}

func NewQuery(id QueryType, args []string) Query {
	return Query{
		id:   id,
		args: args,
	}
}

func (q *Query) Validate() error {
	if q.id == QueryTypeGet && len(q.args) != 1 {
		return fmt.Errorf("%w: expected=%d, got=%d", ErrQueryArgsCount, 1, len(q.args))
	}

	if q.id == QueryTypeSet && len(q.args) != 2 {
		return fmt.Errorf("%w: expected=%d, got=%d", ErrQueryArgsCount, 2, len(q.args))
	}

	if q.id == QueryTypeDelete && len(q.args) != 1 {
		return fmt.Errorf("%w: expected=%d, got=%d", ErrQueryArgsCount, 1, len(q.args))
	}

	for _, arg := range q.args {
		if !argRegex.MatchString(arg) {
			return ErrInvalidQueryArg
		}
	}

	return nil
}

func (q *Query) String() string {
	return fmt.Sprintf("id=%s, args=%s", q.id, strings.Join(q.args, " "))
}

func (q *Query) Id() QueryType {
	return q.id
}

func (q *Query) Key() string {
	return q.args[0]
}

func (q *Query) Value() string {
	return q.args[1]
}
