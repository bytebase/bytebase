package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// IdentityProviderMessage is the message for identity provider.
type IdentityProviderMessage struct {
	ResourceID string
	Title      string
	Domain     string
	Type       storepb.IdentityProviderType
	Config     *storepb.IdentityProviderConfig
}

func getConfigBytes(config *storepb.IdentityProviderConfig) ([]byte, error) {
	if v := config.GetOauth2Config(); v != nil {
		configBytes, err := protojson.Marshal(v)
		return configBytes, err
	} else if v := config.GetOidcConfig(); v != nil {
		configBytes, err := protojson.Marshal(v)
		return configBytes, err
	} else if v := config.GetLdapConfig(); v != nil {
		configBytes, err := protojson.Marshal(v)
		return configBytes, err
	}
	return nil, errors.Errorf("unexpected provider type")
}

// FindIdentityProviderMessage is the message for finding identity providers.
type FindIdentityProviderMessage struct {
	ResourceID *string
}

// UpdateIdentityProviderMessage is the message for updating an identity provider.
type UpdateIdentityProviderMessage struct {
	ResourceID string

	Title  *string
	Domain *string
	Config *storepb.IdentityProviderConfig
}

// CreateIdentityProvider creates an identity provider.
func (s *Store) CreateIdentityProvider(ctx context.Context, create *IdentityProviderMessage) (*IdentityProviderMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
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
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO idp (
			resource_id,
			name,
			domain,
			type,
			config
		)
		VALUES ($1, $2, $3, $4, $5)
		`,
		create.ResourceID,
		create.Title,
		create.Domain,
		create.Type.String(),
		configBytes,
	); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.idpCache.Add(identityProvider.ResourceID, identityProvider)
	return identityProvider, nil
}

// GetIdentityProvider gets an identity provider.
func (s *Store) GetIdentityProvider(ctx context.Context, find *FindIdentityProviderMessage) (*IdentityProviderMessage, error) {
	if find.ResourceID != nil {
		if v, ok := s.idpCache.Get(*find.ResourceID); ok && s.enableCache {
			return v, nil
		}
	}

	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	identityProviders, err := s.listIdentityProvidersImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if len(identityProviders) == 0 {
		return nil, nil
	} else if len(identityProviders) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d identity providers with filter %+v, expect 1", len(identityProviders), find)}
	}

	identityProvider := identityProviders[0]
	s.idpCache.Add(identityProvider.ResourceID, identityProvider)
	return identityProvider, nil
}

// ListIdentityProviders lists identity providers.
func (s *Store) ListIdentityProviders(ctx context.Context, find *FindIdentityProviderMessage) ([]*IdentityProviderMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	identityProviders, err := s.listIdentityProvidersImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	for _, identityProvider := range identityProviders {
		s.idpCache.Add(identityProvider.ResourceID, identityProvider)
	}
	return identityProviders, nil
}

// UpdateIdentityProvider updates an identity provider.
func (s *Store) UpdateIdentityProvider(ctx context.Context, patch *UpdateIdentityProviderMessage) (*IdentityProviderMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	identityProvider, err := s.updateIdentityProviderImpl(ctx, tx, patch)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.idpCache.Add(identityProvider.ResourceID, identityProvider)
	return identityProvider, nil
}

func (*Store) updateIdentityProviderImpl(ctx context.Context, txn *sql.Tx, patch *UpdateIdentityProviderMessage) (*IdentityProviderMessage, error) {
	set, args := []string{}, []any{}
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
	args = append(args, patch.ResourceID)

	identityProvider := &IdentityProviderMessage{}
	var identityProviderType string
	var identityProviderConfig string
	if err := txn.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE idp
		SET `+strings.Join(set, ", ")+`
		WHERE resource_id = $%d
		RETURNING
			resource_id,
			name,
			domain,
			type,
			config
	`, len(args)),
		args...,
	).Scan(
		&identityProvider.ResourceID,
		&identityProvider.Title,
		&identityProvider.Domain,
		&identityProviderType,
		&identityProviderConfig,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("identity provider ID not found: %s", patch.ResourceID)}
		}
		return nil, err
	}

	identityProvider.Type = convertIdentityProviderType(identityProviderType)
	identityProvider.Config = convertIdentityProviderConfigString(identityProvider.Type, identityProviderConfig)
	return identityProvider, nil
}

func (*Store) listIdentityProvidersImpl(ctx context.Context, txn *sql.Tx, find *FindIdentityProviderMessage) ([]*IdentityProviderMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("resource_id = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := txn.QueryContext(ctx, `
		SELECT
			resource_id,
			name,
			domain,
			type,
			config
		FROM idp
		WHERE `+strings.Join(where, " AND ")+` ORDER BY id ASC`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var identityProviderMessages []*IdentityProviderMessage
	for rows.Next() {
		var identityProviderMessage IdentityProviderMessage
		var identityProviderType string
		var identityProviderConfig string
		if err := rows.Scan(
			&identityProviderMessage.ResourceID,
			&identityProviderMessage.Title,
			&identityProviderMessage.Domain,
			&identityProviderType,
			&identityProviderConfig,
		); err != nil {
			return nil, err
		}
		identityProviderMessage.Type = convertIdentityProviderType(identityProviderType)
		identityProviderMessage.Config = convertIdentityProviderConfigString(identityProviderMessage.Type, identityProviderConfig)
		identityProviderMessages = append(identityProviderMessages, &identityProviderMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return identityProviderMessages, nil
}

func (s *Store) DeleteIdentityProvider(ctx context.Context, resourceID string) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM idp
		WHERE resource_id = $1
	`, resourceID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.idpCache.Remove(resourceID)
	return nil
}

func convertIdentityProviderType(identityProviderType string) storepb.IdentityProviderType {
	switch identityProviderType {
	case "OAUTH2":
		return storepb.IdentityProviderType_OAUTH2
	case "OIDC":
		return storepb.IdentityProviderType_OIDC
	case "LDAP":
		return storepb.IdentityProviderType_LDAP
	default:
		return storepb.IdentityProviderType_IDENTITY_PROVIDER_TYPE_UNSPECIFIED
	}
}

func convertIdentityProviderConfigString(identityProviderType storepb.IdentityProviderType, config string) *storepb.IdentityProviderConfig {
	identityProviderConfig := &storepb.IdentityProviderConfig{}
	switch identityProviderType {
	case storepb.IdentityProviderType_OAUTH2:
		var formattedConfig storepb.OAuth2IdentityProviderConfig
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(config), &formattedConfig); err != nil {
			return nil
		}
		identityProviderConfig.Config = &storepb.IdentityProviderConfig_Oauth2Config{
			Oauth2Config: &formattedConfig,
		}
	case storepb.IdentityProviderType_OIDC:
		var formattedConfig storepb.OIDCIdentityProviderConfig
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(config), &formattedConfig); err != nil {
			return nil
		}
		identityProviderConfig.Config = &storepb.IdentityProviderConfig_OidcConfig{
			OidcConfig: &formattedConfig,
		}
	case storepb.IdentityProviderType_LDAP:
		var formattedConfig storepb.LDAPIdentityProviderConfig
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(config), &formattedConfig); err != nil {
			return nil
		}
		identityProviderConfig.Config = &storepb.IdentityProviderConfig_LdapConfig{
			LdapConfig: &formattedConfig,
		}
	default:
		// Return nil for unknown identity provider types
		return nil
	}
	return identityProviderConfig
}
