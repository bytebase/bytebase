package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
)

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
		if v, ok := s.environmentCache.Get(*find.ResourceID); ok {
			return v, nil
		}
	}
	if find.UID != nil {
		if v, ok := s.environmentIDCache.Get(*find.UID); ok {
			return v, nil
		}
	}

	// We will always return the resource regardless of its deleted state.
	find.ShowDeleted = true

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
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
		return nil, err
	}

	s.environmentCache.Add(environment.ResourceID, environment)
	s.environmentIDCache.Add(environment.UID, environment)
	return environment, nil
}

// ListEnvironmentV2 lists all environment.
func (s *Store) ListEnvironmentV2(ctx context.Context, find *FindEnvironmentMessage) ([]*EnvironmentMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	environments, err := listEnvironmentImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	for _, environment := range environments {
		s.environmentCache.Add(environment.ResourceID, environment)
		s.environmentIDCache.Add(environment.UID, environment)
	}
	return environments, nil
}

// CreateEnvironmentV2 creates an environment.
func (s *Store) CreateEnvironmentV2(ctx context.Context, create *EnvironmentMessage, creatorID int) (*EnvironmentMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	value := api.EnvironmentTierValueUnprotected
	if create.Protected {
		value = api.EnvironmentTierValueProtected
	}
	payload, err := (&api.EnvironmentTierPolicy{EnvironmentTier: value}).String()
	if err != nil {
		return nil, err
	}
	if _, err := upsertPolicyV2Impl(ctx, tx, &PolicyMessage{
		ResourceType:      api.PolicyResourceTypeEnvironment,
		ResourceUID:       uid,
		Type:              api.PolicyTypeEnvironmentTier,
		InheritFromParent: true,
		Payload:           payload,
		Enforce:           true,
	}, creatorID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	environment := &EnvironmentMessage{
		ResourceID: create.ResourceID,
		Title:      create.Title,
		Order:      create.Order,
		Protected:  create.Protected,
		UID:        uid,
		Deleted:    false,
	}
	s.environmentCache.Add(environment.ResourceID, environment)
	s.environmentIDCache.Add(environment.UID, environment)
	return environment, nil
}

// UpdateEnvironmentV2 updates an environment.
func (s *Store) UpdateEnvironmentV2(ctx context.Context, environmentID string, patch *UpdateEnvironmentMessage, updaterID int) (*EnvironmentMessage, error) {
	set, args := []string{"updater_id = $1"}, []any{fmt.Sprintf("%d", updaterID)}
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
		return nil, err
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
		return nil, err
	}

	// TODO(d): consider moving tier to environment table to simplify things.
	if patch.Protected != nil {
		value := api.EnvironmentTierValueUnprotected
		if *patch.Protected {
			value = api.EnvironmentTierValueProtected
		}
		payload, err := (&api.EnvironmentTierPolicy{EnvironmentTier: value}).String()
		if err != nil {
			return nil, err
		}
		if _, err := upsertPolicyV2Impl(ctx, tx, &PolicyMessage{
			ResourceType:      api.PolicyResourceTypeEnvironment,
			ResourceUID:       environmentUID,
			Type:              api.PolicyTypeEnvironmentTier,
			InheritFromParent: true,
			Payload:           payload,
			Enforce:           true,
		}, updaterID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	// Invalid the cache and read the value again.
	s.environmentCache.Remove(environmentID)
	s.environmentIDCache.Remove(environmentUID)

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
	where, args := []string{"TRUE"}, []any{}
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
	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			environment.id,
			environment.resource_id,
			environment.name,
			environment.order,
			environment.row_status,
			policy.payload
		FROM environment
		LEFT JOIN policy ON environment.id = policy.resource_id AND policy.resource_type = 'ENVIRONMENT' AND policy.type = 'bb.policy.environment-tier'
		WHERE %s
		ORDER BY environment.order ASC`, strings.Join(where, " AND ")),
		args...,
	)
	if err != nil {
		return nil, err
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
			return nil, err
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
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return environments, nil
}

func convertRowStatusToDeleted(rowStatus string) bool {
	return rowStatus == string(api.Archived)
}
