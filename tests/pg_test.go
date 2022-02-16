package tests

import (
	"database/sql"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/lib/pq"
)

func TestEmbeddedPostgresSelectOne(t *testing.T) {
	database := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().Port(1301).Version(embeddedpostgres.V14))
	if err := database.Start(); err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := database.Stop(); err != nil {
			t.Fatal(err)
		}
	}()

	db, err := sql.Open("postgres", "host=localhost port=1301 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	var result int
	row := db.QueryRow("SELECT 1")
	if err := row.Scan(&result); err != nil {
		t.Fatal(err)
	}
	if result != 1 {
		t.Errorf("want result 1, got %v", result)
	}
}
