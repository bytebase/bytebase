//go:build mysql
// +build mysql

package mysql

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestStartMySQL(t *testing.T) {
	basedir := t.TempDir()
	datadir := filepath.Join(basedir, "data")
	if err := os.Mkdir(datadir, 0755); err != nil {
		t.Fatal(err)
	}

	mysql, err := Install(basedir, datadir, "root")
	require.NoError(t, err)
	err = mysql.Start(13306 /* port */, os.Stdout, os.Stderr)
	require.NoError(t, err)

	db, err := sql.Open("mysql", fmt.Sprintf("root@tcp(localhost:%d)/mysql", mysql.Port()))
	require.NoError(t, err)
	defer db.Close()

	err = db.Ping()
	require.NoError(t, err)

	err = mysql.Stop(os.Stdout, os.Stderr)
	require.NoError(t, err)
}
