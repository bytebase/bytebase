package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// environmentRaw is the store model for an Environment.
// Fields have exactly the same meanings as Environment.
type environmentRaw struct {
	ID int

	// Standard fields
	RowStatus api.RowStatus
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Domain specific fields
	Name  string
	Order int
}

// toEnvironment creates an instance of Environment based on the EnvironmentRaw.
// This is intended to be called when we need to compose an Environment relationship.
func (raw *environmentRaw) toEnvironment() *api.Environment {
	return &api.Environment{
		ID: raw.ID,

		RowStatus: raw.RowStatus,
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		Name:  raw.Name,
		Order: raw.Order,
	}
}

// CreateEnvironment creates an instance of Environment
func (s *Store) CreateEnvironment(ctx context.Context, create *api.EnvironmentCreate) (*api.Environment, error) {
	EnvironmentRaw, err := s.createEnvironmentRaw(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Environment with EnvironmentCreate[%+v], error[%w]", create, err)
	}
	Environment, err := s.composeEnvironment(ctx, EnvironmentRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose Environment with EnvironmentRaw[%+v], error[%w]", EnvironmentRaw, err)
	}
	return Environment, nil
}

// FindEnvironment finds a list of Environment instances
func (s *Store) FindEnvironment(ctx context.Context, find *api.EnvironmentFind) ([]*api.Environment, error) {
	EnvironmentRawList, err := s.findEnvironmentListRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("Failed to find Environment list, error[%w]", err)
	}
	var EnvironmentList []*api.Environment
	for _, raw := range EnvironmentRawList {
		Environment, err := s.composeEnvironment(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("Failed to compose Environment role with EnvironmentRaw[%+v], error[%w]", raw, err)
		}
		EnvironmentList = append(EnvironmentList, Environment)
	}
	return EnvironmentList, nil
}

// PatchEnvironment patches an instance of Environment
func (s *Store) PatchEnvironment(ctx context.Context, patch *api.EnvironmentPatch) (*api.Environment, error) {
	EnvironmentRaw, err := s.patchEnvironmentRaw(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("Failed to patch Environment with EnvironmentPatch[%+v], error[%w]", patch, err)
	}
	Environment, err := s.composeEnvironment(ctx, EnvironmentRaw)
	if err != nil {
		return nil, fmt.Errorf("Failed to compose Environment role with EnvironmentRaw[%+v], error[%w]", EnvironmentRaw, err)
	}
	return Environment, nil
}

// GetEnvironmentByID gets an instance of Environment by ID
func (s *Store) GetEnvironmentByID(ctx context.Context, id int) (*api.Environment, error) {
	envRaw, err := s.getEnvironmentByIDRaw(ctx, id)
	if err != nil {
		return nil, err
	}
	if envRaw == nil {
		return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("environment not found with ID %v", id)}
	}

	env, err := s.composeEnvironment(ctx, envRaw)
	if err != nil {
		return nil, err
	}

	return env, nil
}

//
// private functions
//

func (s *Store) composeEnvironment(ctx context.Context, raw *environmentRaw) (*api.Environment, error) {
	env := raw.toEnvironment()

	creator, err := s.GetPrincipalByID(ctx, env.CreatorID)
	if err != nil {
		return nil, err
	}
	env.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, env.UpdaterID)
	if err != nil {
		return nil, err
	}
	env.Updater = updater

	return env, nil
}

// createEnvironmentRaw creates a new environment.
func (s *Store) createEnvironmentRaw(ctx context.Context, create *api.EnvironmentCreate) (*environmentRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	environment, err := s.createEnvironmentImpl(ctx, tx.PTx, create)
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

// findEnvironmentListRaw retrieves a list of environments based on find.
func (s *Store) findEnvironmentListRaw(ctx context.Context, find *api.EnvironmentFind) ([]*environmentRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := s.findEnvironmentImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
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

// getEnvironmentByIDRaw retrieves a single environment based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getEnvironmentByIDRaw(ctx context.Context, id int) (*environmentRaw, error) {
	envRaw := &environmentRaw{}
	has, err := s.cache.FindCache(api.EnvironmentCache, id, envRaw)
	if err != nil {
		return nil, err
	}
	if has {
		return envRaw, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	find := &api.EnvironmentFind{ID: &id}
	envRawList, err := s.findEnvironmentImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if len(envRawList) == 0 {
		return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("environment not found with EnvironmentFind[%+v]", find)}
	} else if len(envRawList) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d environments with filter %+v, expect 1", len(envRawList), find)}
	}
	if err := s.cache.UpsertCache(api.EnvironmentCache, envRawList[0].ID, envRawList[0]); err != nil {
		return nil, err
	}
	return envRawList[0], nil
}

// patchEnvironmentRaw updates an existing environment by ID.
// Returns ENOTFOUND if environment does not exist.
func (s *Store) patchEnvironmentRaw(ctx context.Context, patch *api.EnvironmentPatch) (*environmentRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	envRaw, err := s.patchEnvironmentImpl(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(api.EnvironmentCache, envRaw.ID, envRaw); err != nil {
		return nil, err
	}

	return envRaw, nil
}

// createEnvironmentImpl creates a new environment.
func (s *Store) createEnvironmentImpl(ctx context.Context, tx *sql.Tx, create *api.EnvironmentCreate) (*environmentRaw, error) {
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
	var envRaw environmentRaw
	if err := row2.Scan(
		&envRaw.ID,
		&envRaw.RowStatus,
		&envRaw.CreatorID,
		&envRaw.CreatedTs,
		&envRaw.UpdaterID,
		&envRaw.UpdatedTs,
		&envRaw.Name,
		&envRaw.Order,
	); err != nil {
		fmt.Printf("Yang4: %v\n", err)
		return nil, FormatError(err)
	}

	return &envRaw, nil
}

func (s *Store) findEnvironmentImpl(ctx context.Context, tx *sql.Tx, find *api.EnvironmentFind) ([]*environmentRaw, error) {
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
	var envRawList []*environmentRaw
	for rows.Next() {
		var environment environmentRaw
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

		envRawList = append(envRawList, &environment)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return envRawList, nil
}

// patchEnvironmentImpl updates a environment by ID. Returns the new state of the environment after update.
func (s *Store) patchEnvironmentImpl(ctx context.Context, tx *sql.Tx, patch *api.EnvironmentPatch) (*environmentRaw, error) {
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
		var environment environmentRaw
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
