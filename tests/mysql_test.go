package tests

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/bytebase/bytebase/common/log"
	pluginmysql "github.com/bytebase/bytebase/plugin/db/mysql"
	resourcemysql "github.com/bytebase/bytebase/resources/mysql"
	"github.com/bytebase/bytebase/resources/mysqlutil"
)

func TestCheckEngineInnoDB(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		mysqlPort := getTestPort()
		stopFn := resourcemysql.SetupTestInstance(t, mysqlPort)
		defer stopFn()
		// time.Sleep(1 * time.Minute)
		t.Log("create test database")
		database := "test_success"
		db, err := connectTestMySQL(mysqlPort, "")
		a.NoError(err)
		defer db.Close()
		_, err = db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE `%s`; USE `%s`;", database, database))
		a.NoError(err)

		t.Log("create table with InnoDB engine")
		_, err = db.ExecContext(ctx, "CREATE TABLE t_innodb (id INT PRIMARY KEY) ENGINE=InnoDB;")
		a.NoError(err)

		t.Log("check InnoDB engine")
		driver, err := getTestMySQLDriver(ctx, t, strconv.Itoa(mysqlPort), database, "")
		a.NoError(err)
		defer driver.Close(ctx)

		mysqlDriver, ok := driver.(*pluginmysql.Driver)
		a.Equal(true, ok)
		err = mysqlDriver.CheckEngineInnoDB(ctx, database)
		a.NoError(err)
	})

	t.Run("fail", func(t *testing.T) {
		t.Parallel()
		mysqlPort := getTestPort()
		stopFn := resourcemysql.SetupTestInstance(t, mysqlPort)
		defer stopFn()
		t.Log("create test database")
		database := "test_fail"
		db, err := connectTestMySQL(mysqlPort, "")
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
		driver, err := getTestMySQLDriver(ctx, t, strconv.Itoa(mysqlPort), database, "")
		a.NoError(err)
		defer driver.Close(ctx)

		mysqlDriver, ok := driver.(*pluginmysql.Driver)
		a.Equal(true, ok)

		err = mysqlDriver.CheckEngineInnoDB(ctx, database)
		a.Error(err)
	})
}

func TestCheckServerVersionAndBinlogForPITR(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()

	mysqlPort := getTestPort()
	stopFn := resourcemysql.SetupTestInstance(t, mysqlPort)
	defer stopFn()

	db, err := connectTestMySQL(mysqlPort, "")
	a.NoError(err)
	defer db.Close()

	driver, err := getTestMySQLDriver(ctx, t, strconv.Itoa(mysqlPort), "", "")
	a.NoError(err)
	defer driver.Close(ctx)

	mysqlDriver, ok := driver.(*pluginmysql.Driver)
	a.Equal(true, ok)

	// the test MySQL instance is 8.0.
	err = mysqlDriver.CheckServerVersionForPITR(ctx)
	a.NoError(err)

	// binlog is default to ON in MySQL 8.0.
	err = mysqlDriver.CheckBinlogEnabled(ctx)
	a.NoError(err)

	// binlog format is default to ROW in MySQL 8.0.
	err = mysqlDriver.CheckBinlogRowFormat(ctx)
	a.NoError(err)
}

func TestFetchBinlogFiles(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()
	log.SetLevel(zapcore.DebugLevel)

	mysqlPort := getTestPort()
	stopFn := resourcemysql.SetupTestInstance(t, mysqlPort)
	defer stopFn()

	db, err := connectTestMySQL(mysqlPort, "")
	a.NoError(err)
	defer db.Close()

	resourceDir := t.TempDir()
	binDir, err := mysqlutil.Install(resourceDir)
	a.NoError(err)

	driver, err := getTestMySQLDriver(ctx, t, strconv.Itoa(mysqlPort), "", binDir)
	a.NoError(err)
	defer driver.Close(ctx)

	mysqlDriver, ok := driver.(*pluginmysql.Driver)
	a.Equal(true, ok)
	binlogDir := mysqlDriver.GetBinlogDir()

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

	binlogFilesOnServerSorted, err := mysqlDriver.GetSortedBinlogFilesOnServer(ctx)
	a.NoError(err)

	t.Log("Download binlog files in empty dir")
	binlogFilesBefore, err := os.ReadDir(binlogDir)
	a.NoError(err)
	for _, file := range binlogFilesBefore {
		path := filepath.Join(binlogDir, file.Name())
		err = os.Remove(path)
		a.NoError(err)
	}
	err = mysqlDriver.FetchAllBinlogFiles(ctx, true /* downloadLatestBinlogFile */, nil)
	a.NoError(err)
	binlogFilesDownloaded, err := mysqlDriver.GetSortedLocalBinlogFiles()
	a.NoError(err)
	a.Equal(numBinlogFiles, len(binlogFilesDownloaded))
	for j := range binlogFilesDownloaded {
		a.Equal(binlogFilesOnServerSorted[j].Name, binlogFilesDownloaded[j].Name)
		a.Equal(binlogFilesOnServerSorted[j].Size, binlogFilesDownloaded[j].Size)
	}

	t.Log("Delete some downloaded files and re-download")
	rand.Seed(time.Now().Unix())
	// Fetch and randomly truncate/delete some binlog files.t.Log("Clean up binlog dir")
	binlogFiles, err := os.ReadDir(binlogDir)
	a.NoError(err)
	for _, file := range binlogFiles {
		path := filepath.Join(binlogDir, file.Name())
		err = os.Remove(path)
		a.NoError(err)
	}
	t.Log("Fetch binlog files")
	err = mysqlDriver.FetchAllBinlogFiles(ctx, true /* downloadLatestBinlogFile */, nil)
	a.NoError(err)
	binlogFilesDownloaded, err = mysqlDriver.GetSortedLocalBinlogFiles()
	a.NoError(err)
	t.Logf("Downloaded %d files to empty dir", len(binlogFilesDownloaded))
	deleteIndex := rand.Intn(numBinlogFiles)
	deletePath := filepath.Join(binlogDir, binlogFilesDownloaded[deleteIndex].Name)
	t.Logf("Deleting file %s", binlogFilesDownloaded[deleteIndex].Name)
	err = os.Remove(deletePath)
	a.NoError(err)
	err = os.Remove(deletePath + ".meta")
	a.NoError(err)
	// Re-download and check.
	t.Log("Re-downloading binlog files")
	err = mysqlDriver.FetchAllBinlogFiles(ctx, true /* downloadLatestBinlogFile */, nil)
	a.NoError(err)
	binlogFilesDownloadedAgain, err := mysqlDriver.GetSortedLocalBinlogFiles()
	a.NoError(err)
	a.Equal(numBinlogFiles, len(binlogFilesDownloadedAgain))
	for i := range binlogFilesDownloadedAgain {
		a.Equal(binlogFilesOnServerSorted[i].Name, binlogFilesDownloadedAgain[i].Name)
		a.Equal(binlogFilesOnServerSorted[i].Size, binlogFilesDownloadedAgain[i].Size)
	}
}
