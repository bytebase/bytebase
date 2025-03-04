package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// DataSourceMessage is the message for data source.
type DataSourceMessage struct {
	Options *storepb.DataSourceOptions
}

// FindDataSourceMessage is the message for finding a database.
type FindDataSourceMessage struct {
	ID         *string
	InstanceID *string
}

// UpdateDataSourceMessage is the message for the data source.
type UpdateDataSourceMessage struct {
	InstanceID string
	ID         string

	Options *storepb.DataSourceOptions
}

func (*Store) listInstanceDataSourceMap(ctx context.Context, tx *Tx, find *FindDataSourceMessage) (map[string][]*DataSourceMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if find.ID != nil {
		where, args = append(where, fmt.Sprintf("options->>'id' = $%d", len(args)+1)), append(args, *find.ID)
	}
	if find.InstanceID != nil {
		where, args = append(where, fmt.Sprintf("instance = $%d", len(args)+1)), append(args, *find.InstanceID)
	}

	instanceDataSourcesMap := make(map[string][]*DataSourceMessage)
	rows, err := tx.QueryContext(ctx, `
		SELECT
			instance,
			options
		FROM data_source
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var protoBytes []byte
		var instanceID string
		var dataSourceMessage DataSourceMessage
		if err := rows.Scan(
			&instanceID,
			&protoBytes,
		); err != nil {
			return nil, err
		}
		dataSourceOptions := &storepb.DataSourceOptions{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(protoBytes, dataSourceOptions); err != nil {
			return nil, err
		}
		dataSourceMessage.Options = dataSourceOptions
		instanceDataSourcesMap[instanceID] = append(instanceDataSourcesMap[instanceID], &dataSourceMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return instanceDataSourcesMap, nil
}

func (s *Store) ListDataSourcesV2(ctx context.Context, find *FindDataSourceMessage) ([]*DataSourceMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	dataSourceMap, err := s.listInstanceDataSourceMap(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	dataSources := []*DataSourceMessage{}
	for _, list := range dataSourceMap {
		dataSources = append(dataSources, list...)
	}
	return dataSources, nil
}

func (s *Store) GetDataSource(ctx context.Context, find *FindDataSourceMessage) (*DataSourceMessage, error) {
	dataSources, err := s.ListDataSourcesV2(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(dataSources) == 0 {
		return nil, nil
	}
	if len(dataSources) > 1 {
		return nil, errors.Errorf("found %d data sources, but expected 1", len(dataSources))
	}
	return dataSources[0], nil
}

// AddDataSourceToInstanceV2 adds a RO data source to an instance and return the instance where the data source is added.
func (s *Store) AddDataSourceToInstanceV2(ctx context.Context, instanceID string, dataSource *DataSourceMessage) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := s.addDataSourceToInstanceImplV2(ctx, tx, instanceID, dataSource); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.instanceCache.Remove(getInstanceCacheKey(instanceID))
	return nil
}

// RemoveDataSourceV2 removes a RO data source from an instance.
func (s *Store) RemoveDataSourceV2(ctx context.Context, instanceID string, dataSourceID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM data_source WHERE instance = $1 AND options->>'id' = $2;
	`, instanceID, dataSourceID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.instanceCache.Remove(getInstanceCacheKey(instanceID))
	return nil
}

// UpdateDataSourceV2 updates a data source and returns the instance.
func (s *Store) UpdateDataSourceV2(ctx context.Context, patch *UpdateDataSourceMessage) error {
	set, args := []string{}, []any{}

	if v := patch.Options; v != nil {
		o, err := protojson.Marshal(v)
		if err != nil {
			return err
		}
		set, args = append(set, fmt.Sprintf("options = $%d", len(args)+1)), append(args, o)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := fmt.Sprintf(`UPDATE data_source SET %s WHERE instance = $%d AND options->>'id' = $%d`, strings.Join(set, ", "), len(args)+1, len(args)+2)
	args = append(args, patch.InstanceID, patch.ID)
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.instanceCache.Remove(getInstanceCacheKey(patch.InstanceID))
	return nil
}

func (*Store) addDataSourceToInstanceImplV2(ctx context.Context, tx *Tx, instanceID string, dataSource *DataSourceMessage) error {
	protoBytes, err := protojson.Marshal(dataSource.Options)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO data_source (
			instance,
			options
		)
		VALUES ($1, $2)
	`, instanceID, protoBytes); err != nil {
		return err
	}

	return nil
}

// clearDataSourceImpl deletes dataSources by instance id and database id.
func (*Store) clearDataSourceImpl(ctx context.Context, tx *Tx, instanceID string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM data_source WHERE instance = $1`, instanceID); err != nil {
		return err
	}
	return nil
}
