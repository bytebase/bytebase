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

// CreateSettingIfNotExist creates an instance of Setting.
func (s *Store) CreateSettingIfNotExist(ctx context.Context, create *api.SettingCreate) (*api.Setting, bool, error) {
	setting, created, err := s.createSettingRawIfNotExist(ctx, create)
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to create Setting with SettingCreate[%+v]", create)
	}
	s.settingCache.Store(setting.Name, setting)
	return setting.toAPISetting(), created, nil
}

// FindSetting finds a list of Setting instances.
func (s *Store) FindSetting(ctx context.Context, find *api.SettingFind) ([]*api.Setting, error) {
	findV2 := &FindSettingMessage{Name: find.Name}
	settings, err := s.ListSettingV2(ctx, findV2)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list setting with [%+v]", findV2)
	}
	var settingList []*api.Setting
	for _, setting := range settings {
		settingList = append(settingList, setting.toAPISetting())
	}
	return settingList, nil
}

// GetSetting gets an instance of Setting.
func (s *Store) GetSetting(ctx context.Context, find *api.SettingFind) (*api.Setting, error) {
	findV2 := &FindSettingMessage{Name: find.Name}
	setting, err := s.GetSettingV2(ctx, findV2)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get setting with [%+v]", findV2)
	}
	if setting == nil {
		return nil, nil
	}
	return setting.toAPISetting(), nil
}

// PatchSetting patches an instance of Setting.
func (s *Store) PatchSetting(ctx context.Context, patch *api.SettingPatch) (*api.Setting, error) {
	setting, err := s.UpsertSettingV2(ctx, &SetSettingMessage{
		Name:  patch.Name,
		Value: patch.Value,
	}, patch.UpdaterID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch setting with [%+v]", patch)
	}
	return setting.toAPISetting(), nil
}

//
// private functions
//

// createSettingRawIfNotExist creates a new setting only if the named setting does not exist.
// The returned bool means the resource is created successfully.
func (s *Store) createSettingRawIfNotExist(ctx context.Context, create *api.SettingCreate) (*SettingMessage, bool, error) {
	// We do a find followed by a create if NOT found. Though SQLite supports UPSERT ON CONFLICT DO NOTHING syntax, it doesn't
	// support RETURNING in such case. So we have to use separate SELECT and INSERT anyway.
	find := &FindSettingMessage{
		Name: &create.Name,
	}
	setting, err := s.GetSettingV2(ctx, find)
	if err != nil {
		return nil, false, err
	}
	if setting == nil {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return nil, false, FormatError(err)
		}
		defer tx.Rollback()

		setting, err := createSettingImpl(ctx, tx, create)
		if err != nil {
			return nil, false, err
		}

		if err := tx.Commit(); err != nil {
			return nil, false, FormatError(err)
		}

		return setting, true, nil
	}

	return setting, false, nil
}

// createSettingImpl creates a new setting.
func createSettingImpl(ctx context.Context, tx *Tx, create *api.SettingCreate) (*SettingMessage, error) {
	// Insert row into database.
	query := `
		INSERT INTO setting (
			creator_id,
			updater_id,
			name,
			value,
			description
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING name, value, description
	`
	var settingMessage SettingMessage
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.CreatorID,
		create.Name,
		create.Value,
		create.Description,
	).Scan(
		&settingMessage.Name,
		&settingMessage.Value,
		&settingMessage.Description,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, FormatError(err)
	}
	return &settingMessage, nil
}

// FindSettingMessage is the message for finding setting.
type FindSettingMessage struct {
	Name *api.SettingName
}

// SetSettingMessage is the message for updating setting.
type SetSettingMessage struct {
	Name        api.SettingName
	Value       string
	Description *string
}

// SettingMessage is the message of setting.
type SettingMessage struct {
	Name        api.SettingName
	Value       string
	Description string
}

func (sm *SettingMessage) toAPISetting() *api.Setting {
	return &api.Setting{
		Name:        sm.Name,
		Value:       sm.Value,
		Description: sm.Description,
	}
}

// GetSettingV2 returns the setting by name.
func (s *Store) GetSettingV2(ctx context.Context, find *FindSettingMessage) (*SettingMessage, error) {
	if find.Name != nil {
		if setting, ok := s.settingCache.Load(*find.Name); ok {
			return setting.(*SettingMessage), nil
		}
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	settings, err := listSettingV2Impl(ctx, tx, find)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list setting")
	}
	if len(settings) == 0 {
		return nil, nil
	}
	if len(settings) > 1 {
		return nil, errors.Errorf("found multiple settings: %v", find.Name)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}
	return settings[0], nil
}

// ListSettingV2 returns a list of settings.
func (s *Store) ListSettingV2(ctx context.Context, find *FindSettingMessage) ([]*SettingMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
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
		s.settingCache.Store(setting.Name, setting)
	}
	return settings, nil
}

// UpsertSettingV2 upserts the setting by name.
func (s *Store) UpsertSettingV2(ctx context.Context, update *SetSettingMessage, principalUID int) (*SettingMessage, error) {
	fields := []string{"creator_id", "updater_id", "name", "value"}
	updateFields := []string{"value = EXCLUDED.value", "updater_id = EXCLUDED.updater_id"}
	valuePlaceholders, args := []string{"$1", "$2", "$3", "$4"}, []interface{}{principalUID, principalUID, update.Name, update.Value}

	if v := update.Description; v != nil {
		fields = append(fields, "description")
		valuePlaceholders = append(valuePlaceholders, fmt.Sprintf("$%d", len(args)+1))
		updateFields = append(updateFields, "description = EXCLUDED.description")
		args = append(args, *v)
	}
	query := `INSERT INTO setting (` + strings.Join(fields, ", ") + `) 
		VALUES (` + strings.Join(valuePlaceholders, ", ") + `) 
		ON CONFLICT (name) DO UPDATE SET ` + strings.Join(updateFields, ", ") + `
		RETURNING name, value, description`

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	var setting SettingMessage
	if err := tx.QueryRowContext(ctx, query, args...).Scan(
		&setting.Name,
		&setting.Value,
		&setting.Description,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("setting not found: %s", update.Name)}
		}
		return nil, FormatError(err)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}
	s.settingCache.Store(setting.Name, &setting)
	return &setting, nil
}

func listSettingV2Impl(ctx context.Context, tx *Tx, find *FindSettingMessage) ([]*SettingMessage, error) {
	where, args := []string{"TRUE"}, []interface{}{}
	if v := find.Name; v != nil {
		where, args = append(where, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT
			name,
			value,
			description
		FROM setting
		WHERE `+strings.Join(where, " AND "), args...)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	var settingMessages []*SettingMessage
	for rows.Next() {
		var settingMessage SettingMessage
		if err := rows.Scan(
			&settingMessage.Name,
			&settingMessage.Value,
			&settingMessage.Description,
		); err != nil {
			return nil, FormatError(err)
		}
		settingMessages = append(settingMessages, &settingMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return settingMessages, nil
}
