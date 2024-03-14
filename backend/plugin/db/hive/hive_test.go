package hive

import (
	"context"
	"fmt"
	"testing"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func newHiveContextAndDriver() (Driver, context.Context) {
	var driver Driver
	ctx := context.Background()
	driver.config.Host = ""
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
	if err := driver.Ping(ctx); err != nil {
		t.Fatal("hive: bad connection")
	}
}

func TestHiveExecute(t *testing.T) {
	driver, ctx := newHiveContextAndDriver()
	if _, err := driver.Open(ctx, -1, driver.config); err != nil {
		t.Fatal("Connection fails")
	}
	affectedRows, err := driver.Execute(ctx, "INSERT INTO pokes(foo, bar) VALUES(2000, 'val_2002')", db.ExecuteOptions{})
	if err != nil {
		t.Fail()
	}
	fmt.Printf("Affected rows: %d\n", affectedRows)
}
