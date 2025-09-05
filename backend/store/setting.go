package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// FindSettingMessage is the message for finding setting.
type FindSettingMessage struct {
	Name *storepb.SettingName
}

// SetSettingMessage is the message for updating setting.
type SetSettingMessage struct {
	Name  storepb.SettingName
	Value string
}

// SettingMessage is the message of setting.
type SettingMessage struct {
	Name  storepb.SettingName
	Value string
}

func (s *Store) GetPasswordRestrictionSetting(ctx context.Context) (*storepb.PasswordRestrictionSetting, error) {
	passwordRestriction := &storepb.PasswordRestrictionSetting{
		MinLength: 8,
	}
	setting, err := s.GetSettingV2(ctx, storepb.SettingName_PASSWORD_RESTRICTION)
	if err != nil {
		return nil, err
	}
	if setting == nil {
		return passwordRestriction, nil
	}

	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), passwordRestriction); err != nil {
		return nil, err
	}
	return passwordRestriction, nil
}

// GetWorkspaceGeneralSetting gets the workspace general setting payload.
func (s *Store) GetWorkspaceGeneralSetting(ctx context.Context) (*storepb.WorkspaceProfileSetting, error) {
	setting, err := s.GetSettingV2(ctx, storepb.SettingName_WORKSPACE_PROFILE)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get setting %v", storepb.SettingName_WORKSPACE_PROFILE)
	}
	if setting == nil {
		return nil, errors.Errorf("cannot find setting %v", storepb.SettingName_WORKSPACE_PROFILE)
	}

	payload := new(storepb.WorkspaceProfileSetting)
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *Store) GetAppIMSetting(ctx context.Context) (*storepb.AppIMSetting, error) {
	setting, err := s.GetSettingV2(ctx, storepb.SettingName_APP_IM)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get setting %v", storepb.SettingName_APP_IM)
	}
	if setting == nil {
		return nil, errors.Errorf("cannot find setting %v", storepb.SettingName_APP_IM)
	}

	payload := new(storepb.AppIMSetting)
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// GetWorkspaceID finds the workspace id in setting bb.workspace.id.
func (s *Store) GetWorkspaceID(ctx context.Context) (string, error) {
	setting, err := s.GetSettingV2(ctx, storepb.SettingName_WORKSPACE_ID)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get setting %v", storepb.SettingName_WORKSPACE_ID)
	}
	if setting == nil {
		return "", errors.Errorf("cannot find setting %v", storepb.SettingName_WORKSPACE_ID)
	}
	return setting.Value, nil
}

// GetWorkspaceApprovalSetting gets the workspace approval setting.
func (s *Store) GetWorkspaceApprovalSetting(ctx context.Context) (*storepb.WorkspaceApprovalSetting, error) {
	setting, err := s.GetSettingV2(ctx, storepb.SettingName_WORKSPACE_APPROVAL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get setting %v", storepb.SettingName_WORKSPACE_APPROVAL)
	}
	if setting == nil {
		return nil, errors.Errorf("cannot find setting %v", storepb.SettingName_WORKSPACE_APPROVAL)
	}

	payload := new(storepb.WorkspaceApprovalSetting)
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// GetSemanticTypesSetting gets the semantic types setting.
func (s *Store) GetSemanticTypesSetting(ctx context.Context) (*storepb.SemanticTypeSetting, error) {
	setting, err := s.GetSettingV2(ctx, storepb.SettingName_SEMANTIC_TYPES)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get setting %v", storepb.SettingName_SEMANTIC_TYPES)
	}
	if setting == nil {
		return &storepb.SemanticTypeSetting{}, nil
	}

	payload := new(storepb.SemanticTypeSetting)
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// GetDataClassificationSetting gets the data classification setting.
func (s *Store) GetDataClassificationSetting(ctx context.Context) (*storepb.DataClassificationSetting, error) {
	setting, err := s.GetSettingV2(ctx, storepb.SettingName_DATA_CLASSIFICATION)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get setting %v", storepb.SettingName_DATA_CLASSIFICATION)
	}
	if setting == nil {
		return &storepb.DataClassificationSetting{}, nil
	}

	payload := new(storepb.DataClassificationSetting)
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// GetDataClassificationConfigByID gets the classification config by the id.
func (s *Store) GetDataClassificationConfigByID(ctx context.Context, classificationConfigID string) (*storepb.DataClassificationSetting_DataClassificationConfig, error) {
	setting, err := s.GetDataClassificationSetting(ctx)
	if err != nil {
		return nil, err
	}
	for _, config := range setting.Configs {
		if config.Id == classificationConfigID {
			return config, nil
		}
	}
	return &storepb.DataClassificationSetting_DataClassificationConfig{}, nil
}

func (s *Store) GetAISetting(ctx context.Context) (*storepb.AISetting, error) {
	aiSetting := &storepb.AISetting{}
	setting, err := s.GetSettingV2(ctx, storepb.SettingName_AI)
	if err != nil {
		return nil, err
	}
	if setting == nil {
		return aiSetting, nil
	}

	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), aiSetting); err != nil {
		return nil, err
	}
	return aiSetting, nil
}

func (s *Store) GetEnvironmentSetting(ctx context.Context) (*storepb.EnvironmentSetting, error) {
	envSetting := &storepb.EnvironmentSetting{}
	setting, err := s.GetSettingV2(ctx, storepb.SettingName_ENVIRONMENT)
	if err != nil {
		return nil, err
	}
	if setting == nil {
		return envSetting, nil
	}

	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), envSetting); err != nil {
		return nil, err
	}
	return envSetting, nil
}

// DeleteCache deletes the cache.
func (s *Store) DeleteCache() {
	s.settingCache.Purge()
	s.policyCache.Purge()
	s.userEmailCache.Purge()
	s.userIDCache.Purge()
}

// GetSettingV2 returns the setting by name.
func (s *Store) GetSettingV2(ctx context.Context, name storepb.SettingName) (*SettingMessage, error) {
	if v, ok := s.settingCache.Get(name); ok && s.enableCache {
		return v, nil
	}

	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	settings, err := listSettingV2Impl(ctx, tx, &FindSettingMessage{
		Name: &name,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list setting")
	}
	if len(settings) == 0 {
		return nil, nil
	}
	if len(settings) > 1 {
		return nil, errors.Errorf("found multiple settings: %v", name)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}
	return settings[0], nil
}

// ListSettingV2 returns a list of settings.
func (s *Store) ListSettingV2(ctx context.Context, find *FindSettingMessage) ([]*SettingMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()
	settings, err := listSettingV2Impl(ctx, tx, find)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list setting")
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}

	for _, setting := range settings {
		s.settingCache.Add(setting.Name, setting)
	}
	return settings, nil
}

func (s *Store) GetSecret(ctx context.Context) (string, error) {
	if s.Secret != "" {
		return s.Secret, nil
	}
	setting, err := s.GetSettingV2(ctx, storepb.SettingName_AUTH_SECRET)
	if err != nil {
		return "", err
	}
	if setting == nil {
		return "", errors.New("auth secret not found")
	}
	s.Secret = setting.Value
	return setting.Value, nil
}

// UpsertSettingV2 upserts the setting by name.
func (s *Store) UpsertSettingV2(ctx context.Context, update *SetSettingMessage) (*SettingMessage, error) {
	fields := []string{"name", "value"}
	updateFields := []string{"value = EXCLUDED.value"}
	valuePlaceholders, args := []string{"$1", "$2"}, []any{update.Name.String(), update.Value}

	query := `INSERT INTO setting (` + strings.Join(fields, ", ") + `) 
		VALUES (` + strings.Join(valuePlaceholders, ", ") + `) 
		ON CONFLICT (name) DO UPDATE SET ` + strings.Join(updateFields, ", ") + `
		RETURNING name, value`

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	var setting SettingMessage
	var nameString string
	if err := tx.QueryRowContext(ctx, query, args...).Scan(
		&nameString,
		&setting.Value,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("setting not found: %s", update.Name)}
		}
		return nil, err
	}
	value, ok := storepb.SettingName_value[nameString]
	if !ok {
		return nil, errors.Errorf("invalid setting name string: %s", nameString)
	}
	setting.Name = storepb.SettingName(value)

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}
	s.settingCache.Add(setting.Name, &setting)
	return &setting, nil
}

// CreateSettingIfNotExistV2 creates a new setting only if the named setting doesn't exist.
func (s *Store) CreateSettingIfNotExistV2(ctx context.Context, create *SettingMessage) (*SettingMessage, bool, error) {
	if v, ok := s.settingCache.Get(create.Name); ok && s.enableCache {
		return v, false, nil
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()
	settings, err := listSettingV2Impl(ctx, tx, &FindSettingMessage{Name: &create.Name})
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to list settings")
	}
	if len(settings) > 1 {
		return nil, false, errors.Errorf("found settings for setting name: %v", create.Name)
	}
	if len(settings) == 1 {
		// Don't create setting if the named setting already exists.
		return settings[0], false, nil
	}

	fields := []string{"name", "value"}
	valuesPlaceholders, args := []string{"$1", "$2"}, []any{create.Name.String(), create.Value}

	query := `INSERT INTO setting (` + strings.Join(fields, ",") + `)
		VALUES (` + strings.Join(valuesPlaceholders, ",") + `)
		RETURNING name, value`
	var setting SettingMessage
	var nameString string
	if err := tx.QueryRowContext(ctx, query, args...).Scan(
		&nameString,
		&setting.Value,
	); err != nil {
		return nil, false, err
	}
	value, ok := storepb.SettingName_value[nameString]
	if !ok {
		return nil, false, errors.Errorf("invalid setting name string: %s", nameString)
	}
	setting.Name = storepb.SettingName(value)

	if err := tx.Commit(); err != nil {
		return nil, false, errors.Wrap(err, "failed to commit transaction")
	}
	s.settingCache.Add(setting.Name, &setting)
	return &setting, true, nil
}

// DeleteSettingV2 deletes a setting by the name.
func (s *Store) DeleteSettingV2(ctx context.Context, name storepb.SettingName) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM setting WHERE name = $1`, name.String()); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	s.settingCache.Remove(name)
	return nil
}

func listSettingV2Impl(ctx context.Context, txn *sql.Tx, find *FindSettingMessage) ([]*SettingMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.Name; v != nil {
		where, args = append(where, fmt.Sprintf("name = $%d", len(args)+1)), append(args, v.String())
	}
	rows, err := txn.QueryContext(ctx, `
		SELECT
			name,
			value
		FROM setting
		WHERE `+strings.Join(where, " AND "), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settingMessages []*SettingMessage
	for rows.Next() {
		var settingMessage SettingMessage
		var nameString string
		if err := rows.Scan(
			&nameString,
			&settingMessage.Value,
		); err != nil {
			return nil, err
		}
		value, ok := storepb.SettingName_value[nameString]
		if !ok {
			return nil, errors.Errorf("invalid setting name string: %s", nameString)
		}
		settingMessage.Name = storepb.SettingName(value)
		settingMessages = append(settingMessages, &settingMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return settingMessages, nil
}
