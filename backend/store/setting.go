package store

import (
	"context"
	"database/sql"
	"math"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// SettingMessage is the message of setting.
type SettingMessage struct {
	Name      storepb.SettingName
	Workspace string
	Value     proto.Message
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
	case storepb.SettingName_EMAIL:
		return &storepb.EmailSetting{}, nil
	default:
		return nil, errors.Errorf("unknown setting name: %v", name)
	}
}

// GetSQLResultSize gets the valid data_export_result_size from the workspace profile setting.
func (s *Store) GetSQLResultSize(ctx context.Context, workspaceID string) (int64, error) {
	workspaceProfile, err := s.GetWorkspaceProfileSetting(ctx, workspaceID)
	if err != nil {
		return 0, err
	}
	maximumResultSize := workspaceProfile.GetSqlResultSize()
	if maximumResultSize <= 0 {
		maximumResultSize = common.DefaultMaximumSQLResultSize
	}
	return maximumResultSize, nil
}

// GetQueryTimeoutInSeconds gets the valid query_timeout from the workspace profile setting.
func (s *Store) GetQueryTimeoutInSeconds(ctx context.Context, workspaceID string) (int64, error) {
	workspaceProfile, err := s.GetWorkspaceProfileSetting(ctx, workspaceID)
	if err != nil {
		return 0, err
	}
	timeout := workspaceProfile.GetQueryTimeout()
	if timeout.GetSeconds() <= 0 {
		return math.MaxInt64, nil
	}
	return timeout.GetSeconds(), nil
}

// GetWorkspaceProfileSetting gets the workspace profile setting payload.
func (s *Store) GetWorkspaceProfileSetting(ctx context.Context, workspaceID string) (*storepb.WorkspaceProfileSetting, error) {
	setting, err := s.GetSetting(ctx, workspaceID, storepb.SettingName_WORKSPACE_PROFILE)
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

func (s *Store) GetAppIMSetting(ctx context.Context, workspaceID string) (*storepb.AppIMSetting, error) {
	setting, err := s.GetSetting(ctx, workspaceID, storepb.SettingName_APP_IM)
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

func (s *Store) GetSystemSetting(ctx context.Context, workspaceID string) (*storepb.SystemSetting, error) {
	setting, err := s.GetSetting(ctx, workspaceID, storepb.SettingName_SYSTEM)
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

// GetSystemSettingUncached gets the SYSTEM setting directly from the database,
// bypassing the setting cache. Returns (nil, nil) if no SYSTEM setting exists.
func (s *Store) GetSystemSettingUncached(ctx context.Context, workspaceID string) (*storepb.SystemSetting, error) {
	setting, err := s.GetSettingUncached(ctx, workspaceID, storepb.SettingName_SYSTEM)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get setting %v", storepb.SettingName_SYSTEM)
	}
	if setting == nil {
		return nil, nil
	}
	val, ok := setting.Value.(*storepb.SystemSetting)
	if !ok {
		return nil, errors.Errorf("invalid setting value type for %s", storepb.SettingName_SYSTEM)
	}
	return val, nil
}

// UpdateLicense updates the license in SYSTEM setting.
func (s *Store) UpdateLicense(ctx context.Context, workspaceID string, license string) error {
	setting, err := s.GetSetting(ctx, workspaceID, storepb.SettingName_SYSTEM)
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
		Name:      storepb.SettingName_SYSTEM,
		Workspace: setting.Workspace,
		Value:     systemSetting,
	}); err != nil {
		return errors.Wrap(err, "failed to upsert system setting")
	}
	return nil
}

// GetWorkspaceApprovalSetting gets the workspace approval setting.
func (s *Store) GetWorkspaceApprovalSetting(ctx context.Context, workspaceID string) (*storepb.WorkspaceApprovalSetting, error) {
	setting, err := s.GetSetting(ctx, workspaceID, storepb.SettingName_WORKSPACE_APPROVAL)
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
func (s *Store) GetSemanticTypesSetting(ctx context.Context, workspaceID string) (*storepb.SemanticTypeSetting, error) {
	setting, err := s.GetSetting(ctx, workspaceID, storepb.SettingName_SEMANTIC_TYPES)
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
func (s *Store) GetDataClassificationSetting(ctx context.Context, workspaceID string) (*storepb.DataClassificationSetting, error) {
	setting, err := s.GetSetting(ctx, workspaceID, storepb.SettingName_DATA_CLASSIFICATION)
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

func (s *Store) GetAISetting(ctx context.Context, workspaceID string) (*storepb.AISetting, error) {
	setting, err := s.GetSetting(ctx, workspaceID, storepb.SettingName_AI)
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

func (s *Store) GetEnvironment(ctx context.Context, workspaceID string) (*storepb.EnvironmentSetting, error) {
	setting, err := s.GetSetting(ctx, workspaceID, storepb.SettingName_ENVIRONMENT)
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
	Workspace string
	Name      *storepb.SettingName
}

// GetSetting returns the setting by name.
func (s *Store) GetSetting(ctx context.Context, workspace string, name storepb.SettingName) (*SettingMessage, error) {
	// With caching disabled (HA), every read goes straight to the database:
	// there is no cache to fill, and taking the publish mutex here would
	// serialize all setting reads in the process behind a single lock.
	if !s.enableCache {
		return s.GetSettingUncached(ctx, workspace, name)
	}
	if v, ok := s.settingCache.Get(getSettingCacheKey(workspace, name)); ok {
		return v, nil
	}
	// Fill the cache under the publish mutex so the fill's content is current
	// at publish time: an unordered fill could read a pre-commit snapshot and
	// cache it after a concurrent writer already published a newer value. No
	// row lock is held here, so the row-lock -> publish-lock order is kept.
	s.settingPublishMu.Lock()
	defer s.settingPublishMu.Unlock()
	setting, err := s.GetSettingUncached(ctx, workspace, name)
	if err != nil || setting == nil {
		return setting, err
	}
	s.settingCache.Add(getSettingCacheKey(workspace, name), setting)
	return setting, nil
}

// GetSettingUncached reads the setting directly from the database, bypassing
// the setting cache. The cache has no TTL and only in-process writes refresh
// it, so callers that must observe out-of-band writes (enforcement gates) or
// that merge-and-write the value back (read-modify-write updates) use this to
// avoid acting on a stale cached copy.
func (s *Store) GetSettingUncached(ctx context.Context, workspace string, name storepb.SettingName) (*SettingMessage, error) {
	settings, err := s.ListSettings(ctx, &FindSettingMessage{Workspace: workspace, Name: &name})
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
	q := qb.Q().Space(`
		SELECT
			name,
			workspace,
			value
		FROM setting
		WHERE workspace = ?
	`, find.Workspace)

	if find.Name != nil {
		q.And("name = ?", find.Name.String())
	}
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	rows, err := s.GetDB().QueryContext(ctx, query, args...)
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
			&settingMessage.Workspace,
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

	// Deliberately no cache publication: this path runs outside the publish
	// mutex (GetSettingUncached callers hit it on every read), and an
	// unordered fill could pin a pre-commit snapshot over a newer published
	// value. Publication happens only in GetSetting's fill, UpsertSetting,
	// and UpdateSettingAtomic — all ordered by settingPublishMu.
	return settingMessages, nil
}

// UpdateSettingAtomic performs a row-locking read-modify-write of a setting:
// the current value is read under SELECT ... FOR UPDATE, apply transforms it,
// and the result is written back in the same transaction — so a concurrent
// write can never be silently reverted by a merge based on a stale read (the
// lost-update hazard of read-then-UpsertSetting). apply receives the value
// freshly unmarshaled from the locked row and may mutate it in place; an error
// from apply aborts the transaction with no write and is returned unwrapped so
// callers keep typed errors. After commit, the served state is refreshed via
// publishSetting: the row is re-read under the publish mutex and the fresh
// value goes to the cache and to postCommit (optional, for derived state) —
// so publications always carry current truth regardless of the order in
// which updaters reach the mutex. apply must stay free of database reads:
// it runs while holding the transaction's pooled connection and the row
// lock, and a nested read wanting a second connection is the bounded-pool
// starvation cycle.
func (s *Store) UpdateSettingAtomic(ctx context.Context, workspace string, name storepb.SettingName, apply func(current proto.Message) (proto.Message, error), postCommit func(current *SettingMessage)) (*SettingMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	q := qb.Q().Space(`
		SELECT value FROM setting WHERE workspace = ? AND name = ? FOR UPDATE
	`, workspace, name.String())
	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	var valueString string
	if err := tx.QueryRowContext(ctx, query, args...).Scan(&valueString); err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.Errorf("setting %s not found", name)
		}
		return nil, errors.Wrapf(err, "failed to lock setting %s", name)
	}
	current, err := getSettingMessage(name)
	if err != nil {
		return nil, err
	}
	if err := common.ProtojsonUnmarshaler.Unmarshal([]byte(valueString), current); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal setting %s", name)
	}

	next, err := apply(current)
	if err != nil {
		return nil, err
	}

	nextBytes, err := protojson.Marshal(next)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal setting value")
	}
	q = qb.Q().Space(`
		UPDATE setting SET value = ? WHERE workspace = ? AND name = ?
	`, string(nextBytes), workspace, name.String())
	query, args, err = q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return nil, errors.Wrapf(err, "failed to update setting %s", name)
	}
	// Commit BEFORE acquiring the publish mutex: commit returns the pooled
	// connection and releases the row lock, so waiting for the mutex never
	// holds pool resources (connection-holders never wait for the mutex, so
	// the mutex/pool wait graph stays acyclic). Ordering is preserved because
	// publishSetting re-reads the row under the mutex — whichever updater
	// publishes last still publishes current truth.
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}
	s.publishSetting(ctx, workspace, name, postCommit)
	return &SettingMessage{Name: name, Workspace: workspace, Value: next}, nil
}

// publishSetting refreshes the served state for a setting after a committed
// write: under settingPublishMu it re-reads the row and publishes the fresh
// value to the cache (when enabled) and to postCommit for derived state. It
// must be called with no transaction, row lock, or pooled connection held —
// the mutex-then-connection order here is what keeps the publish protocol
// deadlock-free against the bounded metadata pool. If the re-read fails, the
// cache entry is evicted rather than risk publishing stale state, and
// postCommit is skipped (derived state converges on the next write).
func (s *Store) publishSetting(ctx context.Context, workspace string, name storepb.SettingName, postCommit func(current *SettingMessage)) {
	if !s.enableCache && postCommit == nil {
		return
	}
	s.settingPublishMu.Lock()
	defer s.settingPublishMu.Unlock()
	if s.settingPublishHookForTest != nil {
		s.settingPublishHookForTest()
	}
	// The write is already committed, so publication must not depend on the
	// caller still listening: re-read on a bounded context detached from
	// request cancellation. If the read still fails (database unreachable),
	// evict and skip postCommit — the cache heals on the next fill, and
	// derived runtime state heals on the next successful publication of this
	// setting (postCommit callbacks must therefore reconcile from the fresh
	// value unconditionally, not only for fields their request touched).
	publishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	fresh, err := s.GetSettingUncached(publishCtx, workspace, name)
	if err != nil || fresh == nil {
		s.settingCache.Remove(getSettingCacheKey(workspace, name))
		return
	}
	if s.enableCache {
		s.settingCache.Add(getSettingCacheKey(workspace, name), fresh)
	}
	if postCommit != nil {
		postCommit(fresh)
	}
}

// UpsertSetting upserts the setting by name.
func (s *Store) UpsertSetting(ctx context.Context, update *SettingMessage) (*SettingMessage, error) {
	valueBytes, err := protojson.Marshal(update.Value)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal setting value")
	}

	q := qb.Q().Space(`
		INSERT INTO setting (name, workspace, value)
		VALUES (?, ?, ?)
		ON CONFLICT (name, workspace) DO UPDATE SET value = EXCLUDED.value
		RETURNING name, workspace, value
	`, update.Name.String(), update.Workspace, string(valueBytes))

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var setting SettingMessage
	var nameString string
	var valueString string
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&nameString,
		&setting.Workspace,
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

	s.publishSetting(ctx, setting.Workspace, setting.Name, nil)
	return &setting, nil
}

// DeleteSetting deletes a setting by the name.
func (s *Store) DeleteSetting(ctx context.Context, workspace string, name storepb.SettingName) error {
	q := qb.Q().Space("DELETE FROM setting WHERE name = ? AND workspace = ?", name.String(), workspace)
	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return err
	}

	// Invalidate under the ordering mutex: a cache-miss fill reads and
	// publishes atomically under the same mutex, so once serialized against
	// it, a fill can no longer resurrect the deleted row.
	s.settingPublishMu.Lock()
	defer s.settingPublishMu.Unlock()
	s.settingCache.Remove(getSettingCacheKey(workspace, name))
	return nil
}
