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

// IdentityProviderMessage is the message for identity provider.
type IdentityProviderMessage struct {
	ResourceID string
	Title      string
	Domain     string
	Type       storepb.IdentityProviderType
	Config     *storepb.IdentityProviderConfig
	// The following fields are output only and not used for creating.
	UID     int
	Deleted bool
}

func getConfigBytes(config *storepb.IdentityProviderConfig) ([]byte, error) {
	if v := config.GetOauth2Config(); v != nil {
		configBytes, err := protojson.Marshal(v)
		return configBytes, err
	} else if v := config.GetOidcConfig(); v != nil {
		configBytes, err := protojson.Marshal(v)
		return configBytes, err
	} else {
		return nil, errors.Errorf("unexpected provider type")
	}
}

// FindIdentityProviderMessage is the message for finding identity providers.
type FindIdentityProviderMessage struct {
	// We should only set either UID or ResourceID.
	// Deprecate UID later once we fully migrate to ResourceID.
	UID         *int
	ResourceID  *string
	ShowDeleted bool
}

// UpdateIdentityProviderMessage is the message for updating an identity provider.
type UpdateIdentityProviderMessage struct {
	UpdaterID  int
	ResourceID string

	Title  *string
	Domain *string
	Config *storepb.IdentityProviderConfig
	Delete *bool
}

// CreateIdentityProvider creates an identity provider.
func (s *Store) CreateIdentityProvider(ctx context.Context, create *IdentityProviderMessage, creatorID int) (*IdentityProviderMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	identityProvider := &IdentityProviderMessage{
		ResourceID: create.ResourceID,
		Title:      create.Title,
		Domain:     create.Domain,
		Type:       create.Type,
		Config:     create.Config,
	}
	configBytes, err := getConfigBytes(identityProvider.Config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal identity provider config")
	}
	if err := tx.QueryRowContext(ctx, `
			INSERT INTO idp (
				creator_id,
				updater_id,
				resource_id,
				name,
				domain,
				type,
				config
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id
		`,
		creatorID,
		creatorID,
		create.ResourceID,
		create.Title,
		create.Domain,
		create.Type.String(),
		configBytes,
	).Scan(
		&identityProvider.UID,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery("failed to create identity provider")
		}
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return identityProvider, nil
}

// GetIdentityProvider gets an identity provider.
func (s *Store) GetIdentityProvider(ctx context.Context, find *FindIdentityProviderMessage) (*IdentityProviderMessage, error) {
	// We will always return the resource regardless of its deleted state.
	find.ShowDeleted = true

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	identityProviders, err := s.listIdentityProvidersImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	if len(identityProviders) == 0 {
		return nil, nil
	} else if len(identityProviders) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d identity providers with filter %+v, expect 1", len(identityProviders), find)}
	}
	return identityProviders[0], nil
}

// ListIdentityProviders lists identity providers.
func (s *Store) ListIdentityProviders(ctx context.Context, find *FindIdentityProviderMessage) ([]*IdentityProviderMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	identityProviders, err := s.listIdentityProvidersImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return identityProviders, nil
}

// UpdateIdentityProvider updates an identity provider.
func (s *Store) UpdateIdentityProvider(ctx context.Context, patch *UpdateIdentityProviderMessage) (*IdentityProviderMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.Rollback()

	identityProvider, err := s.updateIdentityProviderImpl(ctx, tx, patch)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return identityProvider, nil
}

func (*Store) updateIdentityProviderImpl(ctx context.Context, tx *Tx, patch *UpdateIdentityProviderMessage) (*IdentityProviderMessage, error) {
	set, args := []string{"updater_id = $1"}, []interface{}{fmt.Sprintf("%d", patch.UpdaterID)}
	if v := patch.Title; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Domain; v != nil {
		set, args = append(set, fmt.Sprintf("domain = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Config; v != nil {
		configBytes, err := getConfigBytes(v)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal identity provider config")
		}
		set, args = append(set, fmt.Sprintf("config = $%d", len(args)+1)), append(args, string(configBytes))
	}
	if v := patch.Delete; v != nil {
		rowStatus := api.Normal
		if *patch.Delete {
			rowStatus = api.Archived
		}
		set, args = append(set, fmt.Sprintf(`"row_status" = $%d`, len(args)+1)), append(args, rowStatus)
	}
	args = append(args, patch.ResourceID)

	identityProvider := &IdentityProviderMessage{}
	var identityProviderType string
	var identityProviderConfig string
	var rowStatus string
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE idp
		SET `+strings.Join(set, ", ")+`
		WHERE resource_id = $%d
		RETURNING
			id,
			resource_id,
			name,
			domain,
			type,
			config,
			row_status
	`, len(args)),
		args...,
	).Scan(
		&identityProvider.UID,
		&identityProvider.ResourceID,
		&identityProvider.Title,
		&identityProvider.Domain,
		&identityProviderType,
		&identityProviderConfig,
		&rowStatus,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("identity provider ID not found: %s", patch.ResourceID)}
		}
		return nil, FormatError(err)
	}

	identityProvider.Type = convertIdentityProviderType(identityProviderType)
	identityProvider.Config = convertIdentityProviderConfigString(identityProvider.Type, identityProviderConfig)
	identityProvider.Deleted = convertRowStatusToDeleted(rowStatus)
	return identityProvider, nil
}

func (*Store) listIdentityProvidersImpl(ctx context.Context, tx *Tx, find *FindIdentityProviderMessage) ([]*IdentityProviderMessage, error) {
	where, args := []string{"TRUE"}, []interface{}{}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if !find.ShowDeleted {
		where, args = append(where, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, api.Normal)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			resource_id,
			name,
			domain,
			type,
			config,
			row_status
		FROM idp
		WHERE `+strings.Join(where, " AND ")+` ORDER BY id ASC`,
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	var identityProviderMessages []*IdentityProviderMessage
	for rows.Next() {
		var identityProviderMessage IdentityProviderMessage
		var identityProviderType string
		var identityProviderConfig string
		var rowStatus string
		if err := rows.Scan(
			&identityProviderMessage.UID,
			&identityProviderMessage.ResourceID,
			&identityProviderMessage.Title,
			&identityProviderMessage.Domain,
			&identityProviderType,
			&identityProviderConfig,
			&rowStatus,
		); err != nil {
			return nil, FormatError(err)
		}
		identityProviderMessage.Type = convertIdentityProviderType(identityProviderType)
		identityProviderMessage.Config = convertIdentityProviderConfigString(identityProviderMessage.Type, identityProviderConfig)
		identityProviderMessage.Deleted = convertRowStatusToDeleted(rowStatus)
		identityProviderMessages = append(identityProviderMessages, &identityProviderMessage)
	}

	return identityProviderMessages, nil
}

func convertIdentityProviderType(identityProviderType string) storepb.IdentityProviderType {
	if identityProviderType == "OAUTH2" {
		return storepb.IdentityProviderType_OAUTH2
	} else if identityProviderType == "OIDC" {
		return storepb.IdentityProviderType_OIDC
	}
	return storepb.IdentityProviderType_IDENTITY_PROVIDER_TYPE_UNSPECIFIED
}

func convertIdentityProviderConfigString(identityProviderType storepb.IdentityProviderType, config string) *storepb.IdentityProviderConfig {
	identityProviderConfig := &storepb.IdentityProviderConfig{}
	if identityProviderType == storepb.IdentityProviderType_OAUTH2 {
		var formatedConfig storepb.OAuth2IdentityProviderConfig
		decoder := protojson.UnmarshalOptions{DiscardUnknown: true}
		if err := decoder.Unmarshal([]byte(config), &formatedConfig); err != nil {
			return nil
		}
		identityProviderConfig.Config = &storepb.IdentityProviderConfig_Oauth2Config{
			Oauth2Config: &formatedConfig,
		}
	} else if identityProviderType == storepb.IdentityProviderType_OIDC {
		var formatedConfig storepb.OIDCIdentityProviderConfig
		decoder := protojson.UnmarshalOptions{DiscardUnknown: true}
		if err := decoder.Unmarshal([]byte(config), &formatedConfig); err != nil {
			return nil
		}
		identityProviderConfig.Config = &storepb.IdentityProviderConfig_OidcConfig{
			OidcConfig: &formatedConfig,
		}
	}
	return identityProviderConfig
}
