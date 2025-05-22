package compute

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"reflect"
	"testing"
)

func TestComputeParseQuery(t *testing.T) {
	comp := NewCompute(zap.NewNop())

	tests := []struct {
		name    string
		raw     string
		want    Query
		wantErr error
	}{
		{
			name:    "invalid query",
			raw:     "GETT",
			wantErr: ErrUnknownQuery,
		},

		{
			name:    "empty query",
			raw:     "",
			wantErr: ErrEmptyQuery,
		},

		// GET
		{
			name:    "invalid GET without ars",
			raw:     "GET",
			wantErr: ErrQueryArgsCount,
		},

		{
			name:    "too many args for GET",
			raw:     "GET aaaa bbbb",
			wantErr: ErrQueryArgsCount,
		},

		{
			name: "valid GET abc",
			raw:  "GET abc",
			want: Query{id: QueryGetID, args: []string{"abc"}},
		},

		// SET
		{
			name:    "too less args for SET",
			raw:     "SET bbb",
			wantErr: ErrQueryArgsCount,
		},
		{
			name:    "too many args for SET",
			raw:     "SET bbb cccc dddd",
			wantErr: ErrQueryArgsCount,
		},
		{
			name: "valid SET",
			raw:  "SET bbb 123",
			want: Query{id: QuerySetID, args: []string{"bbb", "123"}},
		},

		// DELETE
		{
			name:    "too many args for DEL",
			raw:     "DEL bbb cccc dddd",
			wantErr: ErrQueryArgsCount,
		},
		{
			name: "valid DEL",
			raw:  "DEL ccc",
			want: Query{id: QueryDeleteID, args: []string{"ccc"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := comp.ParseQuery(tt.raw)
			if tt.wantErr != nil {
				require.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				assert.True(t, reflect.DeepEqual(tt.want, query))
			}
		})
	}
}
