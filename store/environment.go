package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"go.uber.org/zap"
)

var (
	_ api.EnvironmentService = (*EnvironmentService)(nil)
)

// EnvironmentService represents a service for managing environment.
type EnvironmentService struct {
	l  *zap.Logger
	db *DB

	cache api.CacheService
}

// NewEnvironmentService returns a new instance of EnvironmentService.
func NewEnvironmentService(logger *zap.Logger, db *DB, cache api.CacheService) *EnvironmentService {
	return &EnvironmentService{l: logger, db: db, cache: cache}
}

// CreateEnvironment creates a new environment.
func (s *EnvironmentService) CreateEnvironment(ctx context.Context, create *api.EnvironmentCreate) (*api.Environment, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	environment, err := s.pgCreateEnvironment(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(api.EnvironmentCache, environment.ID, environment); err != nil {
		return nil, err
	}

	return environment, nil
}

// FindEnvironmentList retrieves a list of environments based on find.
func (s *EnvironmentService) FindEnvironmentList(ctx context.Context, find *api.EnvironmentFind) ([]*api.Environment, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findEnvironmentList(ctx, tx.PTx, find)
	if err != nil {
		return []*api.Environment{}, err
	}

	if err == nil {
		for _, environment := range list {
			if err := s.cache.UpsertCache(api.EnvironmentCache, environment.ID, environment); err != nil {
				return nil, err
			}
		}
	}

	return list, nil
}

// FindEnvironment retrieves a single environment based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *EnvironmentService) FindEnvironment(ctx context.Context, find *api.EnvironmentFind) (*api.Environment, error) {
	if find.ID != nil {
		environment := &api.Environment{}
		has, err := s.cache.FindCache(api.EnvironmentCache, *find.ID, environment)
		if err != nil {
			return nil, err
		}
		if has {
			return environment, nil
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findEnvironmentList(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d environments with filter %+v, expect 1", len(list), find)}
	}
	if err := s.cache.UpsertCache(api.EnvironmentCache, list[0].ID, list[0]); err != nil {
		return nil, err
	}
	return list[0], nil
}

// PatchEnvironment updates an existing environment by ID.
// Returns ENOTFOUND if environment does not exist.
func (s *EnvironmentService) PatchEnvironment(ctx context.Context, patch *api.EnvironmentPatch) (*api.Environment, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	environment, err := s.pgPatchEnvironment(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(api.EnvironmentCache, environment.ID, environment); err != nil {
		return nil, err
	}

	return environment, nil
}

// createEnvironment creates a new environment.
func (s *EnvironmentService) pgCreateEnvironment(ctx context.Context, tx *sql.Tx, create *api.EnvironmentCreate) (*api.Environment, error) {
	// The order is the MAX(order) + 1
	row1, err1 := tx.QueryContext(ctx, `
		SELECT "order"
		FROM environment
		ORDER BY "order" DESC
		LIMIT 1
	`)
	fmt.Printf("Yang1: %v\n", err1)
	if err1 != nil {
		return nil, FormatError(err1)
	}

	row1.Next()
	var order int
	if err1 := row1.Scan(
		&order,
	); err1 != nil {
		fmt.Printf("Yang2: %v\n", err1)
		return nil, FormatError(err1)
	}
	if err := row1.Close(); err != nil {
		return nil, FormatError(err)
	}

	// Insert row into database.
	row2, err2 := tx.QueryContext(ctx, `
		INSERT INTO environment (
			creator_id,
			updater_id,
			name,
			"order"
		)
		VALUES ($1, $2, $3, $4)
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, name, "order"
	`,
		create.CreatorID,
		create.CreatorID,
		create.Name,
		order+1,
	)

	fmt.Printf("Yang3: %v\n", err2)
	if err2 != nil {
		return nil, FormatError(err2)
	}
	defer row2.Close()

	row2.Next()
	var environment api.Environment
	if err := row2.Scan(
		&environment.ID,
		&environment.RowStatus,
		&environment.CreatorID,
		&environment.CreatedTs,
		&environment.UpdaterID,
		&environment.UpdatedTs,
		&environment.Name,
		&environment.Order,
	); err != nil {
		fmt.Printf("Yang4: %v\n", err)
		return nil, FormatError(err)
	}

	return &environment, nil
}

func (s *EnvironmentService) findEnvironmentList(ctx context.Context, tx *sql.Tx, find *api.EnvironmentFind) (_ []*api.Environment, err error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.RowStatus; v != nil {
		where, args = append(where, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			row_status,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			name,
			"order"
		FROM environment
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into list.
	list := make([]*api.Environment, 0)
	for rows.Next() {
		var environment api.Environment
		if err := rows.Scan(
			&environment.ID,
			&environment.RowStatus,
			&environment.CreatorID,
			&environment.CreatedTs,
			&environment.UpdaterID,
			&environment.UpdatedTs,
			&environment.Name,
			&environment.Order,
		); err != nil {
			return nil, FormatError(err)
		}

		list = append(list, &environment)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return list, nil
}

// pgPatchEnvironment updates a environment by ID. Returns the new state of the environment after update.
func (s *EnvironmentService) pgPatchEnvironment(ctx context.Context, tx *sql.Tx, patch *api.EnvironmentPatch) (*api.Environment, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.RowStatus; v != nil {
		set, args = append(set, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, api.RowStatus(*v))
	}
	if v := patch.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Order; v != nil {
		set, args = append(set, fmt.Sprintf(`"order" = $%d`, len(args)+1)), append(args, *v)
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, fmt.Sprintf(`
		UPDATE environment
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, name, "order"
	`, len(args)),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var environment api.Environment
		if err := row.Scan(
			&environment.ID,
			&environment.RowStatus,
			&environment.CreatorID,
			&environment.CreatedTs,
			&environment.UpdaterID,
			&environment.UpdatedTs,
			&environment.Name,
			&environment.Order,
		); err != nil {
			return nil, FormatError(err)
		}
		return &environment, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("environment ID not found: %d", patch.ID)}
}
