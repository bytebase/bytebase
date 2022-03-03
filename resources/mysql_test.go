package resources

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"path"
	"testing"

	_ "github.com/go-sql-driver/mysql"
)

func TestExtractMySQLTarGz(t *testing.T) {
	testDir := os.TempDir()
	if err := os.MkdirAll(testDir, os.ModePerm); err != nil {
		t.Fatalf("Failed to create test directory %q: %s", testDir, err)
	}
	defer os.RemoveAll(testDir)

	f, err := mysqlResources.Open("mysql-8.0.28-macos11-arm64.tar.gz")
	if err != nil {
		t.Fatalf("Failed to open MySQL tar.gz: %s", err)
	}
	defer f.Close()

	if err := extractTarGz(f, testDir); err != nil {
		t.Fatalf("Failed to extract MySQL tar.gz: %s", err)
	}

	if _, err := os.Stat(path.Join(testDir, "mysql-8.0.28-macos11-arm64", "bin", "mysqld")); err != nil {
		t.Fatalf("Failed to stat MySQL binary: %s", err)
	}
}

func TestStartMySQL(t *testing.T) {
	mysql, err := CreateMysqlInstance()
	if err != nil {
		t.Fatalf("Failed to start MySQL: %s", err)
	}
	defer mysql.Purge()

	db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(localhost:%d)/mysql", mysql.Port()))
	if err != nil {
		t.Fatalf("Failed to open MySQL: %s", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Fatalf("Failed to ping MySQL: %s", err)
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
