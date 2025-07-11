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

var argRegex = regexp.MustCompile(`^[a-zA-Z0-9*./_]+$`)

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

func NewQueryFromString(query string) (Query, error) {
	tokens := strings.Fields(query)
	if len(tokens) == 0 {
		return Query{}, fmt.Errorf("%w: %s", ErrEmptyQuery, query)
	}

	commandId, args := CommandId(tokens[0]), tokens[1:]

	return NewQuery(commandId, args), nil
}

func (q *Query) Validate() error {
	if q.id == GetCommandId && len(q.args) != GetCommandArgsCount {
		return fmt.Errorf("%w: expected=%d, got=%d", ErrQueryArgsCount, GetCommandArgsCount, len(q.args))
	}

	if q.id == SetCommandId && len(q.args) != SetCommandArgsCount {
		return fmt.Errorf("%w: expected=%d, got=%d", ErrQueryArgsCount, SetCommandArgsCount, len(q.args))
	}

	if q.id == DeleteCommandId && len(q.args) != DeleteCommandArgsCount {
		return fmt.Errorf("%w: expected=%d, got=%d", ErrQueryArgsCount, DeleteCommandArgsCount, len(q.args))
	}

	if q.id == ReplicationCommandId && len(q.args) > ReplicationCommandArgsCount {
		return fmt.Errorf("%w: expected zero or one, got=%d", ErrQueryArgsCount, len(q.args))
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
	if len(q.args) == 0 {
		return ""
	}

	return q.args[0]
}

func (q *Query) Value() string {
	return q.args[1]
}

func (q *Query) Args() []string {
	return q.args
}
