package cmd

import (
	"testing"

	"github.com/xo/dburl"
)

func TestGetDatabase(t *testing.T) {
	for _, tt := range []struct {
		u   string
		exp string
	}{
		{"mysql://root@localhost:3306/bytebase_test_todo", "bytebase_test_todo"},
		{"mysql://root@localhost:13308/", ""},
		{"mysql://root@localhost:13308/bytebase_test_todo?ssl-key=a", "bytebase_test_todo"},
	} {
		u, err := dburl.Parse(tt.u)
		if err != nil {
			t.Error(err)
		}
		if getDatabase(u) != tt.exp {
			t.Error("expected", tt.exp, "got", getDatabase(u))
		}
	}
}
