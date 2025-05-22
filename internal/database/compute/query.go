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

const (
	QueryGetID    = "GET"
	QuerySetID    = "SET"
	QueryDeleteID = "DEL"
)

type Query struct {
	id   string
	args []string
}

func NewQuery(id string, args []string) Query {
	return Query{
		id:   id,
		args: args,
	}
}

func (q *Query) Validate() error {
	if q.id == QueryGetID && len(q.args) != 1 {
		return fmt.Errorf("%w: expected=%d, got=%d", ErrQueryArgsCount, 1, len(q.args))
	}

	if q.id == QuerySetID && len(q.args) != 2 {
		return fmt.Errorf("%w: expected=%d, got=%d", ErrQueryArgsCount, 2, len(q.args))
	}

	if q.id == QueryDeleteID && len(q.args) != 1 {
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

func (q *Query) Id() string {
	return q.id
}

func (q *Query) Args() []string {
	return q.args
}
