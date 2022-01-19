package mysql

import "testing"

func TestMysql8WindowFunction(t *testing.T) {
	parser := newParser()
	_, warns, err := parser.Parse("SELECT row_number() OVER ( ORDER BY id ), id FROM xxx;", "utf8mb4", "utf8mb4_general_ci")
	if err != nil {
		t.Errorf("Expect no error, but got %v", err)
	}
	if len(warns) != 0 {
		t.Errorf("Expect no warning, but got %+v", warns)
	}
}
