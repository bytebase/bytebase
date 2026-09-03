package store

import (
	"context"
	stdsql "database/sql"
	"fmt"
	"slices"
	"strings"

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
	return projects[0], nil
}

// ListProjects lists all projects.
func (s *Store) ListProjects(ctx context.Context, find *FindProjectMessage) ([]*ProjectMessage, error) {
	if s.enableCache {
		// Keep the database read and cache publication ordered with project
		// invalidation. Otherwise this query can read the pre-update row and
		// repopulate it after UpdateProjects has already removed the old entry.
		s.projectPublishMu.Lock()
		defer s.projectPublishMu.Unlock()
	}

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

	// Titles are not unique; resource_id is the primary key.
	orderBy := []string{}
	for _, v := range find.OrderByKeys {
		orderBy = append(orderBy, fmt.Sprintf("%s %s", v.Key, v.SortOrder.String()))
	}
	orderBy = append(orderBy, "project.resource_id ASC")
	q.Space("ORDER BY " + strings.Join(orderBy, ", "))
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
		s.projectCache.Add(project.ResourceID, project)
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

	// Evict on both sides of the write. Before, so no reader can be served a
	// cache hit carrying the pre-update row once the write commits; after, so a
	// reader that refilled the cache from the pre-update row during the write
	// does not leave that entry behind it.
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

	// The second eviction: see the comment above. Project settings gate
	// authorization, so stale entries could preserve revoked permissions. The
	// publication mutex ensures a pre-update read cannot refill the cache after
	// this eviction.
	for _, patch := range patches {
		s.removeProjectCache(patch.ResourceID)
	}

	return nil
}

func (s *Store) storeProjectCache(project *ProjectMessage) {
	s.projectPublishMu.Lock()
	defer s.projectPublishMu.Unlock()
	s.projectCache.Add(project.ResourceID, project)
}

func (s *Store) removeProjectCache(resourceID string) {
	s.projectPublishMu.Lock()
	defer s.projectPublishMu.Unlock()
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
// row that DeleteProjects will delete (project instances and their databases
// and schemas) or reassign (workspace-instance databases owned by the
// projects). It must run inside the purge transaction before any of those rows
// are removed.
func captureProjectDescendantCacheKeys(ctx context.Context, tx *stdsql.Tx, projectIDs []string) (*projectDescendantCacheKeys, error) {
	keys := &projectDescendantCacheKeys{}

	rows, err := tx.QueryContext(ctx, `
		SELECT resource_id
		FROM instance
		WHERE project = ANY($1)
		ORDER BY resource_id
	`, projectIDs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to capture project instance cache keys for projects %s", strings.Join(projectIDs, ", "))
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
		WHERE instance.project = ANY($1) OR db.project = ANY($1)
		ORDER BY db.instance, db.name
	`, projectIDs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to capture project database cache keys for projects %s", strings.Join(projectIDs, ", "))
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
		WHERE instance.project = ANY($1)
		ORDER BY db_schema.instance, db_schema.db_name
	`, projectIDs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to capture project schema cache keys for projects %s", strings.Join(projectIDs, ", "))
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
// entries and the project entries themselves. It must only run after the purge
// transaction commits; a failed purge must not publish invalidation.
func (s *Store) removeProjectDescendantCaches(keys *projectDescendantCacheKeys, projectIDs []string) {
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
	for _, projectID := range projectIDs {
		s.removeProjectCache(projectID)
	}
}

// DeleteProjects permanently purges soft-deleted projects and all related
// resources in one transaction, so a batch either removes every named project
// or leaves them all intact. This is irreversible and should only be used for:
// - Administrative cleanup of old soft-deleted projects
// - Test cleanup
// Following AIP-164/165, this only works on projects where deleted = TRUE.
func (s *Store) DeleteProjects(ctx context.Context, workspace string, resourceIDs ...string) error {
	if len(resourceIDs) == 0 {
		return nil
	}
	projectList := strings.Join(resourceIDs, ", ")

	defaultProjectID, err := s.GetDefaultProjectID(ctx, workspace)
	if err != nil {
		return errors.Wrap(err, "failed to get default project ID")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()
	// Capture descendant cache keys before any descendant rows are removed so
	// the post-commit invalidation is precise and does not require re-reading
	// rows that no longer exist.
	cacheKeys, err := captureProjectDescendantCacheKeys(ctx, tx, resourceIDs)
	if err != nil {
		return errors.Wrap(err, "failed to capture project descendant cache keys")
	}

	// Delete query history before database-scoped rows.
	q := qb.Q().Space("DELETE FROM query_history WHERE project = ANY(?)", resourceIDs)
	sql, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build query_history delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete query_history for projects %s", projectList)
	}

	// Delete policy entries that reference these projects. Policy rows name the
	// project by resource name, not by ID.
	policyResources := make([]string, 0, len(resourceIDs))
	for _, resourceID := range resourceIDs {
		policyResources = append(policyResources, common.FormatProject(resourceID))
	}
	q = qb.Q().Space("DELETE FROM policy")
	q.Space("WHERE (resource_type = ? AND resource = ANY(?))", storepb.Policy_PROJECT.String(), policyResources)
	q.And("workspace = ?", workspace)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build policy delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete policy for projects %s", projectList)
	}

	// Delete star rows before their parents, in full primary-key order: stars
	// held by project principals, and stars on the saved queries the next
	// statement deletes.
	q = qb.Q().Space(`DELETE FROM saved_query_star
		WHERE (saved_query, principal) IN (
			SELECT saved_query, principal
			FROM saved_query_star
			WHERE principal IN (SELECT email FROM service_account WHERE project = ANY(?) AND workspace = ?)
			   OR principal IN (SELECT email FROM workload_identity WHERE project = ANY(?) AND workspace = ?)
			   OR saved_query IN (
					SELECT resource_id FROM saved_query
					WHERE creator IN (SELECT email FROM service_account WHERE project = ANY(?) AND workspace = ?)
					   OR creator IN (SELECT email FROM workload_identity WHERE project = ANY(?) AND workspace = ?)
			   )
		)`, resourceIDs, workspace, resourceIDs, workspace, resourceIDs, workspace, resourceIDs, workspace)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build saved_query_star delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete saved_query_star for projects %s", projectList)
	}

	// Delete saved queries created by project service accounts or workload identities.
	q = qb.Q().Space(`DELETE FROM saved_query
		WHERE creator IN (SELECT email FROM service_account WHERE project = ANY(?) AND workspace = ?)
		   OR creator IN (SELECT email FROM workload_identity WHERE project = ANY(?) AND workspace = ?)`, resourceIDs, workspace, resourceIDs, workspace)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build saved query delete query for principals")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete saved queries for project principals %s", projectList)
	}

	// Reassign remaining saved queries associated with these projects.
	q = qb.Q().Space("UPDATE saved_query SET project = ? WHERE project = ANY(?)", defaultProjectID, resourceIDs)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build saved query update query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to update saved queries for projects %s", projectList)
	}

	// Delete the project-scoped rows, deepest child first: every chain in
	// backend/store/AGENTS.md is walked bottom-up for the whole batch at once,
	// so one table's rows are removed for every project before the next table.
	for _, statement := range []struct {
		name string
		sql  string
	}{
		{"issue_comment", "DELETE FROM issue_comment WHERE project = ANY(?)"},
		{"review_run", "DELETE FROM review_run WHERE project = ANY(?)"},
		{"issue", "DELETE FROM issue WHERE project = ANY(?)"},
		{"plan_webhook_delivery", "DELETE FROM plan_webhook_delivery WHERE project = ANY(?)"},
		{"plan_check_run", "DELETE FROM plan_check_run WHERE project = ANY(?)"},
		{"task_run_log", "DELETE FROM task_run_log WHERE project = ANY(?)"},
		{"task_run", "DELETE FROM task_run WHERE project = ANY(?)"},
		{"task", "DELETE FROM task WHERE project = ANY(?)"},
		{"plan", "DELETE FROM plan WHERE project = ANY(?)"},
		{"access_grant", "DELETE FROM access_grant WHERE project = ANY(?)"},
		{"release", "DELETE FROM release WHERE project = ANY(?)"},
		{"db_group", "DELETE FROM db_group WHERE project = ANY(?)"},
	} {
		q = qb.Q().Space(statement.sql, resourceIDs)
		sql, args, err = q.ToSQL()
		if err != nil {
			return errors.Wrapf(err, "failed to build %s delete query", statement.name)
		}
		if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
			return errors.Wrapf(err, "failed to delete %s for projects %s", statement.name, projectList)
		}
	}

	// Purge the databases and database-scoped history of project instances.
	// Workspace instances remain shared infrastructure, so their databases are
	// reassigned to the default project below instead.
	for _, statement := range []struct {
		name string
		sql  string
	}{
		{"changelog", "DELETE FROM changelog WHERE instance IN (SELECT resource_id FROM instance WHERE project = ANY(?))"},
		{"sync_history", "DELETE FROM sync_history WHERE instance IN (SELECT resource_id FROM instance WHERE project = ANY(?))"},
		{"revision deleter", `UPDATE revision SET deleter = NULL
			WHERE deleter IN (SELECT email FROM service_account WHERE project = ANY(?) AND workspace = ?)
			   OR deleter IN (SELECT email FROM workload_identity WHERE project = ANY(?) AND workspace = ?)`},
		{"revision", "DELETE FROM revision WHERE instance IN (SELECT resource_id FROM instance WHERE project = ANY(?))"},
		{"db_schema", "DELETE FROM db_schema WHERE instance IN (SELECT resource_id FROM instance WHERE project = ANY(?))"},
		{"database", "DELETE FROM db WHERE instance IN (SELECT resource_id FROM instance WHERE project = ANY(?))"},
	} {
		if statement.name == "revision deleter" {
			q = qb.Q().Space(statement.sql, resourceIDs, workspace, resourceIDs, workspace)
		} else {
			q = qb.Q().Space(statement.sql, resourceIDs)
		}
		sql, args, err = q.ToSQL()
		if err != nil {
			return errors.Wrapf(err, "failed to build %s delete query", statement.name)
		}
		if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
			return errors.Wrapf(err, "failed to delete %s for projects %s", statement.name, projectList)
		}
	}

	// Move only workspace-instance databases to the default project. Project
	// instance databases were deleted with their owning instance above.
	q = qb.Q().Space(`
		UPDATE db
		SET project = ?
		FROM instance
		WHERE db.instance = instance.resource_id
		  AND db.project = ANY(?)
		  AND instance.project IS NULL
	`, defaultProjectID, resourceIDs)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build workspace database update query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to move workspace databases to default project for projects %s", projectList)
	}

	// Delete sheet refs owned by these projects. Blobs stay: content-addressed
	// rows may be shared with other projects, and unreferenced blobs are a
	// future GC's concern (see the sheet_blob comment in LATEST.sql).
	q = qb.Q().Space("DELETE FROM sheet_blob_ref WHERE project = ANY(?)", resourceIDs)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build sheet_blob_ref delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete sheet_blob_ref for projects %s", projectList)
	}

	// Delete project webhooks
	q = qb.Q().Space("DELETE FROM project_webhook WHERE project = ANY(?)", resourceIDs)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build project_webhook delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete project_webhook for projects %s", projectList)
	}

	// Delete project service accounts
	q = qb.Q().Space("DELETE FROM service_account WHERE project = ANY(?) AND workspace = ?", resourceIDs, workspace)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build service_account delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete service accounts for projects %s", projectList)
	}

	// Delete project workload identities
	q = qb.Q().Space("DELETE FROM workload_identity WHERE project = ANY(?) AND workspace = ?", resourceIDs, workspace)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build workload_identity delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete workload identities for projects %s", projectList)
	}

	// Every project instance is deleted with its owner; they are never converted
	// into workspace instances.
	q = qb.Q().Space("DELETE FROM instance WHERE project = ANY(?)", resourceIDs)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build project instance delete query")
	}
	if _, err := tx.ExecContext(ctx, sql, args...); err != nil {
		return errors.Wrapf(err, "failed to delete project instances for projects %s", projectList)
	}

	// Finally, delete the projects themselves. The predicate is the archived
	// guard: a name that is missing, still active, or in another workspace does
	// not come back, and rolls the whole batch back.
	q = qb.Q().Space("DELETE FROM project WHERE resource_id = ANY(?) AND deleted = TRUE AND workspace = ? RETURNING resource_id", resourceIDs, workspace)
	sql, args, err = q.ToSQL()
	if err != nil {
		return errors.Wrap(err, "failed to build project delete query")
	}
	rows, err := tx.QueryContext(ctx, sql, args...)
	if err != nil {
		return errors.Wrapf(err, "failed to delete projects %s", projectList)
	}
	defer rows.Close()
	var purged []string
	for rows.Next() {
		var resourceID string
		if err := rows.Scan(&resourceID); err != nil {
			return errors.Wrap(err, "failed to scan purged project")
		}
		purged = append(purged, resourceID)
	}
	if err := rows.Err(); err != nil {
		return errors.Wrapf(err, "failed to read purged projects %s", projectList)
	}
	for _, resourceID := range resourceIDs {
		if !slices.Contains(purged, resourceID) {
			return errors.Errorf("project %s not found or not marked as deleted", resourceID)
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	// Publish invalidation only after the purge commits: every captured
	// descendant entry is removed while unrelated cache entries survive.
	s.removeProjectDescendantCaches(cacheKeys, resourceIDs)

	return nil
}

func GetListProjectFilter(workspace, filter string) (*qb.Query, error) {
	if filter == "" {
		return nil, nil
	}

	ast, err := common.ParseCELFilter(filter)
	if err != nil {
		return nil, err
	}

	var getFilter func(expr celast.Expr) (*qb.Query, error)

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
				return buildLabelFilterSQL("project.setting", labelKey, value)
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
					return qb.Q().Space("LOWER(project.name) LIKE ? ESCAPE '\\'", containsPattern(strings.ToLower(strValue))), nil
				case "resource_id":
					return qb.Q().Space("LOWER(project.resource_id) LIKE ? ESCAPE '\\'", containsPattern(strings.ToLower(strValue))), nil
				default:
					return nil, errors.Errorf("unsupport variable %q", variable)
				}
			case celoperators.In:
				variable, value := getVariableAndValueFromExpr(expr)
				if labelKey, ok := strings.CutPrefix(variable, "labels."); ok {
					return buildLabelFilterSQL("project.setting", labelKey, value)
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
