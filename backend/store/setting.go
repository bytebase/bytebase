package store

import (
	"context"
	"database/sql"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// SettingMessage is the message of setting.
type SettingMessage struct {
	Name  storepb.SettingName
	Value proto.Message
}

func getSettingMessage(name storepb.SettingName) (proto.Message, error) {
	switch name {
	case storepb.SettingName_WORKSPACE_PROFILE:
		return &storepb.WorkspaceProfileSetting{}, nil
	case storepb.SettingName_APP_IM:
		return &storepb.AppIMSetting{}, nil
	case storepb.SettingName_SYSTEM:
		return &storepb.SystemSetting{}, nil
	case storepb.SettingName_WORKSPACE_APPROVAL:
		return &storepb.WorkspaceApprovalSetting{}, nil
	case storepb.SettingName_SEMANTIC_TYPES:
		return &storepb.SemanticTypeSetting{}, nil
	case storepb.SettingName_DATA_CLASSIFICATION:
		return &storepb.DataClassificationSetting{}, nil
	case storepb.SettingName_AI:
		return &storepb.AISetting{}, nil
	case storepb.SettingName_ENVIRONMENT:
		return &storepb.EnvironmentSetting{}, nil
	default:
		return nil, errors.Errorf("unknown setting name: %v", name)
	}
}

// GetWorkspaceProfileSetting gets the workspace profile setting payload.
func (s *Store) GetWorkspaceProfileSetting(ctx context.Context) (*storepb.WorkspaceProfileSetting, error) {
	setting, err := s.GetSetting(ctx, storepb.SettingName_WORKSPACE_PROFILE)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get setting %v", storepb.SettingName_WORKSPACE_PROFILE)
	}
	if setting == nil {
		return nil, errors.Errorf("cannot find setting %v", storepb.SettingName_WORKSPACE_PROFILE)
	}

	val, ok := setting.Value.(*storepb.WorkspaceProfileSetting)
	if !ok {
		return nil, errors.Errorf("invalid setting value type for %s", storepb.SettingName_WORKSPACE_PROFILE)
	}
	return val, nil
}

func (s *Store) GetAppIMSetting(ctx context.Context) (*storepb.AppIMSetting, error) {
	setting, err := s.GetSetting(ctx, storepb.SettingName_APP_IM)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get setting %v", storepb.SettingName_APP_IM)
	}
	if setting == nil {
		return nil, errors.Errorf("cannot find setting %v", storepb.SettingName_APP_IM)
	}

	val, ok := setting.Value.(*storepb.AppIMSetting)
	if !ok {
		return nil, errors.Errorf("invalid setting value type for %s", storepb.SettingName_APP_IM)
	}
	return val, nil
}

func (s *Store) GetSystemSetting(ctx context.Context) (*storepb.SystemSetting, error) {
	setting, err := s.GetSetting(ctx, storepb.SettingName_SYSTEM)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get setting %v", storepb.SettingName_SYSTEM)
	}
	if setting == nil {
		return nil, errors.Errorf("cannot find setting %v", storepb.SettingName_SYSTEM)
	}

	val, ok := setting.Value.(*storepb.SystemSetting)
	if !ok {
		return nil, errors.Errorf("invalid setting value type for %s", storepb.SettingName_SYSTEM)
	}
	return val, nil
}

// UpdateLicense updates the license in SYSTEM setting.
func (s *Store) UpdateLicense(ctx context.Context, license string) error {
	setting, err := s.GetSetting(ctx, storepb.SettingName_SYSTEM)
	if err != nil {
		return errors.Wrap(err, "failed to get system setting")
	}
	if setting == nil {
		return errors.Errorf("system setting not found")
	}
	systemSetting, ok := setting.Value.(*storepb.SystemSetting)
	if !ok {
		return errors.Errorf("invalid system setting value type for %s", storepb.SettingName_SYSTEM)
	}

	systemSetting.License = license
	if _, err := s.UpsertSetting(ctx, &SettingMessage{
		Name:  storepb.SettingName_SYSTEM,
		Value: systemSetting,
	}); err != nil {
		return errors.Wrap(err, "failed to upsert system setting")
	}
	return nil
}

// GetWorkspaceApprovalSetting gets the workspace approval setting.
func (s *Store) GetWorkspaceApprovalSetting(ctx context.Context) (*storepb.WorkspaceApprovalSetting, error) {
	setting, err := s.GetSetting(ctx, storepb.SettingName_WORKSPACE_APPROVAL)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get setting %v", storepb.SettingName_WORKSPACE_APPROVAL)
	}
	if setting == nil {
		return nil, errors.Errorf("cannot find setting %v", storepb.SettingName_WORKSPACE_APPROVAL)
	}

	val, ok := setting.Value.(*storepb.WorkspaceApprovalSetting)
	if !ok {
		return nil, errors.Errorf("invalid setting value type for %s", storepb.SettingName_WORKSPACE_APPROVAL)
	}
	return val, nil
}

// GetSemanticTypesSetting gets the semantic types setting.
func (s *Store) GetSemanticTypesSetting(ctx context.Context) (*storepb.SemanticTypeSetting, error) {
	setting, err := s.GetSetting(ctx, storepb.SettingName_SEMANTIC_TYPES)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get setting %v", storepb.SettingName_SEMANTIC_TYPES)
	}
	if setting == nil {
		return &storepb.SemanticTypeSetting{}, nil
	}

	val, ok := setting.Value.(*storepb.SemanticTypeSetting)
	if !ok {
		return nil, errors.Errorf("invalid setting value type for %s", storepb.SettingName_SEMANTIC_TYPES)
	}
	return val, nil
}

// GetDataClassificationSetting gets the data classification setting.
func (s *Store) GetDataClassificationSetting(ctx context.Context) (*storepb.DataClassificationSetting, error) {
	setting, err := s.GetSetting(ctx, storepb.SettingName_DATA_CLASSIFICATION)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get setting %v", storepb.SettingName_DATA_CLASSIFICATION)
	}
	if setting == nil {
		return &storepb.DataClassificationSetting{}, nil
	}

	val, ok := setting.Value.(*storepb.DataClassificationSetting)
	if !ok {
		return nil, errors.Errorf("invalid setting value type for %s", storepb.SettingName_DATA_CLASSIFICATION)
	}
	return val, nil
}

func (s *Store) GetAISetting(ctx context.Context) (*storepb.AISetting, error) {
	setting, err := s.GetSetting(ctx, storepb.SettingName_AI)
	if err != nil {
		return nil, err
	}
	if setting == nil {
		return &storepb.AISetting{}, nil
	}

	val, ok := setting.Value.(*storepb.AISetting)
	if !ok {
		return nil, errors.Errorf("invalid setting value type for %s", storepb.SettingName_AI)
	}
	return val, nil
}

func (s *Store) GetEnvironment(ctx context.Context) (*storepb.EnvironmentSetting, error) {
	setting, err := s.GetSetting(ctx, storepb.SettingName_ENVIRONMENT)
	if err != nil {
		return nil, err
	}
	if setting == nil {
		return &storepb.EnvironmentSetting{}, nil
	}

	val, ok := setting.Value.(*storepb.EnvironmentSetting)
	if !ok {
		return nil, errors.Errorf("invalid setting value type for %s", storepb.SettingName_ENVIRONMENT)
	}
	return val, nil
}

// FindSettingMessage is the message for finding settings.
type FindSettingMessage struct {
	Name *storepb.SettingName
}

// GetSetting returns the setting by name.
func (s *Store) GetSetting(ctx context.Context, name storepb.SettingName) (*SettingMessage, error) {
	if v, ok := s.settingCache.Get(name); ok && s.enableCache {
		return v, nil
	}

	settings, err := s.ListSettings(ctx, &FindSettingMessage{Name: &name})
	if err != nil {
		return nil, err
	}
	if len(settings) == 0 {
		return nil, nil
	}
	if len(settings) > 1 {
		return nil, errors.Errorf("found multiple settings: %v", name)
	}

	return settings[0], nil
}

// ListSettings returns a list of settings.
func (s *Store) ListSettings(ctx context.Context, find *FindSettingMessage) ([]*SettingMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	q := qb.Q().Space(`
		SELECT
			name,
			value
		FROM setting
		WHERE TRUE
	`)
	if find != nil && find.Name != nil {
		q.And("name = ?", find.Name.String())
	}
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settingMessages []*SettingMessage
	for rows.Next() {
		var settingMessage SettingMessage
		var nameString string
		var valueString string
		if err := rows.Scan(
			&nameString,
			&valueString,
		); err != nil {
			return nil, err
		}
		value, ok := storepb.SettingName_value[nameString]
		if !ok {
			return nil, errors.Errorf("invalid setting name string: %s", nameString)
		}
		settingMessage.Name = storepb.SettingName(value)

		msg, err := getSettingMessage(settingMessage.Name)
		if err != nil {
			return nil, err
		}
		if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(valueString), msg); err != nil {
			return nil, err
		}
		settingMessage.Value = msg

		settingMessages = append(settingMessages, &settingMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}

	for _, setting := range settingMessages {
		s.settingCache.Add(setting.Name, setting)
	}
	return settingMessages, nil
}

// UpsertSetting upserts the setting by name.
func (s *Store) UpsertSetting(ctx context.Context, update *SettingMessage) (*SettingMessage, error) {
	valueBytes, err := protojson.Marshal(update.Value)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal setting value")
	}

	q := qb.Q().Space(`
		INSERT INTO setting (name, value)
		VALUES (?, ?)
		ON CONFLICT (name) DO UPDATE SET value = EXCLUDED.value
		RETURNING name, value
	`, update.Name.String(), string(valueBytes))

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
	var valueString string
	if err := tx.QueryRowContext(ctx, query, args...).Scan(
		&nameString,
		&valueString,
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

	msg, err := getSettingMessage(setting.Name)
	if err != nil {
		return nil, err
	}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(valueString), msg); err != nil {
		return nil, err
	}
	setting.Value = msg

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}
	s.settingCache.Add(setting.Name, &setting)
	return &setting, nil
}

// DeleteSetting deletes a setting by the name.
func (s *Store) DeleteSetting(ctx context.Context, name storepb.SettingName) error {
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
