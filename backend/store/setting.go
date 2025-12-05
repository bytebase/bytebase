package store

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
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

func (s *Store) GetPasswordRestriction(ctx context.Context) (*storepb.PasswordRestrictionSetting, error) {
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

func (s *Store) GetSystemSetting(ctx context.Context) (*storepb.SystemSetting, error) {
	setting, err := s.GetSettingV2(ctx, storepb.SettingName_SYSTEM)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get setting %v", storepb.SettingName_SYSTEM)
	}
	if setting == nil {
		return nil, errors.Errorf("cannot find setting %v", storepb.SettingName_SYSTEM)
	}

	payload := new(storepb.SystemSetting)
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(setting.Value), payload); err != nil {
		return nil, err
	}
	return payload, nil
}

// GetWorkspaceID finds the workspace id in setting bb.workspace.id.
func (s *Store) GetWorkspaceID(ctx context.Context) (string, error) {
	systemSetting, err := s.GetSystemSetting(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to get system setting")
	}
	return systemSetting.WorkspaceId, nil
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

func (s *Store) GetEnvironment(ctx context.Context) (*storepb.EnvironmentSetting, error) {
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
	systemSetting, err := s.GetSystemSetting(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to get system setting")
	}
	s.Secret = systemSetting.AuthSecret
	return systemSetting.AuthSecret, nil
}

// UpsertSettingV2 upserts the setting by name.
func (s *Store) UpsertSettingV2(ctx context.Context, update *SetSettingMessage) (*SettingMessage, error) {
	q := qb.Q().Space(`
		INSERT INTO setting (name, value)
		VALUES (?, ?)
		ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
		RETURNING name, value
	`, update.Name.String(), update.Value)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

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

	q := qb.Q().Space(`
		INSERT INTO setting (name, value)
		VALUES (?, ?)
		RETURNING name, value
	`, create.Name.String(), create.Value)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to build sql")
	}

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
	q := qb.Q().Space("DELETE FROM setting WHERE name = ?", name.String())
	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	s.settingCache.Remove(name)
	return nil
}

func listSettingV2Impl(ctx context.Context, txn *sql.Tx, find *FindSettingMessage) ([]*SettingMessage, error) {
	q := qb.Q().Space(`
		SELECT
			name,
			value
		FROM setting
		WHERE TRUE
	`)
	if v := find.Name; v != nil {
		q.And("name = ?", v.String())
	}
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	rows, err := txn.QueryContext(ctx, query, args...)
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
