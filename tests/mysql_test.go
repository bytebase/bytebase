package tests

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/bytebase/bytebase/common/log"
	pluginmysql "github.com/bytebase/bytebase/plugin/db/mysql"
	resourcemysql "github.com/bytebase/bytebase/resources/mysql"
	"github.com/bytebase/bytebase/resources/mysqlutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestCheckEngineInnoDB(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()

	port := getTestPort(t.Name())

	t.Run("success", func(t *testing.T) {
		port := port
		t.Parallel()
		_, stopFn := resourcemysql.SetupTestInstance(t, port)
		defer stopFn()
		t.Log("create test database")
		database := "test_success"
		db, err := connectTestMySQL(port, "")
		a.NoError(err)
		defer db.Close()
		_, err = db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`; USE `%s`;", database, database))
		a.NoError(err)

		t.Log("create table with InnoDB engine")
		_, err = db.ExecContext(ctx, "CREATE TABLE t_innodb (id INT PRIMARY KEY) ENGINE=InnoDB;")
		a.NoError(err)

		t.Log("check InnoDB engine")
		driver, err := getTestMySQLDriver(ctx, strconv.Itoa(port), database)
		a.NoError(err)
		defer driver.Close(ctx)

		connCfg := getMySQLConnectionConfig(strconv.Itoa(port), database)

		mysqlDriver, ok := driver.(*pluginmysql.Driver)
		a.Equal(true, ok)
		mysqlRestore := pluginmysql.NewRestore(mysqlDriver, nil, connCfg, "" /*binlog directory*/)
		err = mysqlRestore.CheckEngineInnoDB(ctx, database)
		a.NoError(err)
	})

	t.Run("fail", func(t *testing.T) {
		port := port + 1
		t.Parallel()
		_, stopFn := resourcemysql.SetupTestInstance(t, port)
		defer stopFn()
		t.Log("create test database")
		database := "test_fail"
		db, err := connectTestMySQL(port, "")
		a.NoError(err)
		defer db.Close()
		_, err = db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`; USE `%s`;", database, database))
		a.NoError(err)

		t.Log("create table with InnoDB engine")
		_, err = db.ExecContext(ctx, "CREATE TABLE t_innodb (id INT PRIMARY KEY) ENGINE=InnoDB;")
		a.NoError(err)
		t.Log("create table with MyISAM engine")
		_, err = db.ExecContext(ctx, "CREATE TABLE t_myisam (id INT PRIMARY KEY) ENGINE=MyISAM;")
		a.NoError(err)

		t.Log("check InnoDB engine")
		driver, err := getTestMySQLDriver(ctx, strconv.Itoa(port), database)
		a.NoError(err)
		defer driver.Close(ctx)

		connCfg := getMySQLConnectionConfig(strconv.Itoa(port), database)

		mysqlDriver, ok := driver.(*pluginmysql.Driver)
		a.Equal(true, ok)
		mysqlRestore := pluginmysql.NewRestore(mysqlDriver, nil, connCfg, "" /*binlog directory*/)

		err = mysqlRestore.CheckEngineInnoDB(ctx, database)
		a.Error(err)
	})
}

func TestCheckServerVersionAndBinlogForPITR(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()

	port := getTestPort(t.Name())
	_, stopFn := resourcemysql.SetupTestInstance(t, port)
	defer stopFn()

	db, err := connectTestMySQL(port, "")
	a.NoError(err)
	defer db.Close()

	driver, err := getTestMySQLDriver(ctx, strconv.Itoa(port), "")
	a.NoError(err)
	defer driver.Close(ctx)

	connCfg := getMySQLConnectionConfig(strconv.Itoa(port), "")
	mysqlDriver, ok := driver.(*pluginmysql.Driver)
	a.Equal(true, ok)
	mysqlRestore := pluginmysql.NewRestore(mysqlDriver, nil, connCfg, "" /*binlog directory*/)

	// the test MySQL instance is 8.0.
	err = mysqlRestore.CheckServerVersionForPITR(ctx)
	a.NoError(err)

	// binlog is default to ON in MySQL 8.0.
	err = mysqlRestore.CheckBinlogEnabled(ctx)
	a.NoError(err)

	// binlog format is default to ROW in MySQL 8.0.
	err = mysqlRestore.CheckBinlogRowFormat(ctx)
	a.NoError(err)
}

func TestFetchBinlogFiles(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	log.SetLevel(zapcore.DebugLevel)

	port := getTestPort(t.Name())
	_, stopFn := resourcemysql.SetupTestInstance(t, port)
	defer stopFn()

	db, err := connectTestMySQL(port, "")
	a.NoError(err)
	defer db.Close()

	driver, err := getTestMySQLDriver(ctx, strconv.Itoa(port), "")
	a.NoError(err)
	defer driver.Close(ctx)

	connCfg := getMySQLConnectionConfig(strconv.Itoa(port), "")
	mysqlDriver, ok := driver.(*pluginmysql.Driver)
	a.Equal(true, ok)
	utilDir := t.TempDir()
	utilInstance, err := mysqlutil.Install(utilDir)
	a.NoError(err)
	binlogDir := t.TempDir()
	mysqlRestore := pluginmysql.NewRestore(mysqlDriver, utilInstance, connCfg, binlogDir)

	// init schema
	_, err = db.ExecContext(ctx, `
		CREATE DATABASE test;
		USE test;
		CREATE TABLE tbl (id int);
		`)
	a.NoError(err)

	// Rotate to create multiple binlog files for test.
	var startTsList []int64
	numRotate := 10
	// There'll be `numBinlogFiles` binlog files generated, and the last one contains no actual data.
	numBinlogFiles := numRotate + 1
	for i := 0; i < numRotate; i++ {
		// Record the start event ts of the current binlog file.
		startTsList = append(startTsList, time.Now().Unix())
		// Insert some data to grow the current binlog file.
		_, err = db.ExecContext(ctx, fmt.Sprintf(`
			USE test;
			INSERT INTO tbl VALUES (%d);
			`, i))
		a.NoError(err)
		// Rotate the binlog file.
		_, err = db.ExecContext(ctx, "FLUSH BINARY LOGS;")
		a.NoError(err)
	}
	t.Logf("startTsList: %v\n", startTsList)

	binlogFilesOnServerSorted, err := mysqlRestore.GetSortedBinlogFilesMetaOnServer(ctx)
	a.NoError(err)

	t.Log("Download binlog files in empty dir")
	binlogFilesBefore, err := ioutil.ReadDir(binlogDir)
	a.NoError(err)
	for _, file := range binlogFilesBefore {
		path := filepath.Join(binlogDir, file.Name())
		err = os.Remove(path)
		a.NoError(err)
	}
	err = mysqlRestore.FetchAllBinlogFiles(ctx)
	a.NoError(err)
	binlogFilesDownloaded, err := pluginmysql.GetSortedLocalBinlogFiles(binlogDir)
	a.NoError(err)
	a.Equal(numBinlogFiles, len(binlogFilesDownloaded))
	for j := range binlogFilesDownloaded {
		a.Equal(binlogFilesOnServerSorted[j].Name, binlogFilesDownloaded[j].Name)
		a.Equal(binlogFilesOnServerSorted[j].Size, binlogFilesDownloaded[j].Size)
	}

	t.Log("Truncate or delete downloaded files and re-download")
	rand.Seed(time.Now().Unix())
	// Fetch and randomly truncate/delete some binlog files.t.Log("Clean up binlog dir")
	binlogFiles, err := ioutil.ReadDir(binlogDir)
	a.NoError(err)
	for _, file := range binlogFiles {
		path := filepath.Join(binlogDir, file.Name())
		err = os.Remove(path)
		a.NoError(err)
	}
	t.Log("Fetch binlog files")
	err = mysqlRestore.FetchAllBinlogFiles(ctx)
	a.NoError(err)
	binlogFilesDownloaded, err = pluginmysql.GetSortedLocalBinlogFiles(binlogDir)
	a.NoError(err)
	t.Logf("Downloaded %d files to empty dir", len(binlogFilesDownloaded))
	truncateIndex := rand.Intn(numBinlogFiles)
	path := filepath.Join(binlogDir, binlogFilesDownloaded[truncateIndex].Name)
	t.Logf("Truncating file %s", binlogFilesDownloaded[truncateIndex].Name)
	err = os.Truncate(path, 1)
	a.NoError(err)
	deleteIndex := rand.Intn(numBinlogFiles)
	path = filepath.Join(binlogDir, binlogFilesDownloaded[deleteIndex].Name)
	t.Logf("Deleting file %s", binlogFilesDownloaded[deleteIndex].Name)
	err = os.Remove(path)
	a.NoError(err)
	// Re-download and check.
	t.Log("Re-downloading binlog files")
	err = mysqlRestore.FetchAllBinlogFiles(ctx)
	a.NoError(err)
	binlogFilesDownloadedAgain, err := pluginmysql.GetSortedLocalBinlogFiles(binlogDir)
	a.NoError(err)
	a.Equal(numBinlogFiles, len(binlogFilesDownloadedAgain))
	for i := range binlogFilesDownloadedAgain {
		a.Equal(binlogFilesOnServerSorted[i].Name, binlogFilesDownloadedAgain[i].Name)
		a.Equal(binlogFilesOnServerSorted[i].Size, binlogFilesDownloadedAgain[i].Size)
	}
}
