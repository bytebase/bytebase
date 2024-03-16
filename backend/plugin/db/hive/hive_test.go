package hive

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/beltran/gohive"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func newHiveContextAndDriver() (Driver, context.Context) {
	var driver Driver
	ctx := context.Background()
	driver.config.Host = "114.132.70.108"
	driver.config.Port = "10000"
	driver.config.Username = "hive"
	driver.config.Password = "hive"
	return driver, ctx
}

func TestHiveConnectionAndClose(t *testing.T) {
	driver, ctx := newHiveContextAndDriver()
	_, err := driver.Open(ctx, storepb.Engine_HIVE, driver.config)
	if err != nil {
		t.Fatal("connection fails")
	}
	if err := driver.Close(ctx); err != nil {
		t.Fatal("closing fails")
	}
}

func TestHivePing(t *testing.T) {
	driver, ctx := newHiveContextAndDriver()
	_, err := driver.Open(ctx, storepb.Engine_HIVE, driver.config)
	if err != nil {
		t.Fatal(err.Error())
	}
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	if err := driver.Ping(ctx); err != nil {
		t.Fatal(err.Error())
	}
}

func TestHiveQuery(t *testing.T) {
	driver, ctx := newHiveContextAndDriver()
	_, err := driver.Open(ctx, storepb.Engine_HIVE, driver.config)
	if err != nil {
		t.Fatal("connection fails")
	}

	testSQL := "SELECT VERSION()"

	results, err := driver.QueryConn(ctx, nil, testSQL, &db.QueryContext{})
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Printf("%v", results)
}

func TestHiveMetastore(t *testing.T) {
	driver, ctx := newHiveContextAndDriver()
	configuration := gohive.NewMetastoreConnectConfiguration()
	connection, err := gohive.ConnectToMetastore("47.236.165.100", 9083, "NOSASL", configuration)
	if err != nil {
		t.Fatal("cannot connect to HMS")
	}
	ctx = context.WithValue(ctx, "metastore", connection.Client)
	metaData, err := driver.SyncInstance(ctx)
	if err != nil {
		t.Fatal("failed to fetch metadata")
	}
	fmt.Println("THRIFT VERSION: ", metaData.Version)
}

func TestHiveSyncInstanceVersion(t *testing.T) {
	driver, ctx := newHiveContextAndDriver()
	_, err := driver.Open(ctx, storepb.Engine_HIVE, driver.config)
	if err != nil {
		t.Fatal(err)
	}
	instanceMeta, err := driver.SyncInstance(ctx)
	if instanceMeta != nil {
		fmt.Println(instanceMeta.Version)
	} else {
		t.Fatal("cannot sync instance")
	}
}
