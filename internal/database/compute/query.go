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

type Query struct {
	id   CommandId
	args []string
}

func NewQuery(id CommandId, args []string) Query {
	return Query{
		id:   id,
		args: args,
	}
}

func NewQueryFromString(s string) Query {
	data := strings.Split(s, ";")
	key, args := data[0], strings.Split(data[1], ",")

	return Query{
		id:   CommandId(key),
		args: args,
	}
}

func (q *Query) Validate() error {
	if q.id == GetCommandId && len(q.args) != GetCommandArgsCount {
		return fmt.Errorf("%w: expected=%d, got=%d", ErrQueryArgsCount, 1, len(q.args))
	}

	if q.id == SetCommandId && len(q.args) != SetCommandArgsCount {
		return fmt.Errorf("%w: expected=%d, got=%d", ErrQueryArgsCount, 2, len(q.args))
	}

	if q.id == DeleteCommandId && len(q.args) != DeleteCommandArgsCount {
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
	return fmt.Sprintf("%s;%s", q.id, strings.Join(q.args, ","))
}

func (q *Query) CommandId() CommandId {
	return q.id
}

func (q *Query) Key() string {
	return q.args[0]
}

func (q *Query) Value() string {
	return q.args[1]
}

func (q *Query) Args() []string {
	return q.args
}
