package utils

import (
	"errors"
	"reflect"
	"testing"
)

func TestPromiseGet(t *testing.T) {
	type testCase[T any] struct {
		name string
		p    Promise[T]
		want T
	}

	tests := []testCase[error]{
		{
			name: "valid Get and Set as nil",
			p: func() Promise[error] {
				p := NewPromise[error]()
				p.Set(nil)
				return p
			}(),
			want: nil,
		},
		{
			name: "valid Get and Set as error",
			p: func() Promise[error] {
				p := NewPromise[error]()
				p.Set(errors.New("error"))
				return p
			}(),
			want: errors.New("error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.Get(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
		})
	}
}
