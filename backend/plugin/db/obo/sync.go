package obo

import (
	"context"
	"errors"
	"time"

	"github.com/bytebase/bytebase/backend/plugin/db"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

func (*Driver) SyncInstance(context.Context) (*db.InstanceMetadata, error) {
	return nil, errors.New("not implemented")
}

func (*Driver) SyncDBSchema(context.Context) (*storepb.DatabaseSchemaMetadata, error) {
	return nil, errors.New("not implemented")
}

func (*Driver) SyncSlowQuery(context.Context, time.Time) (map[string]*storepb.SlowQueryStatistics, error) {
	return nil, errors.New("not implemented")
}

func (*Driver) CheckSlowQueryLogEnabled(context.Context) error {
	return errors.New("not implemented")
}
