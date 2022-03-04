package mysql

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

func TestStartMySQL(t *testing.T) {
	basedir, err := os.MkdirTemp("", "mysql_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	datadir := filepath.Join(basedir, "data")
	if err := os.Mkdir(datadir, 0755); err != nil {
		t.Fatal(err)
	}

	mysql, err := Install(basedir, datadir, "root")
	if err != nil {
		t.Fatalf("Failed to start MySQL: %s", err)
	}
	if err := mysql.Start(0, os.Stdout, os.Stderr, 60); err != nil {
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

	if err := os.RemoveAll(basedir); err != nil {
		t.Errorf("Failed to remove MySQL instance: %s", err)
	}
}

func TestRandomUnusedPort(t *testing.T) {
	port, err := randomUnusedPort()
	if err != nil {
		t.Fatalf("Failed to get random unused port: %s", err)
	}

	if port == 0 {
		t.Fatalf("Random unused port is 0")
	}

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Fatalf("Failed to listen on port %d: %s", port, err)
	}
	defer l.Close()
}
