package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/plugin/advisor"
)

// EnvironmentCreate is the API message for creating an environment.
// TODO(ed): This is an temporary struct to compatible with OpenAPI and JSONAPI. Find way to move it into the API package.
type EnvironmentCreate struct {
	// Standard fields
	CreatorID int

	// Related fields
	EnvironmentTierPolicy  *api.EnvironmentTierPolicy
	PipelineApprovalPolicy *api.PipelineApprovalPolicy
	BackupPlanPolicy       *api.BackupPlanPolicy
	SQLReviewPolicy        *advisor.SQLReviewPolicy

	// Domain specific fields
	Name  string
	Order *int
}

// EnvironmentPatch is the API message for patching an environment.
// TODO(ed): This is an temporary struct to compatible with OpenAPI and JSONAPI. Find way to move it into the API package.
type EnvironmentPatch struct {
	ID int

	// Standard fields
	RowStatus *string
	UpdaterID int

	// Related fields
	EnvironmentTierPolicy  *api.EnvironmentTierPolicy
	PipelineApprovalPolicy *api.PipelineApprovalPolicy
	BackupPlanPolicy       *api.BackupPlanPolicy
	SQLReviewPolicy        *advisor.SQLReviewPolicy

	// Domain specific fields
	Name  *string
	Order *int
}

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

// toEnvironment creates an instance of Environment based on the environmentRaw.
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

// CreateEnvironment creates an instance of Environment.
func (s *Store) CreateEnvironment(ctx context.Context, create *EnvironmentCreate) (*api.Environment, error) {
	environmentRaw, err := s.createEnvironmentRaw(ctx, create)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create Environment with EnvironmentCreate[%+v]", create)
	}
	environment, err := s.composeEnvironment(ctx, environmentRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Environment with environmentRaw[%+v]", environmentRaw)
	}
	return environment, nil
}

// FindEnvironment finds a list of Environment instances.
func (s *Store) FindEnvironment(ctx context.Context, find *api.EnvironmentFind) ([]*api.Environment, error) {
	environmentRawList, err := s.findEnvironmentRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find Environment list with EnvironmentFind[%+v]", find)
	}
	sort.Slice(environmentRawList, func(i, j int) bool {
		return environmentRawList[i].Order < environmentRawList[j].Order
	})
	var environmentList []*api.Environment
	for _, raw := range environmentRawList {
		environment, err := s.composeEnvironment(ctx, raw)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose Environment role with environmentRaw[%+v]", raw)
		}
		environmentList = append(environmentList, environment)
	}
	return environmentList, nil
}

// PatchEnvironment patches an instance of Environment.
func (s *Store) PatchEnvironment(ctx context.Context, patch *EnvironmentPatch) (*api.Environment, error) {
	environmentRaw, err := s.patchEnvironmentRaw(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch Environment with EnvironmentPatch[%+v]", patch)
	}
	environment, err := s.composeEnvironment(ctx, environmentRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Environment role with environmentRaw[%+v]", environmentRaw)
	}
	return environment, nil
}

// GetEnvironmentByID gets an instance of Environment by ID.
func (s *Store) GetEnvironmentByID(ctx context.Context, id int) (*api.Environment, error) {
	envRaw, err := s.getEnvironmentByIDRaw(ctx, id)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get environment with ID %d", id)
	}
	if envRaw == nil {
		return nil, common.Errorf(common.NotFound, "environment %d not found", id)
	}

	env, err := s.composeEnvironment(ctx, envRaw)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose environment with environmentRaw[%+v]", envRaw)
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

	tier, err := s.GetEnvironmentTierPolicyByEnvID(ctx, env.ID)
	if err != nil {
		return nil, err
	}
	env.Tier = tier.EnvironmentTier

	return env, nil
}

// createEnvironmentRaw creates a new environment.
func (s *Store) createEnvironmentRaw(ctx context.Context, create *EnvironmentCreate) (*environmentRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	environment, err := s.createEnvironmentImpl(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if p := create.PipelineApprovalPolicy; p != nil {
		if _, err := s.upsertEnvironmentPolicy(ctx, tx, environment, api.PolicyTypePipelineApproval, p); err != nil {
			return nil, err
		}
	}
	if p := create.BackupPlanPolicy; p != nil {
		if _, err := s.upsertEnvironmentPolicy(ctx, tx, environment, api.PolicyTypeBackupPlan, p); err != nil {
			return nil, err
		}
	}
	if p := create.EnvironmentTierPolicy; p != nil {
		if _, err := s.upsertEnvironmentPolicy(ctx, tx, environment, api.PolicyTypeEnvironmentTier, p); err != nil {
			return nil, err
		}
	}
	if p := create.SQLReviewPolicy; p != nil {
		if _, err := s.upsertEnvironmentPolicy(ctx, tx, environment, api.PolicyTypeSQLReview, p); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(environmentCacheNamespace, environment.ID, environment); err != nil {
		return nil, err
	}

	return environment, nil
}

func (s *Store) upsertEnvironmentPolicy(
	ctx context.Context,
	tx *Tx,
	env *environmentRaw,
	policyType api.PolicyType,
	policy interface{},
) (*policyRaw, error) {
	bytes, err := json.Marshal(policy)
	payload := string(bytes)
	if err != nil {
		return nil, FormatError(err)
	}

	policyUpsert := &api.PolicyUpsert{
		ResourceType: api.PolicyResourceTypeEnvironment,
		ResourceID:   env.ID,
		Type:         policyType,
		UpdaterID:    env.CreatorID,
		Payload:      &payload,
	}

	policyRaw, err := upsertPolicyImpl(ctx, tx, policyUpsert)
	if err != nil {
		return nil, err
	}

	if err := s.upsertPolicyCache(policyUpsert.Type, policyUpsert.ResourceID, policyRaw.Payload); err != nil {
		return nil, err
	}

	return policyRaw, nil
}

// findEnvironmentRaw retrieves a list of environments based on find.
func (s *Store) findEnvironmentRaw(ctx context.Context, find *api.EnvironmentFind) ([]*environmentRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	list, err := s.findEnvironmentImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err == nil {
		for _, environment := range list {
			if err := s.cache.UpsertCache(environmentCacheNamespace, environment.ID, environment); err != nil {
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
	has, err := s.cache.FindCache(environmentCacheNamespace, id, envRaw)
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
	defer tx.Rollback()

	find := &api.EnvironmentFind{ID: &id}
	envRawList, err := s.findEnvironmentImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if len(envRawList) == 0 {
		return nil, nil
	} else if len(envRawList) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d environments with filter %+v, expect 1", len(envRawList), find)}
	}
	if err := s.cache.UpsertCache(environmentCacheNamespace, envRawList[0].ID, envRawList[0]); err != nil {
		return nil, err
	}
	return envRawList[0], nil
}

// patchEnvironmentRaw updates an existing environment by ID.
// Returns ENOTFOUND if environment does not exist.
func (s *Store) patchEnvironmentRaw(ctx context.Context, patch *EnvironmentPatch) (*environmentRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	envRaw, err := s.patchEnvironmentImpl(ctx, tx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if p := patch.PipelineApprovalPolicy; p != nil {
		if _, err := s.upsertEnvironmentPolicy(ctx, tx, envRaw, api.PolicyTypePipelineApproval, p); err != nil {
			return nil, err
		}
	}
	if p := patch.BackupPlanPolicy; p != nil {
		if _, err := s.upsertEnvironmentPolicy(ctx, tx, envRaw, api.PolicyTypeBackupPlan, p); err != nil {
			return nil, err
		}
	}
	if p := patch.EnvironmentTierPolicy; p != nil {
		if _, err := s.upsertEnvironmentPolicy(ctx, tx, envRaw, api.PolicyTypeEnvironmentTier, p); err != nil {
			return nil, err
		}
	}
	if p := patch.SQLReviewPolicy; p != nil {
		if _, err := s.upsertEnvironmentPolicy(ctx, tx, envRaw, api.PolicyTypeSQLReview, p); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if err := s.cache.UpsertCache(environmentCacheNamespace, envRaw.ID, envRaw); err != nil {
		return nil, err
	}

	return envRaw, nil
}

// createEnvironmentImpl creates a new environment.
func (Store) createEnvironmentImpl(ctx context.Context, tx *Tx, create *EnvironmentCreate) (*environmentRaw, error) {
	var order int

	if create.Order != nil {
		order = *create.Order
	} else {
		// The order is the MAX(order) + 1
		if err := tx.QueryRowContext(ctx, `
		SELECT "order"
		FROM environment
		ORDER BY "order" DESC
		LIMIT 1
	`).Scan(&order); err != nil {
			if err == sql.ErrNoRows {
				return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("no environment record found")}
			}
			return nil, FormatError(err)
		}
		order++
	}

	// TODO(d): allow users to set resource_id.
	resourceID := strings.ToLower(create.Name)
	// Insert row into database.
	query := `
		INSERT INTO environment (
			creator_id,
			updater_id,
			name,
			"order",
			resource_id
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, name, "order"
	`
	var envRaw environmentRaw
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.CreatorID,
		create.Name,
		order,
		resourceID,
	).Scan(
		&envRaw.ID,
		&envRaw.RowStatus,
		&envRaw.CreatorID,
		&envRaw.CreatedTs,
		&envRaw.UpdaterID,
		&envRaw.UpdatedTs,
		&envRaw.Name,
		&envRaw.Order,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	return &envRaw, nil
}

func (*Store) findEnvironmentImpl(ctx context.Context, tx *Tx, find *api.EnvironmentFind) ([]*environmentRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Name; v != nil {
		where, args = append(where, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
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
func (*Store) patchEnvironmentImpl(ctx context.Context, tx *Tx, patch *EnvironmentPatch) (*environmentRaw, error) {
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

	var environment environmentRaw
	// Execute update query with RETURNING.
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE environment
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, name, "order"
	`, len(args)),
		args...,
	).Scan(
		&environment.ID,
		&environment.RowStatus,
		&environment.CreatorID,
		&environment.CreatedTs,
		&environment.UpdaterID,
		&environment.UpdatedTs,
		&environment.Name,
		&environment.Order,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("environment ID not found: %d", patch.ID)}
		}
		return nil, FormatError(err)
	}
	return &environment, nil
}

// EnvironmentMessage is the mssage for environment.
type EnvironmentMessage struct {
	EnvironmentID string
	Title         string
	Order         int32
	Deleted       bool
}

// UpdateEnvironmentMessage is the message for updating an environment.
type UpdateEnvironmentMessage struct {
	Name  *string
	Order *int32
}

// GetEnvironmentV2 gets environment by resource ID.
func (s *Store) GetEnvironmentV2(ctx context.Context, resourceID string) (*EnvironmentMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	var environmentMessage EnvironmentMessage
	var rowStatus string
	if err := tx.QueryRowContext(ctx, `
		SELECT
			resource_id,
			name,
			"order",
			row_status
		FROM environment
		WHERE resource_id = $1`,
		resourceID,
	).Scan(
		&environmentMessage.EnvironmentID,
		&environmentMessage.Title,
		&environmentMessage.Order,
		&rowStatus,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, FormatError(err)
	}
	environmentMessage.Deleted = convertRowStatusToDeleted(rowStatus)

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return &environmentMessage, nil
}

// ListEnvironmentV2 lists all environment.
func (s *Store) ListEnvironmentV2(ctx context.Context, showDeleted bool) ([]*EnvironmentMessage, error) {
	where, args := []string{"1 = 1"}, []interface{}{}
	if !showDeleted {
		where, args = append(where, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, api.Normal)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	var environmentMessages []*EnvironmentMessage
	rows, err := tx.QueryContext(ctx, `
		SELECT
			resource_id
			name,
			"order",
			row_status
		FROM environment
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var environmentMessage EnvironmentMessage
		var rowStatus string
		if err := rows.Scan(
			&environmentMessage.EnvironmentID,
			&environmentMessage.Title,
			&environmentMessage.Order,
			&rowStatus,
		); err != nil {
			return nil, FormatError(err)
		}
		environmentMessage.Deleted = convertRowStatusToDeleted(rowStatus)

		environmentMessages = append(environmentMessages, &environmentMessage)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return environmentMessages, nil
}

// CreateEnvironmentV2 creates an environment.
func (s *Store) CreateEnvironmentV2(ctx context.Context, environmentMessage *EnvironmentMessage, creatorID int) (*EnvironmentMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
			INSERT INTO environment (
				resource_id,
				name,
				"order",
				creator_id,
				updater_id,
			)
			VALUES ($1, $2, $3, $4, $5)
		`,
		environmentMessage.EnvironmentID,
		environmentMessage.Title,
		environmentMessage.Order,
		creatorID,
		creatorID,
	); err != nil {
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return &EnvironmentMessage{
		EnvironmentID: environmentMessage.EnvironmentID,
		Title:         environmentMessage.Title,
		Order:         environmentMessage.Order,
		Deleted:       false,
	}, nil
}

// UpdateEnvironmentV2 updates an environment.
func (s *Store) UpdateEnvironmentV2(ctx context.Context, environmentID string, patch *UpdateEnvironmentMessage, updaterID int) (*EnvironmentMessage, error) {
	set, args := []string{"updater_id = $1"}, []interface{}{fmt.Sprintf("%d", updaterID)}
	if v := patch.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Order; v != nil {
		set, args = append(set, fmt.Sprintf(`"order" = $%d`, len(args)+1)), append(args, *v)
	}
	args = append(args, environmentID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	var environmentMessage EnvironmentMessage
	var rowStatus string
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
			UPDATE environment
			SET `+strings.Join(set, ", ")+`
			WHERE resource_id = $%d
			RETURNING resource_id, name, "order", row_status
		`, len(args)),
		args...,
	).Scan(
		&environmentMessage.EnvironmentID,
		&environmentMessage.Title,
		&environmentMessage.Order,
		&rowStatus,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, FormatError(err)
	}
	environmentMessage.Deleted = convertRowStatusToDeleted(rowStatus)

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return &environmentMessage, nil
}

// DeleteOrUndeleteEnvironmentV2 deletes or undeletes an environment (archiving).
func (s *Store) DeleteOrUndeleteEnvironmentV2(ctx context.Context, environmentID string, delete bool, updaterID int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.Rollback()

	rowStatus := api.Normal
	if delete {
		rowStatus = api.Archived
	}
	if _, err := tx.ExecContext(ctx, `
			UPDATE environment
			SET
				row_status = $1,
				updater_id = $2
			WHERE resource_id = $3
		`,
		rowStatus,
		updaterID,
		environmentID,
	); err != nil {
		return FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

func convertRowStatusToDeleted(rowStatus string) bool {
	return rowStatus == string(api.Archived)
}
