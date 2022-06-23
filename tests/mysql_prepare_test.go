//go:build mysql
// +build mysql

package tests

import (
	"testing"

	"github.com/bytebase/bytebase/resources/mysql"
	"github.com/stretchr/testify/require"
)

func TestPrepare(t *testing.T) {
	port := getTestPort(t.Name())
	t.Parallel()
	a := require.New(t)

	_, stopInstance := mysql.SetupTestInstance(t, port)
	defer stopInstance()

	db, err := connectTestMySQL(port, "")
	a.NoError(err)
	defer db.Close()

	const tableSize = 100000
	err = prepare(port, tableSize)
	a.NoError(err)

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM sbtest.sbtest1").Scan(&count)
	a.NoError(err)
	a.Equal(tableSize, count)
}
