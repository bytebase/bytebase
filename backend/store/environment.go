package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// EnvironmentMessage is the mssage for environment.
type EnvironmentMessage struct {
	ResourceID string
	Title      string
	Order      int32
	Protected  bool
	Color      string
	Deleted    bool
}

// FindEnvironmentMessage is the message to find environments.
type FindEnvironmentMessage struct {
	ResourceID  *string
	ShowDeleted bool
}

// UpdateEnvironmentMessage is the message for updating an environment.
type UpdateEnvironmentMessage struct {
	ResourceID string

	Name      *string
	Order     *int32
	Protected *bool
	Delete    *bool
	Color     *string
}

// GetEnvironmentV2 gets environment by resource ID.
func (s *Store) GetEnvironmentV2(ctx context.Context, find *FindEnvironmentMessage) (*EnvironmentMessage, error) {
	if find.ResourceID != nil {
		if v, ok := s.environmentCache.Get(*find.ResourceID); ok && s.enableCache {
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
	}
	return environments, nil
}

// CreateEnvironmentV2 creates an environment.
func (s *Store) CreateEnvironmentV2(ctx context.Context, create *EnvironmentMessage) (*EnvironmentMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
			INSERT INTO environment (
				resource_id,
				name,
				"order"
			)
			VALUES ($1, $2, $3)
		`,
		create.ResourceID,
		create.Title,
		create.Order,
	); err != nil {
		return nil, err
	}

	value := storepb.EnvironmentTierPolicy_UNPROTECTED
	if create.Protected {
		value = storepb.EnvironmentTierPolicy_PROTECTED
	}
	payload, err := protojson.Marshal(&storepb.EnvironmentTierPolicy{
		EnvironmentTier: value,
		Color:           create.Color,
	})
	if err != nil {
		return nil, err
	}
	if _, err := upsertPolicyV2Impl(ctx, tx, &PolicyMessage{
		ResourceType:      api.PolicyResourceTypeEnvironment,
		Resource:          common.FormatEnvironment(create.ResourceID),
		Type:              api.PolicyTypeEnvironmentTier,
		InheritFromParent: true,
		Payload:           string(payload),
		Enforce:           true,
	}); err != nil {
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
		Color:      create.Color,
		Deleted:    false,
	}
	s.environmentCache.Add(environment.ResourceID, environment)
	return environment, nil
}

// UpdateEnvironmentV2 updates an environment.
func (s *Store) UpdateEnvironmentV2(ctx context.Context, patch *UpdateEnvironmentMessage) (*EnvironmentMessage, error) {
	set, args := []string{}, []any{}
	if v := patch.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Order; v != nil {
		set, args = append(set, fmt.Sprintf(`"order" = $%d`, len(args)+1)), append(args, *v)
	}
	if v := patch.Delete; v != nil {
		set, args = append(set, fmt.Sprintf("deleted = $%d", len(args)+1)), append(args, *v)
	}
	args = append(args, patch.ResourceID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if len(set) > 0 {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
			UPDATE environment
			SET `+strings.Join(set, ", ")+`
			WHERE resource_id = $%d
		`, len(args)),
			args...,
		); err != nil {
			if err == sql.ErrNoRows {
				return nil, nil
			}
			return nil, err
		}
	}

	// TODO(d): consider moving tier to environment table to simplify things.
	if patch.Protected != nil || patch.Color != nil {
		resourceType := api.PolicyResourceTypeEnvironment
		resource := common.FormatEnvironment(patch.ResourceID)
		policyType := api.PolicyTypeEnvironmentTier
		policy, err := s.GetPolicyV2(ctx, &FindPolicyMessage{
			ResourceType: &resourceType,
			Type:         &policyType,
			Resource:     &resource,
		})
		if err != nil {
			return nil, err
		}
		environmentPolicy := &storepb.EnvironmentTierPolicy{
			EnvironmentTier: storepb.EnvironmentTierPolicy_UNPROTECTED,
		}
		if policy != nil {
			if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(policy.Payload), environmentPolicy); err != nil {
				return nil, errors.Wrapf(err, "failed to unmarshal environment policy payload")
			}
		}
		if patch.Protected != nil {
			value := storepb.EnvironmentTierPolicy_UNPROTECTED
			if *patch.Protected {
				value = storepb.EnvironmentTierPolicy_PROTECTED
			}
			environmentPolicy.EnvironmentTier = value
		}
		if v := patch.Color; v != nil {
			environmentPolicy.Color = *v
		}
		payload, err := protojson.Marshal(environmentPolicy)
		if err != nil {
			return nil, err
		}
		if _, err := upsertPolicyV2Impl(ctx, tx, &PolicyMessage{
			ResourceType:      resourceType,
			Resource:          resource,
			Type:              policyType,
			InheritFromParent: true,
			Payload:           string(payload),
			Enforce:           true,
		}); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	// Invalid the cache and read the value again.
	s.environmentCache.Remove(patch.ResourceID)

	return s.GetEnvironmentV2(ctx, &FindEnvironmentMessage{
		ResourceID: &patch.ResourceID,
	})
}

func (*Store) getEnvironmentImplV2(ctx context.Context, txn *sql.Tx, find *FindEnvironmentMessage) (*EnvironmentMessage, error) {
	environments, err := listEnvironmentImplV2(ctx, txn, find)
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

func listEnvironmentImplV2(ctx context.Context, txn *sql.Tx, find *FindEnvironmentMessage) ([]*EnvironmentMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("environment.resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if !find.ShowDeleted {
		where, args = append(where, fmt.Sprintf("environment.deleted = $%d", len(args)+1)), append(args, false)
	}

	var environments []*EnvironmentMessage
	environmentResource := `format('environments/%s', environment.resource_id)`
	rows, err := txn.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			environment.resource_id,
			environment.name,
			environment.order,
			environment.deleted,
			policy.payload
		FROM environment
		LEFT JOIN policy ON %s = policy.resource AND policy.resource_type = 'ENVIRONMENT' AND policy.type = 'bb.policy.environment-tier'
		WHERE %s
		ORDER BY environment.order ASC`, environmentResource, strings.Join(where, " AND ")),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var environment EnvironmentMessage
		var tierPayload sql.NullString
		if err := rows.Scan(
			&environment.ResourceID,
			&environment.Title,
			&environment.Order,
			&environment.Deleted,
			&tierPayload,
		); err != nil {
			return nil, err
		}
		if tierPayload.Valid {
			policy := &storepb.EnvironmentTierPolicy{}
			if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(tierPayload.String), policy); err != nil {
				return nil, err
			}
			environment.Protected = policy.EnvironmentTier == storepb.EnvironmentTierPolicy_PROTECTED
			environment.Color = policy.Color
		}

		environments = append(environments, &environment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return environments, nil
}
