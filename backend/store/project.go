package store

import (
	"context"
	stdsql "database/sql"
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	celoverloads "github.com/google/cel-go/common/overloads"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	v1pb "github.com/bytebase/bytebase/backend/generated-go/v1"
)

// ProjectMessage is the message for project.
type ProjectMessage struct {
	ResourceID string
	Workspace  string
	Title      string
	Webhooks   []*ProjectWebhookMessage
	Setting    *storepb.Project
	Deleted    bool
}

func (p *ProjectMessage) GetName() string {
	return fmt.Sprintf("projects/%s", p.ResourceID)
}

// FindProjectMessage is the message for finding projects.
type FindProjectMessage struct {
	Workspace   string
	ResourceID  *string
	ResourceIDs []string
	ShowDeleted bool
	Limit       *int
	Offset      *int
	FilterQ     *qb.Query
	OrderByKeys []*OrderByKey
}

// UpdateProjectMessage is the message for updating a project.
type UpdateProjectMessage struct {
	ResourceID string
	Workspace  string

	Title   *string
	Setting *storepb.Project
	Delete  *bool
}

// GetProject gets project by resource ID.
// GetDefaultProjectID returns the default project resource ID for the given workspace.
// Checks for new format ("default-{workspaceID}") first, falls back to legacy ("default").
func (s *Store) GetDefaultProjectID(ctx context.Context, workspace string) (string, error) {
	newID := common.DefaultProjectID(workspace)
	project, err := s.GetProject(ctx, &FindProjectMessage{Workspace: workspace, ResourceID: new(newID)})
	if err != nil {
		return "", err
	}
	if project != nil {
		return newID, nil
	}
	// Legacy fallback.
	legacyID := "default"
	project, err = s.GetProject(ctx, &FindProjectMessage{Workspace: workspace, ResourceID: &legacyID})
	if err != nil {
		return "", err
	}
	if project != nil {
		return legacyID, nil
	}
	return newID, nil
}

func (s *Store) GetProject(ctx context.Context, find *FindProjectMessage) (*ProjectMessage, error) {
	if find.ResourceID != nil {
		if v, ok := s.projectCache.Get(*find.ResourceID); ok && s.enableCache {
			return v, nil
		}
	}

	// We will always return the resource regardless of its deleted state.
	find.ShowDeleted = true

	projects, err := s.ListProjects(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(projects) == 0 {
		return nil, nil
	}
	if len(projects) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d projects with filter %+v, expect 1", len(projects), find)}
	}
	project := projects[0]

	s.storeProjectCache(project)
	return project, nil
}

// ListProjects lists all projects.
func (s *Store) ListProjects(ctx context.Context, find *FindProjectMessage) ([]*ProjectMessage, error) {
	q := qb.Q().Space("SELECT resource_id, workspace, name, setting, deleted FROM project WHERE workspace = ?", find.Workspace)
	if filterQ := find.FilterQ; filterQ != nil {
		q.And("?", filterQ)
	}
	if v := find.ResourceID; v != nil {
		q.And("resource_id = ?", *v)
	}
	if len(find.ResourceIDs) > 0 {
		q.And("resource_id = ANY(?)", find.ResourceIDs)
	}
	if !find.ShowDeleted {
		q.And("deleted = ?", false)
	}

	if len(find.OrderByKeys) > 0 {
		orderBy := []string{}
		for _, v := range find.OrderByKeys {
			orderBy = append(orderBy, fmt.Sprintf("%s %s", v.Key, v.SortOrder.String()))
		}
		q.Space(fmt.Sprintf("ORDER BY %s", strings.Join(orderBy, ", ")))
	} else {
		q.Space("ORDER BY project.resource_id")
	}
	if v := find.Limit; v != nil {
		q.Space("LIMIT ?", *v)
	}
	if v := find.Offset; v != nil {
		q.Space("OFFSET ?", *v)
	}

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}

	var projectMessages []*ProjectMessage
	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var projectMessage ProjectMessage
		var payload []byte
		if err := rows.Scan(
			&projectMessage.ResourceID,
			&projectMessage.Workspace,
			&projectMessage.Title,
			&payload,
			&projectMessage.Deleted,
		); err != nil {
			return nil, err
		}
		setting := &storepb.Project{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, setting); err != nil {
			return nil, err
		}
		projectMessage.Setting = setting
		projectMessages = append(projectMessages, &projectMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Batch fetch webhooks for all projects in a single query.
	if len(projectMessages) > 0 {
		projectIDs := make([]string, 0, len(projectMessages))
		for _, p := range projectMessages {
			projectIDs = append(projectIDs, p.ResourceID)
		}
		webhooks, err := s.ListProjectWebhooks(ctx, &FindProjectWebhookMessage{ProjectIDs: projectIDs})
		if err != nil {
			return nil, err
		}
		// Group webhooks by project ID.
		webhooksByProject := make(map[string][]*ProjectWebhookMessage)
		for _, w := range webhooks {
			webhooksByProject[w.ProjectID] = append(webhooksByProject[w.ProjectID], w)
		}
		for _, project := range projectMessages {
			project.Webhooks = webhooksByProject[project.ResourceID]
		}
	}

	for _, project := range projectMessages {
		s.storeProjectCache(project)
	}
	return projectMessages, nil
}

// CreateProject creates a project.
func (s *Store) CreateProject(ctx context.Context, create *ProjectMessage, creator *UserMessage) (*ProjectMessage, error) {
	if creator == nil {
		return nil, errors.Errorf("creator cannot be nil")
	}
	if create.Setting == nil {
		create.Setting = &storepb.Project{}
	}
	payload, err := protojson.Marshal(create.Setting)
	if err != nil {
		return nil, err
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	project := &ProjectMessage{
		ResourceID: create.ResourceID,
		Workspace:  create.Workspace,
		Title:      create.Title,
		Setting:    create.Setting,
	}
	q := qb.Q().Space("INSERT INTO project (resource_id, workspace, name, setting)")
	q.Space("VALUES (?, ?, ?, ?)", create.ResourceID, create.Workspace, create.Title, payload)
	sql, args, err := q.ToSQL()
	if err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return nil, err
	}

	iamPolicy := &storepb.IamPolicy{
		Bindings: []*storepb.Binding{
			{
				Role: common.FormatRole(ProjectOwnerRole),
				Members: []string{
					common.FormatUserEmail(creator.Email),
				},
				Condition: &expr.Expr{},
			},
		},
	}
	policyPayload, err := protojson.Marshal(iamPolicy)
	if err != nil {
		return nil, err
	}
	policyMessage, err := upsertPolicyImpl(ctx, tx, &PolicyMessage{
		Workspace:         create.Workspace,
		ResourceType:      storepb.Policy_PROJECT,
		Resource:          common.FormatProject(project.ResourceID),
		Payload:           string(policyPayload),
		Type:              storepb.Policy_IAM,
		InheritFromParent: false,
		// Enforce cannot be false while creating a policy.
		Enforce: true,
	})
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.policyCache.Add(getPolicyCacheKey(policyMessage.Workspace, policyMessage.ResourceType, policyMessage.Resource, policyMessage.Type), policyMessage)
	s.storeProjectCache(project)
	return project, nil
}

// UpdateProjects updates projects in a single transaction.
func (s *Store) UpdateProjects(ctx context.Context, patches ...*UpdateProjectMessage) error {
	if len(patches) == 0 {
		return nil
	}

	// Remove all projects from cache first
	for _, patch := range patches {
		s.removeProjectCache(patch.ResourceID)
	}

	// Prepare arrays for batch update
	resourceIDs := make([]string, len(patches))
	titles := make([]*string, len(patches))
	deleteds := make([]*bool, len(patches))
	settings := make([]*string, len(patches))

	for i, patch := range patches {
		resourceIDs[i] = patch.ResourceID
		titles[i] = patch.Title
		deleteds[i] = patch.Delete
		if patch.Setting != nil {
			payload, err := protojson.Marshal(patch.Setting)
			if err != nil {
				return err
			}
			settings[i] = new(string(payload))
		}
	}

	// Batch update using UPDATE FROM unnest
	q := qb.Q().Space(`
		UPDATE project AS p SET
			name = COALESCE(u.name, p.name),
			deleted = COALESCE(u.deleted, p.deleted),
			setting = COALESCE(u.setting::jsonb, p.setting)
		FROM unnest(?::text[], ?::text[], ?::bool[], ?::text[]) AS u(resource_id, name, deleted, setting)
		WHERE p.resource_id = u.resource_id AND p.workspace = ?
	`, resourceIDs, titles, deleteds, settings, patches[0].Workspace)

	query, args, err := q.ToSQL()
	if err != nil {
		return err
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return err
	}

	return nil
}

func (s *Store) storeProjectCache(project *ProjectMessage) {
	s.projectCache.Add(project.ResourceID, project)
}

func (s *Store) removeProjectCache(resourceID string) {
	s.projectCache.Remove(resourceID)
}

// projectDescendantCacheKeys snapshots the cache keys of every instance,
// database, and schema row that a project purge is about to remove or
// reassign, so their cache entries can be invalidated after the purge commits.
type projectDescendantCacheKeys struct {
	instanceIDs []string
	databases   []projectDatabaseCacheKey
	schemas     []projectSchemaCacheKey
}

type projectDatabaseCacheKey struct {
	workspace    string
	instanceID   string
	databaseName string
}

type projectSchemaCacheKey struct {
	instanceID   string
	databaseName string
}

// captureProjectDescendantCacheKeys reads the cache keys of every descendant
// row that DeleteProject will delete (project instances and their databases
// and schemas) or reassign (workspace-instance databases owned by the
// project). It must run inside the purge transaction before any of those rows
// are removed.
func captureProjectDescendantCacheKeys(ctx context.Context, tx *stdsql.Tx, projectID string) (*projectDescendantCacheKeys, error) {
	keys := &projectDescendantCacheKeys{}

	rows, err := tx.QueryContext(ctx, `
		SELECT resource_id
		FROM instance
		WHERE project = $1
		ORDER BY resource_id
	`, projectID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to capture project instance cache keys for project %s", projectID)
	}
	defer rows.Close()
	for rows.Next() {
		var instanceID string
		if err := rows.Scan(&instanceID); err != nil {
			return nil, errors.Wrap(err, "failed to scan project instance cache key")
		}
		keys.instanceIDs = append(keys.instanceIDs, instanceID)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to read project instance cache keys")
	}

	rows, err = tx.QueryContext(ctx, `
		SELECT db.instance, db.name, instance.workspace
		FROM db
		JOIN instance ON instance.resource_id = db.instance
		WHERE instance.project = $1 OR db.project = $1
		ORDER BY db.instance, db.name
	`, projectID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to capture project database cache keys for project %s", projectID)
	}
	defer rows.Close()
	for rows.Next() {
		var key projectDatabaseCacheKey
		if err := rows.Scan(&key.instanceID, &key.databaseName, &key.workspace); err != nil {
			return nil, errors.Wrap(err, "failed to scan project database cache key")
		}
		keys.databases = append(keys.databases, key)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to read project database cache keys")
	}

	rows, err = tx.QueryContext(ctx, `
		SELECT db_schema.instance, db_schema.db_name
		FROM db_schema
		JOIN instance ON instance.resource_id = db_schema.instance
		WHERE instance.project = $1
		ORDER BY db_schema.instance, db_schema.db_name
	`, projectID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to capture project schema cache keys for project %s", projectID)
	}
	defer rows.Close()
	for rows.Next() {
		var key projectSchemaCacheKey
		if err := rows.Scan(&key.instanceID, &key.databaseName); err != nil {
			return nil, errors.Wrap(err, "failed to scan project schema cache key")
		}
		keys.schemas = append(keys.schemas, key)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to read project schema cache keys")
	}

	return keys, nil
}

// removeProjectDescendantCaches invalidates the captured descendant cache
// entries and the project entry itself. It must only run after the purge
// transaction commits; a failed purge must not publish invalidation.
func (s *Store) removeProjectDescendantCaches(keys *projectDescendantCacheKeys, projectID string) {
	for _, instanceID := range keys.instanceIDs {
		s.instanceCache.Remove(getInstanceCacheKey(instanceID))
	}
	for _, key := range keys.databases {
		// Remove both the unscoped (runner) and workspace-scoped (API) entries.
		s.databaseCache.Remove(getDatabaseCacheKey("", key.instanceID, key.databaseName))
		s.databaseCache.Remove(getDatabaseCacheKey(key.workspace, key.instanceID, key.databaseName))
	}
	for _, key := range keys.schemas {
		s.dbSchemaCache.Remove(getDBSchemaCacheKey(key.instanceID, key.databaseName))
	}
	s.projectCache.Remove(projectID)
}

// DeleteProject permanently purges a soft-deleted project and all related resources.
// This operation is irreversible and should only be used for:
// - Administrative cleanup of old soft-deleted projects
// - Test cleanup
// Following AIP-164/165, this only works on projects where deleted = TRUE.
func (s *Store) DeleteProject(ctx context.Context, workspace string, resourceID string) error {
	defaultProjectID, err := s.GetDefaultProjectID(ctx, workspace)
	if err != nil {
		return errors.Wrap(err, "failed to get default project ID")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()
	if err := acquireProjectPurgeLock(ctx, tx, resourceID); err != nil {
		return errors.Wrapf(err, "failed to lock project purge fence for %s", resourceID)
	}

	// Capture descendant cache keys before any descendant rows are removed so
	// the post-commit invalidation is precise and does not require re-reading
	// rows that no longer exist.
	cacheKeys, err := captureProjectDescendantCacheKeys(ctx, tx, resourceID)
	if err != nil {
		return errors.Wrap(err, "failed to capture project descendant cache keys")
	}

	// Delete query history before locking database-scoped rows to preserve the
	// canonical sibling-branch order.
	q := qb.Q().Space("DELETE FROM query_history WHERE project = ?", resourceID)
	sql, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build query_history delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete query_history for project %s", resourceID)
	}

	// Delete policy entries that reference this project.
	q = qb.Q().Space("DELETE FROM policy")
	q.Space("WHERE (resource_type = ? AND resource = 'projects/' || ?)", storepb.Policy_PROJECT.String(), resourceID)
	q.And("workspace = ?", workspace)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build policy delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete policy for project %s", resourceID)
	}

	// Delete saved_query_organizer entries referencing project principals.
	q = qb.Q().Space(`DELETE FROM saved_query_organizer
		WHERE principal IN (SELECT email FROM service_account WHERE project = ? AND workspace = ?)
		   OR principal IN (SELECT email FROM workload_identity WHERE project = ? AND workspace = ?)`, resourceID, workspace, resourceID, workspace)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build saved_query_organizer delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete saved_query_organizer for project %s", resourceID)
	}

	// Delete saved queries created by project service accounts or workload identities.
	q = qb.Q().Space(`DELETE FROM saved_query
		WHERE creator IN (SELECT email FROM service_account WHERE project = ? AND workspace = ?)
		   OR creator IN (SELECT email FROM workload_identity WHERE project = ? AND workspace = ?)`, resourceID, workspace, resourceID, workspace)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build saved query delete query for principals")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete saved queries for project principals %s", resourceID)
	}

	// Reassign remaining saved queries associated with this project.
	q = qb.Q().Space("UPDATE saved_query SET project = ? WHERE project = ?", defaultProjectID, resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build saved query update query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to update saved queries for project %s", resourceID)
	}

	// Delete issue_comment entries for issues in this project
	q = qb.Q().Space("DELETE FROM issue_comment WHERE project = ?", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build issue_comment delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete issue_comment for project %s", resourceID)
	}

	// Delete issues associated with this project
	q = qb.Q().Space("DELETE FROM issue WHERE project = ?", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build issue delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete issues for project %s", resourceID)
	}

	// Delete plan_webhook_delivery entries for plans in this project
	q = qb.Q().Space("DELETE FROM plan_webhook_delivery WHERE project = ?", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build plan_webhook_delivery delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete plan_webhook_delivery for project %s", resourceID)
	}

	// Delete plan_check_run entries for plans in this project
	q = qb.Q().Space("DELETE FROM plan_check_run WHERE project = ?", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build plan_check_run delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete plan_check_run for project %s", resourceID)
	}

	// Delete task_run_log entries for tasks in plans of this project
	q = qb.Q().Space("DELETE FROM task_run_log WHERE project = ?", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build task_run_log delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete task_run_log for project %s", resourceID)
	}

	// Delete task_run entries for tasks in plans of this project
	q = qb.Q().Space("DELETE FROM task_run WHERE project = ?", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build task_run delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete task_run for project %s", resourceID)
	}

	// Lock tasks in full primary-key order before deleting them. Pending Task Run
	// creation uses the same order, so concurrent batches cannot form a wait cycle.
	q = qb.Q().Space(`
		SELECT project, id
		FROM task
		WHERE project = ?
		ORDER BY project, id
		FOR UPDATE`, resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build task lock query")
	}
	if err := func() error {
		rows, err := tx.QueryContext(ctx, sql, args...)
		if err != nil {
			return errors.Wrapf(err, "failed to lock tasks for project %s", resourceID)
		}
		defer rows.Close()
		for rows.Next() {
			var lockedProjectID string
			var lockedTaskID int64
			if err := rows.Scan(&lockedProjectID, &lockedTaskID); err != nil {
				return errors.Wrap(err, "failed to scan locked task")
			}
		}
		if err := rows.Err(); err != nil {
			return errors.Wrap(err, "failed to read locked tasks")
		}
		return nil
	}(); err != nil {
		return err
	}

	// Delete tasks in plans of this project after acquiring every task lock.
	q = qb.Q().Space("DELETE FROM task WHERE project = ?", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build task delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete tasks for project %s", resourceID)
	}

	// Delete plans associated with this project
	q = qb.Q().Space("DELETE FROM plan WHERE project = ?", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build plan delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete plans for project %s", resourceID)
	}

	// Delete access_grant associated with this project
	q = qb.Q().Space("DELETE FROM access_grant WHERE project = ?", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build access_grant delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete access_grants for project %s", resourceID)
	}

	// Delete releases associated with this project
	q = qb.Q().Space("DELETE FROM release WHERE project = ?", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build release delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete releases for project %s", resourceID)
	}

	// Delete db_groups associated with this project
	q = qb.Q().Space("DELETE FROM db_group WHERE project = ?", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build db_group delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete db_groups for project %s", resourceID)
	}

	// Purge the databases and database-scoped history of project instances.
	// Workspace instances remain shared infrastructure, so their databases are
	// reassigned to the default project below instead.
	for _, statement := range []struct {
		name string
		sql  string
	}{
		{"changelog", "DELETE FROM changelog WHERE instance IN (SELECT resource_id FROM instance WHERE project = ?)"},
		{"sync_history", "DELETE FROM sync_history WHERE instance IN (SELECT resource_id FROM instance WHERE project = ?)"},
		{"revision deleter", `UPDATE revision SET deleter = NULL
			WHERE deleter IN (SELECT email FROM service_account WHERE project = ? AND workspace = ?)
			   OR deleter IN (SELECT email FROM workload_identity WHERE project = ? AND workspace = ?)`},
		{"revision", "DELETE FROM revision WHERE instance IN (SELECT resource_id FROM instance WHERE project = ?)"},
		{"db_schema", "DELETE FROM db_schema WHERE instance IN (SELECT resource_id FROM instance WHERE project = ?)"},
		{"database", "DELETE FROM db WHERE instance IN (SELECT resource_id FROM instance WHERE project = ?)"},
	} {
		if statement.name == "revision deleter" {
			q = qb.Q().Space(statement.sql, resourceID, workspace, resourceID, workspace)
		} else {
			q = qb.Q().Space(statement.sql, resourceID)
		}
		sql, args, err = q.ToSQL()
		if err != nil {
			return errors.Wrapf(err, "failed to build %s delete query", statement.name)
		}
		if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
			return errors.Wrapf(err, "failed to delete %s for project %s", statement.name, resourceID)
		}
	}

	// Move only workspace-instance databases to the default project. Project
	// instance databases were deleted with their owning instance above.
	q = qb.Q().Space(`
		UPDATE db
		SET project = ?
		FROM instance
		WHERE db.instance = instance.resource_id
		  AND db.project = ?
		  AND instance.project IS NULL
	`, defaultProjectID, resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build workspace database update query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to move workspace databases to default project for project %s", resourceID)
	}

	// Delete sheet refs owned by this project. Blobs stay: content-addressed
	// rows may be shared with other projects, and unreferenced blobs are a
	// future GC's concern (see the sheet_blob comment in LATEST.sql).
	q = qb.Q().Space("DELETE FROM sheet_blob_ref WHERE project = ?", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build sheet_blob_ref delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete sheet_blob_ref for project %s", resourceID)
	}

	// Delete project webhooks
	q = qb.Q().Space("DELETE FROM project_webhook WHERE project = ?", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build project_webhook delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete project_webhook for project %s", resourceID)
	}

	// Delete project service accounts
	q = qb.Q().Space("DELETE FROM service_account WHERE project = ? AND workspace = ?", resourceID, workspace)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build service_account delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete service accounts for project %s", resourceID)
	}

	// Delete project workload identities
	q = qb.Q().Space("DELETE FROM workload_identity WHERE project = ? AND workspace = ?", resourceID, workspace)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build workload_identity delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete workload identities for project %s", resourceID)
	}

	// Lock existing project instances after all of their descendants. A new
	// project instance is rejected because this project is already soft-deleted.
	q = qb.Q().Space(`
		SELECT resource_id
		FROM instance
		WHERE project = ?
		ORDER BY resource_id
		FOR UPDATE
	`, resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build project instance lock query")
	}
	if err := func() error {
		rows, err := tx.QueryContext(ctx, sql, args...)
		if err != nil {
			return errors.Wrapf(err, "failed to lock project instances for project %s", resourceID)
		}
		defer rows.Close()
		for rows.Next() {
			var instanceID string
			if err := rows.Scan(&instanceID); err != nil {
				return errors.Wrap(err, "failed to scan locked project instance")
			}
		}
		return rows.Err()
	}(); err != nil {
		return err
	}

	// Every project instance is deleted with its owner; they are never converted
	// into workspace instances.
	q = qb.Q().Space("DELETE FROM instance WHERE project = ?", resourceID)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build project instance delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete project instances for project %s", resourceID)
	}

	var projectDeleted bool
	if err := tx.QueryRowContext(ctx, `
		SELECT deleted
		FROM project
		WHERE resource_id = $1 AND workspace = $2
		FOR UPDATE
	`, resourceID, workspace).Scan(&projectDeleted); err != nil {
		if errors.Is(err, stdsql.ErrNoRows) {
			return errors.Errorf("project %s not found or not marked as deleted", resourceID)
		}
		return errors.Wrapf(err, "failed to lock project %s", resourceID)
	}
	if !projectDeleted {
		return errors.Errorf("project %s not found or not marked as deleted", resourceID)
	}

	// Finally, delete the project itself (only if it's marked as deleted)
	q = qb.Q().Space("DELETE FROM project WHERE resource_id = ? AND deleted = TRUE AND workspace = ?", resourceID, workspace)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build project delete query")
	}
	result, err := tx.ExecContext(ctx, sql, args...)
	if err != nil {
		return errors.Wrapf(err, "failed to delete project %s", resourceID)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to get rows affected")
	}
	if rowsAffected == 0 {
		return errors.Errorf("project %s not found or not marked as deleted", resourceID)
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	// Publish invalidation only after the purge commits: every captured
	// descendant entry is removed while unrelated cache entries survive.
	s.removeProjectDescendantCaches(cacheKeys, resourceID)

	return nil
}

func GetListProjectFilter(workspace, filter string) (*qb.Query, error) {
	if filter == "" {
		return nil, nil
	}

	e, err := cel.NewEnv()
	if err != nil {
		return nil, errors.Errorf("failed to create cel env")
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return nil, errors.Errorf("failed to parse filter %v, error: %v", filter, iss.String())
	}

	var getFilter func(expr celast.Expr) (*qb.Query, error)

	parseToLabelFilterSQL := func(resource, key string, value any) (*qb.Query, error) {
		switch v := value.(type) {
		case string:
			return qb.Q().Space(fmt.Sprintf("%s->'labels'->>'%s' = ?", resource, key), v), nil
		case []any:
			if len(v) == 0 {
				return nil, errors.Errorf("empty label filter")
			}

			labelValueList := make([]any, len(v))
			for i, raw := range v {
				str, ok := raw.(string)
				if !ok {
					return nil, errors.Errorf("label value must be string, got %T", raw)
				}
				labelValueList[i] = str
			}
			return qb.Q().Space(fmt.Sprintf("%s->'labels'->>'%s' = ANY(?)", resource, key), labelValueList), nil
		default:
			return nil, errors.Errorf("empty value %v for label filter", value)
		}
	}

	parseToSQL := func(variable, value any) (*qb.Query, error) {
		switch variable {
		case "name":
			return qb.Q().Space("project.name = ?", value.(string)), nil
		case "resource_id":
			return qb.Q().Space("project.resource_id = ?", value.(string)), nil
		case "exclude_default":
			if excludeDefault, ok := value.(bool); excludeDefault && ok {
				return qb.Q().Space("project.resource_id != ? AND project.resource_id != 'default'", common.DefaultProjectID(workspace)), nil
			}
			return qb.Q().Space("TRUE"), nil
		case "state":
			stateStr, ok := value.(string)
			if !ok {
				return nil, errors.Errorf("state value must be string, got %T", value)
			}
			// Try with STATE_ prefix first (e.g., "STATE_ACTIVE", "STATE_DELETED")
			v1State, ok := v1pb.State_value[stateStr]
			if !ok {
				// If not found, try without STATE_ prefix (e.g., "ACTIVE", "DELETED")
				if v, exists := v1pb.State_value[strings.TrimPrefix(stateStr, "STATE_")]; exists {
					v1State = v
					ok = true
				}
			}
			if !ok {
				return nil, errors.Errorf("invalid state filter %q", value)
			}
			return qb.Q().Space("project.deleted = ?", v1pb.State(v1State) == v1pb.State_DELETED), nil
		default:
			varStr, ok := variable.(string)
			if !ok {
				return nil, errors.Errorf("unsupport variable %q", variable)
			}
			if labelKey, ok := strings.CutPrefix(varStr, "labels."); ok {
				return parseToLabelFilterSQL("project.setting", labelKey, value)
			}
			return nil, errors.Errorf("unsupport variable %q", variable)
		}
	}

	getFilter = func(expr celast.Expr) (*qb.Query, error) {
		q := qb.Q()
		switch expr.Kind() {
		case celast.CallKind:
			functionName := expr.AsCall().FunctionName()
			switch functionName {
			case celoperators.LogicalOr:
				for _, arg := range expr.AsCall().Args() {
					qq, err := getFilter(arg)
					if err != nil {
						return nil, err
					}
					q.Or("?", qq)
				}
				return qb.Q().Space("(?)", q), nil
			case celoperators.LogicalAnd:
				for _, arg := range expr.AsCall().Args() {
					qq, err := getFilter(arg)
					if err != nil {
						return nil, err
					}
					q.And("?", qq)
				}
				return qb.Q().Space("(?)", q), nil
			case celoperators.Equals:
				variable, value := getVariableAndValueFromExpr(expr)
				return parseToSQL(variable, value)
			case celoverloads.Contains:
				variable := expr.AsCall().Target().AsIdent()
				args := expr.AsCall().Args()
				if len(args) != 1 {
					return nil, errors.Errorf(`invalid args for %q`, variable)
				}
				value := args[0].AsLiteral().Value()
				strValue, ok := value.(string)
				if !ok {
					return nil, errors.Errorf("expect string, got %T, hint: filter literals should be string", value)
				}

				switch variable {
				case "name":
					return qb.Q().Space("LOWER(project.name) LIKE ?", "%"+strings.ToLower(strValue)+"%"), nil
				case "resource_id":
					return qb.Q().Space("LOWER(project.resource_id) LIKE ?", "%"+strings.ToLower(strValue)+"%"), nil
				default:
					return nil, errors.Errorf("unsupport variable %q", variable)
				}
			case celoperators.In:
				variable, value := getVariableAndValueFromExpr(expr)
				if labelKey, ok := strings.CutPrefix(variable, "labels."); ok {
					return parseToLabelFilterSQL("project.setting", labelKey, value)
				}
				return nil, errors.Errorf("unexpected %v operator for %v", functionName, variable)
			default:
				return nil, errors.Errorf("unexpected function %v", functionName)
			}
		default:
			return nil, errors.Errorf("unexpected expr kind %v", expr.Kind())
		}
	}

	q, err := getFilter(ast.NativeRep().Expr())
	if err != nil {
		return nil, err
	}
	return qb.Q().Space("(?)", q), nil
}

func GetProjectOrders(orderBy string) ([]*OrderByKey, error) {
	keys, err := parseOrderBy(orderBy)
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return nil, nil
	}
	if len(keys) > 1 || keys[0].Key != "title" {
		return nil, errors.Errorf(`only support order by "title"`)
	}

	return []*OrderByKey{
		{
			Key:       "name",
			SortOrder: keys[0].SortOrder,
		},
	}, nil
}
