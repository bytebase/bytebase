package store

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// EnvironmentCreate is the API message for creating an environment.
// TODO(ed): This is an temporary struct to compatible with OpenAPI and JSONAPI. Find way to move it into the API package.
type EnvironmentCreate struct {
	// Standard fields
	CreatorID int

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

	// Domain specific fields
	Name  *string
	Order *int
}

// CreateEnvironment creates an instance of Environment.
func (s *Store) CreateEnvironment(ctx context.Context, create *EnvironmentCreate) (*api.Environment, error) {
	if create.Order == nil {
		return nil, errors.Errorf("order must be set in legacy CreateEnvironment")
	}
	environment, err := s.CreateEnvironmentV2(ctx, &EnvironmentMessage{
		ResourceID: strings.ToLower(create.Name),
		Title:      create.Name,
		Order:      int32(*create.Order),
		Protected:  false,
	}, create.CreatorID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create Environment with EnvironmentCreate[%+v]", create)
	}

	composedEnvironment, err := s.composeEnvironment(ctx, environment)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Environment with environmentRaw[%+v]", environment)
	}
	return composedEnvironment, nil
}

// FindEnvironment finds a list of Environment instances.
func (s *Store) FindEnvironment(ctx context.Context, find *api.EnvironmentFind) ([]*api.Environment, error) {
	v2Find := &FindEnvironmentMessage{ShowDeleted: true}
	environments, err := s.ListEnvironmentV2(ctx, v2Find)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find Environment list with ListEnvironmentV2[%+v]", find)
	}
	sort.Slice(environments, func(i, j int) bool {
		return environments[i].Order < environments[j].Order
	})
	var composedEnvironmentList []*api.Environment
	for _, environment := range environments {
		composedEnvironment, err := s.composeEnvironment(ctx, environment)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose Environment role with environmentRaw[%+v]", environment)
		}
		if find.RowStatus != nil && composedEnvironment.RowStatus != *find.RowStatus {
			continue
		}
		composedEnvironmentList = append(composedEnvironmentList, composedEnvironment)
	}
	return composedEnvironmentList, nil
}

// PatchEnvironment patches an instance of Environment.
func (s *Store) PatchEnvironment(ctx context.Context, patch *EnvironmentPatch) (*api.Environment, error) {
	environment, err := s.GetEnvironmentV2(ctx, &FindEnvironmentMessage{UID: &patch.ID})
	if err != nil {
		return nil, err
	}
	v2Update := &UpdateEnvironmentMessage{
		Name: patch.Name,
	}
	if patch.Order != nil {
		order := int32(*patch.Order)
		v2Update.Order = &order
	}
	if patch.RowStatus != nil {
		deleted := false
		if *patch.RowStatus == string(api.Archived) {
			deleted = true
		}
		v2Update.Delete = &deleted
	}
	environment, err = s.UpdateEnvironmentV2(ctx, environment.ResourceID, v2Update, patch.UpdaterID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch Environment with EnvironmentPatch[%+v]", patch)
	}
	composedEnvironment, err := s.composeEnvironment(ctx, environment)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Environment with environmentRaw[%+v]", environment)
	}
	return composedEnvironment, nil
}

// GetEnvironmentByID gets an instance of Environment by ID.
func (s *Store) GetEnvironmentByID(ctx context.Context, id int) (*api.Environment, error) {
	environment, err := s.GetEnvironmentV2(ctx, &FindEnvironmentMessage{UID: &id})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get environment with ID %d", id)
	}
	if environment == nil {
		return nil, common.Errorf(common.NotFound, "environment %d not found", id)
	}

	env, err := s.composeEnvironment(ctx, environment)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose environment with environmentRaw[%+v]", environment)
	}

	return env, nil
}

//
// private functions
//

func (s *Store) composeEnvironment(ctx context.Context, raw *EnvironmentMessage) (*api.Environment, error) {
	rowStatus := api.Normal
	if raw.Deleted {
		rowStatus = api.Archived
	}
	tier := api.EnvironmentTierValueUnprotected
	if raw.Protected {
		tier = api.EnvironmentTierValueProtected
	}
	ret := &api.Environment{
		ID:         raw.UID,
		ResourceID: raw.ResourceID,

		RowStatus: rowStatus,
		CreatorID: api.SystemBotID,
		CreatedTs: 0,
		UpdaterID: api.SystemBotID,
		UpdatedTs: 0,

		Name:  raw.Title,
		Order: int(raw.Order),
		Tier:  tier,
	}
	bot, err := s.GetPrincipalByID(ctx, api.SystemBotID)
	if err != nil {
		return nil, err
	}
	ret.Creator = bot
	ret.Updater = bot

	return ret, nil
}

// EnvironmentMessage is the mssage for environment.
type EnvironmentMessage struct {
	ResourceID string
	Title      string
	Order      int32
	Protected  bool

	// The following fields are output only and not used for create().
	UID     int
	Deleted bool
}

// FindEnvironmentMessage is the message to find environments.
type FindEnvironmentMessage struct {
	// We should only set either UID or ResourceID.
	// Deprecate UID later once we fully migrate to ResourceID.
	UID         *int
	ResourceID  *string
	ShowDeleted bool
}

// UpdateEnvironmentMessage is the message for updating an environment.
type UpdateEnvironmentMessage struct {
	Name      *string
	Order     *int32
	Protected *bool
	Delete    *bool
}

// GetEnvironmentV2 gets environment by resource ID.
func (s *Store) GetEnvironmentV2(ctx context.Context, find *FindEnvironmentMessage) (*EnvironmentMessage, error) {
	if find.ResourceID != nil {
		if environment, ok := s.environmentCache[*find.ResourceID]; ok {
			return environment, nil
		}
	}
	if find.UID != nil {
		if environment, ok := s.environmentIDCache[*find.UID]; ok {
			return environment, nil
		}
	}

	// We will always return the resource regardless of its deleted state.
	find.ShowDeleted = true

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	environment, err := s.getEnvironmentImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	if environment == nil {
		return nil, nil
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	s.environmentCache[environment.ResourceID] = environment
	s.environmentIDCache[environment.UID] = environment
	return environment, nil
}

// ListEnvironmentV2 lists all environment.
func (s *Store) ListEnvironmentV2(ctx context.Context, find *FindEnvironmentMessage) ([]*EnvironmentMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	environments, err := listEnvironmentImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	for _, environment := range environments {
		s.environmentCache[environment.ResourceID] = environment
		s.environmentIDCache[environment.UID] = environment
	}
	return environments, nil
}

// CreateEnvironmentV2 creates an environment.
func (s *Store) CreateEnvironmentV2(ctx context.Context, create *EnvironmentMessage, creatorID int) (*EnvironmentMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	var uid int
	if err := tx.QueryRowContext(ctx, `
			INSERT INTO environment (
				resource_id,
				name,
				"order",
				creator_id,
				updater_id
			)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`,
		create.ResourceID,
		create.Title,
		create.Order,
		creatorID,
		creatorID,
	).Scan(
		&uid,
	); err != nil {
		return nil, FormatError(err)
	}

	value := api.EnvironmentTierValueUnprotected
	if create.Protected {
		value = api.EnvironmentTierValueProtected
	}
	payload, err := (&api.EnvironmentTierPolicy{EnvironmentTier: value}).String()
	if err != nil {
		return nil, err
	}
	if _, err := upsertPolicyImpl(ctx, tx, &api.PolicyUpsert{
		ResourceType: api.PolicyResourceTypeEnvironment,
		ResourceID:   uid,
		Type:         api.PolicyTypeEnvironmentTier,
		Payload:      &payload,
		UpdaterID:    creatorID,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	environment := &EnvironmentMessage{
		ResourceID: create.ResourceID,
		Title:      create.Title,
		Order:      create.Order,
		Protected:  create.Protected,
		UID:        uid,
		Deleted:    false,
	}
	s.environmentCache[environment.ResourceID] = environment
	s.environmentIDCache[environment.UID] = environment
	return environment, nil
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
	if v := patch.Delete; v != nil {
		rowStatus := api.Normal
		if *patch.Delete {
			rowStatus = api.Archived
		}
		set, args = append(set, fmt.Sprintf(`"row_status" = $%d`, len(args)+1)), append(args, rowStatus)
	}
	args = append(args, environmentID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	var environmentUID int
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
			UPDATE environment
			SET `+strings.Join(set, ", ")+`
			WHERE resource_id = $%d
			RETURNING id
		`, len(args)),
		args...,
	).Scan(
		&environmentUID,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, FormatError(err)
	}

	// TODO(d): consider moving tier to environment table to simplify things.
	// TODO(d): move policy interface to store v2.
	if patch.Protected != nil {
		value := api.EnvironmentTierValueUnprotected
		if *patch.Protected {
			value = api.EnvironmentTierValueProtected
		}
		payload, err := (&api.EnvironmentTierPolicy{EnvironmentTier: value}).String()
		if err != nil {
			return nil, err
		}
		if _, err := upsertPolicyImpl(ctx, tx, &api.PolicyUpsert{
			ResourceType: api.PolicyResourceTypeEnvironment,
			ResourceID:   environmentUID,
			Type:         api.PolicyTypeEnvironmentTier,
			Payload:      &payload,
			UpdaterID:    updaterID,
		}); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}
	// Invalid the cache and read the value again.
	delete(s.environmentCache, environmentID)
	delete(s.environmentIDCache, environmentUID)

	return s.GetEnvironmentV2(ctx, &FindEnvironmentMessage{
		ResourceID: &environmentID,
	})
}

func (*Store) getEnvironmentImplV2(ctx context.Context, tx *Tx, find *FindEnvironmentMessage) (*EnvironmentMessage, error) {
	environments, err := listEnvironmentImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	if len(environments) == 0 {
		return nil, nil
	}
	if len(environments) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d environments with resource ID %s, expect 1", len(environments), *find.ResourceID)}
	}
	return environments[0], nil
}

func listEnvironmentImplV2(ctx context.Context, tx *Tx, find *FindEnvironmentMessage) ([]*EnvironmentMessage, error) {
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("environment.resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("environment.id = $%d", len(args)+1)), append(args, *v)
	}
	if !find.ShowDeleted {
		where, args = append(where, fmt.Sprintf("environment.row_status = $%d", len(args)+1)), append(args, api.Normal)
	}

	var environments []*EnvironmentMessage
	rows, err := tx.QueryContext(ctx, `
		SELECT
			environment.id,
			environment.resource_id,
			environment.name,
			environment.order,
			environment.row_status,
			policy.payload
		FROM environment
		LEFT JOIN policy ON environment.id = policy.resource_id AND policy.resource_type = 'ENVIRONMENT' AND policy.type = 'bb.policy.environment-tier'
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var environment EnvironmentMessage
		var tierPayload sql.NullString
		var rowStatus string
		if err := rows.Scan(
			&environment.UID,
			&environment.ResourceID,
			&environment.Title,
			&environment.Order,
			&rowStatus,
			&tierPayload,
		); err != nil {
			return nil, FormatError(err)
		}
		environment.Deleted = convertRowStatusToDeleted(rowStatus)
		if tierPayload.Valid {
			policy, err := api.UnmarshalEnvironmentTierPolicy(tierPayload.String)
			if err != nil {
				return nil, err
			}
			environment.Protected = policy.EnvironmentTier == api.EnvironmentTierValueProtected
		}

		environments = append(environments, &environment)
	}
	return environments, nil
}

func convertRowStatusToDeleted(rowStatus string) bool {
	return rowStatus == string(api.Archived)
}
