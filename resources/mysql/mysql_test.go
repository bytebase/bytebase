package mysql

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

func TestStartMySQL(t *testing.T) {
	basedir := t.TempDir()
	datadir := filepath.Join(basedir, "data")
	if err := os.Mkdir(datadir, 0755); err != nil {
		t.Fatal(err)
	}

	mysql, err := Install(basedir, datadir, "root")
	if err != nil {
		t.Fatalf("Failed to start MySQL: %s", err)
	}
	if err := mysql.Start(13306, os.Stdout, os.Stderr, 60); err != nil {
		t.Fatalf("Failed to start MySQL: %s", err)
	}

	db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(localhost:%d)/mysql", mysql.Port()))
	if err != nil {
		t.Fatalf("Failed to open MySQL: %s", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping MySQL: %s", err)
	}

	if err := mysql.Stop(os.Stdout, os.Stderr); err != nil {
		t.Fatalf("Failed to stop MySQL: %s", err)
	}
}
