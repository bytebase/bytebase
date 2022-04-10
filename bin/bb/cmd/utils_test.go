package cmd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseDns(t *testing.T) {
	for _, test := range []struct {
		dsn     string
		want    *dataSource
		wantErr error
	}{
		{"", nil, fmt.Errorf("invalid dsn: ")},
		{"mysql://user:pass@localhost:3306/db?param1=value1&param2=value2", &dataSource{driver: "mysql", username: "user", password: "pass", host: "localhost", port: "3306", database: "db", params: map[string]string{"param1": "value1", "param2": "value2"}}, nil},
		{"postgres://user@localhost/db", &dataSource{driver: "postgres", username: "user", host: "localhost", database: "db", params: map[string]string{}}, nil},
	} {
		ds, err := parseDSN(test.dsn)
		require.Equal(t, test.want, ds)
		require.Equal(t, test.wantErr, err)
	}
}
