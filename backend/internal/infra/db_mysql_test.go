package infra

import (
	"errors"
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"
)

func TestIsIndexExistsErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "MySQL ER_DUP_KEYNAME (1061) matches",
			err:  &mysqldriver.MySQLError{Number: 1061, Message: "Duplicate key name 'idx_da_status_next'"},
			want: true,
		},
		{
			name: "MySQL other error does not match",
			err:  &mysqldriver.MySQLError{Number: 1146, Message: "Table doesn't exist"},
			want: false,
		},
		{
			name: "wrapped dup-keyname still matches via errors.As",
			err:  errors.Join(errors.New("ctx"), &mysqldriver.MySQLError{Number: 1061}),
			want: true,
		},
		{
			name: "non-MySQL error does not match",
			err:  errors.New("some unrelated error 1061 in prose"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isIndexExistsErr(tt.err); got != tt.want {
				t.Errorf("isIndexExistsErr(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
