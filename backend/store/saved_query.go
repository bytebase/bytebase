package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/cel-go/cel"
	celast "github.com/google/cel-go/common/ast"
	celoperators "github.com/google/cel-go/common/operators"
	celoverloads "github.com/google/cel-go/common/overloads"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// SavedQueryMessage is the message for a saved query.
type SavedQueryMessage struct {
	ProjectID  string
	ResourceID string
	// The connected database's canonical resource name, or "" for none.
	// Validated against the saved query's own project at write time; the
	// reference is soft and may dangle afterwards.
	Database string

	Creator string

	Title     string
	Statement string
	// The folder path this saved query lives in ("a/b/c", "" = unfiled).
	Folder string

	// Bindings are the per-object grants. Empty means private to the creator
	// (the admin backstop aside).
	Bindings []*storepb.SavedQueryBinding

	// Output only fields
	Size      int64
	CreatedAt time.Time
	UpdatedAt time.Time
	// Whether FindSavedQueryMessage.PrincipalEmail starred this saved query.
	Starred bool
}

// FindSavedQueryMessage is the API message for finding sheets.
type FindSavedQueryMessage struct {
	// Either ProjectIDs or Workspace is required.
	ProjectIDs []string
	Workspace  string

	PrincipalEmail string

	// AccessMembers restricts results to saved queries the caller can read:
	// their own, plus any whose bindings name one of these principals
	// ("user:{email}" and "group:{email}" — the caller and their groups).
	// Leave nil to skip the access clause entirely, which is what the auditor
	// List does.
	//
	// Must describe the same caller as PrincipalEmail: the clause reads "own"
	// from PrincipalEmail and "granted" from here. Setting one without the
	// other narrows the result rather than widening it, but the two disagreeing
	// would be a bug.
	AccessMembers []string

	ResourceID *string

	// LoadFull is used if we want to load the full sheet.
	LoadFull bool

	FilterQ *qb.Query

	Limit  *int
	Offset *int
}

// PatchSavedQueryMessage is the message to patch a saved query.
type PatchSavedQueryMessage struct {
	ResourceID string
	Title      *string
	Statement  *string
	// Database sets the connected database ("" clears it).
	Database *string
	// Folder re-files the saved query ("" = unfiled).
	Folder *string
}

// GetSavedQuery gets a sheet.
func (s *Store) GetSavedQuery(ctx context.Context, find *FindSavedQueryMessage) (*SavedQueryMessage, error) {
	sheets, err := s.ListSavedQueries(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(sheets) == 0 {
		return nil, nil
	}
	if len(sheets) > 1 {
		return nil, errors.Errorf("expected 1 sheet, got %d", len(sheets))
	}
	sheet := sheets[0]

	return sheet, nil
}

// ListSavedQueries returns a list of sheets.
func (s *Store) ListSavedQueries(ctx context.Context, find *FindSavedQueryMessage) ([]*SavedQueryMessage, error) {
	if len(find.ProjectIDs) == 0 && find.Workspace == "" {
		return nil, errors.Errorf("empty project filter")
	}
	statementField := fmt.Sprintf("LEFT(saved_query.statement, %d)", common.MaxSheetSize)
	if find.LoadFull {
		statementField = "saved_query.statement"
	}

	q := qb.Q().Space(fmt.Sprintf(`
		SELECT
			saved_query.resource_id,
			saved_query.creator,
			saved_query.created_at,
			saved_query.updated_at,
			saved_query.project,
			saved_query.payload,
			saved_query.folder,
			saved_query.name,
			saved_query.bindings,
			%s,
			OCTET_LENGTH(saved_query.statement),
			EXISTS (SELECT 1 FROM saved_query_star WHERE saved_query_star.saved_query = saved_query.resource_id AND saved_query_star.principal = ?)
		FROM saved_query
		WHERE TRUE`, statementField), find.PrincipalEmail)

	// The read predicate, pushed into SQL rather than applied to the page
	// afterwards: filtering after LIMIT would silently return short pages.
	if find.AccessMembers != nil {
		access := qb.Q().Space("saved_query.creator = ?", find.PrincipalEmail)
		for _, member := range find.AccessMembers {
			probe, err := bindingProbe(member)
			if err != nil {
				return nil, err
			}
			access.Or("saved_query.bindings @> ?", probe)
		}
		q.And("(?)", access)
	}

	if find.Workspace != "" {
		q.And("EXISTS (SELECT 1 FROM project WHERE project.resource_id = saved_query.project AND project.workspace = ? AND project.deleted = FALSE)", find.Workspace)
	}
	if len(find.ProjectIDs) == 1 {
		q.And("saved_query.project = ?", find.ProjectIDs[0])
	} else if len(find.ProjectIDs) > 1 {
		q.And("saved_query.project = ANY(?)", find.ProjectIDs)
	}

	if filterQ := find.FilterQ; filterQ != nil {
		q.And("?", filterQ)
	}

	if v := find.ResourceID; v != nil {
		q.And("saved_query.resource_id = ?", *v)
	}

	q.Space("ORDER BY saved_query.name, saved_query.resource_id")
	if v := find.Limit; v != nil {
		q.Space("LIMIT ?", *v)
	}
	if v := find.Offset; v != nil {
		q.Space("OFFSET ?", *v)
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

	var sheets []*SavedQueryMessage
	for rows.Next() {
		var sheet SavedQueryMessage
		var payloadBytes, bindingsBytes []byte
		if err := rows.Scan(
			&sheet.ResourceID,
			&sheet.Creator,
			&sheet.CreatedAt,
			&sheet.UpdatedAt,
			&sheet.ProjectID,
			&payloadBytes,
			&sheet.Folder,
			&sheet.Title,
			&bindingsBytes,
			&sheet.Statement,
			&sheet.Size,
			&sheet.Starred,
		); err != nil {
			return nil, err
		}

		var payload storepb.SavedQueryPayload
		if err := common.ProtojsonUnmarshaler.Unmarshal(payloadBytes, &payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal saved query payload")
		}
		sheet.Database = payload.Database

		bindings, err := unmarshalSavedQueryBindings(bindingsBytes)
		if err != nil {
			return nil, err
		}
		sheet.Bindings = bindings

		sheets = append(sheets, &sheet)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sheets, nil
}

// bindingProbe builds the jsonb containment operand that matches any binding
// naming this principal, e.g. `[{"members":["user:a@corp.com"]}]`. `@>` on a
// jsonb array is "contains an element containing this", so one probe per
// principal, BitmapOr'd by the planner, answers "shared with me".
func bindingProbe(member string) (string, error) {
	probe, err := json.Marshal([]map[string]any{{"members": []string{member}}})
	if err != nil {
		return "", errors.Wrapf(err, "failed to build binding probe for %q", member)
	}
	return string(probe), nil
}

// marshalSavedQueryBindings writes the bindings as a protojson array at the
// jsonb root. protojson has no top-level array form, so each element is
// marshalled on its own and the array assembled here; the shape is pinned
// because the GIN containment probes above depend on it.
func marshalSavedQueryBindings(bindings []*storepb.SavedQueryBinding) (string, error) {
	elements := make([]json.RawMessage, 0, len(bindings))
	for _, binding := range bindings {
		element, err := protojson.Marshal(binding)
		if err != nil {
			return "", errors.Wrapf(err, "failed to marshal saved query binding")
		}
		elements = append(elements, element)
	}
	out, err := json.Marshal(elements)
	if err != nil {
		return "", errors.Wrapf(err, "failed to marshal saved query bindings")
	}
	return string(out), nil
}

func unmarshalSavedQueryBindings(b []byte) ([]*storepb.SavedQueryBinding, error) {
	if len(b) == 0 {
		return nil, nil
	}
	var elements []json.RawMessage
	if err := json.Unmarshal(b, &elements); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal saved query bindings")
	}
	bindings := make([]*storepb.SavedQueryBinding, 0, len(elements))
	for _, element := range elements {
		var binding storepb.SavedQueryBinding
		if err := common.ProtojsonUnmarshaler.Unmarshal(element, &binding); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal saved query binding")
		}
		bindings = append(bindings, &binding)
	}
	return bindings, nil
}

// SavedQueryPolicyEtag derives the etag from the stored bindings themselves,
// so no column has to be kept in step with them. Two policies with the same
// grants produce the same etag, which is what compare-and-swap needs: a write
// is rejected only when the grants actually moved.
func SavedQueryPolicyEtag(bindings []*storepb.SavedQueryBinding) (string, error) {
	marshalled, err := marshalSavedQueryBindings(bindings)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(marshalled))
	return hex.EncodeToString(sum[:]), nil
}

// SetSavedQueryBindings replaces a saved query's grants under compare-and-swap.
// The row is locked and its current etag compared before the write, so a
// full-replacement write can never silently undo a concurrent revocation.
// Returns ErrSavedQueryEtagMismatch when the caller's etag is stale, and
// reports false when the saved query is gone from the named project -- deleted,
// or re-parented to the default project by a purge.
func (s *Store) SetSavedQueryBindings(ctx context.Context, projectID, resourceID string, bindings []*storepb.SavedQueryBinding, expectedEtag string) (bool, error) {
	marshalled, err := marshalSavedQueryBindings(bindings)
	if err != nil {
		return false, err
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return false, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	// Scoped to the project the caller named, not the saved query's global id.
	// A project purge re-parents its members' saved queries to the default
	// project, so a write resolved against the old project must land on
	// nothing rather than on a row that has since moved out from under it.
	var currentBytes []byte
	if err := tx.QueryRowContext(ctx,
		`SELECT bindings FROM saved_query WHERE resource_id = $1 AND project = $2 FOR UPDATE`,
		resourceID, projectID).Scan(&currentBytes); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, errors.Wrapf(err, "failed to lock saved query %s", resourceID)
	}
	current, err := unmarshalSavedQueryBindings(currentBytes)
	if err != nil {
		return false, err
	}
	currentEtag, err := SavedQueryPolicyEtag(current)
	if err != nil {
		return false, err
	}
	if expectedEtag != currentEtag {
		return false, ErrSavedQueryEtagMismatch
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE saved_query SET bindings = $1 WHERE resource_id = $2 AND project = $3`,
		marshalled, resourceID, projectID); err != nil {
		return false, errors.Wrapf(err, "failed to update bindings for saved query %s", resourceID)
	}
	if err := tx.Commit(); err != nil {
		return false, errors.Wrap(err, "failed to commit transaction")
	}
	return true, nil
}

// ErrSavedQueryEtagMismatch reports that the policy moved under a
// compare-and-swap write; the caller refetches and reapplies.
var ErrSavedQueryEtagMismatch = errors.New("saved query policy etag mismatch")

// SavedQueryPrincipals returns the binding members that stand for a caller:
// themselves, plus each group they belong to. Groups are matched by reference
// and never expanded into their members, so a 1,000-member group costs the
// same as a 3-member one.
func (s *Store) SavedQueryPrincipals(ctx context.Context, workspaceID, email string) ([]string, error) {
	principals := []string{common.UserBindingPrefix + email}
	groups, err := s.GetUserGroupsSnapshot(ctx, workspaceID, common.FormatUserEmail(email))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get groups for %s", email)
	}
	for _, group := range groups {
		principals = append(principals, common.GroupBindingPrefix+strings.TrimPrefix(group, common.GroupPrefix))
	}
	return principals, nil
}

// CreateSavedQuery creates a new saved query. The insert transaction locks
// the owning project row and requires it to be active, so a create racing a
// project purge fails cleanly instead of as an FK violation — the parent
// fence for a new child row that cannot be locked in advance.
func (s *Store) CreateSavedQuery(ctx context.Context, create *SavedQueryMessage) (*SavedQueryMessage, error) {
	payloadStr, err := protojson.Marshal(&storepb.SavedQueryPayload{Database: create.Database})
	if err != nil {
		return nil, err
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	// A saved query is a new child row, so it takes the "requires an active
	// project" fence: lock the parent and refuse if it is archived or purged.
	if err := lockActiveProject(ctx, tx, create.ProjectID); err != nil {
		return nil, err
	}

	query, args, err := qb.Q().Space(`
		INSERT INTO saved_query (
			creator,
			project,
			name,
			statement,
			folder,
			payload
		)
		VALUES (?, ?, ?, ?, ?, ?)
		RETURNING resource_id, created_at, updated_at, OCTET_LENGTH(statement)
	`, create.Creator, create.ProjectID, create.Title, create.Statement, create.Folder, payloadStr).ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	if err := tx.QueryRowContext(ctx, query, args...).Scan(
		&create.ResourceID,
		&create.CreatedAt,
		&create.UpdatedAt,
		&create.Size,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}

	return create, nil
}

// PatchSavedQuery updates a sheet.
func (s *Store) PatchSavedQuery(ctx context.Context, patch *PatchSavedQueryMessage) error {
	set := qb.Q()
	set.Comma("updated_at = ?", time.Now())
	if v := patch.Title; v != nil {
		set.Comma("name = ?", *v)
	}
	if v := patch.Statement; v != nil {
		set.Comma("statement = ?", *v)
	}
	if v := patch.Folder; v != nil {
		set.Comma("folder = ?", *v)
	}
	if v := patch.Database; v != nil {
		if *v == "" {
			set.Comma("payload = payload - 'database'")
		} else {
			set.Comma("payload = payload || jsonb_build_object('database', ?::text)", *v)
		}
	}

	query, args, err := qb.Q().Space("UPDATE saved_query SET ? WHERE resource_id = ?", set, patch.ResourceID).ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return err
	}
	return nil
}

// DeleteSavedQuery deletes an existing saved query by resource ID. Star rows
// are deleted first, in full primary-key order — explicitly, not via the FK
// cascade (which would lock the parent first) — matching the star and purge
// lock order so a delete racing a star toggle or a purge cannot deadlock.
func (s *Store) DeleteSavedQuery(ctx context.Context, resourceID string) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM saved_query_star
		WHERE (saved_query, principal) IN (
			SELECT saved_query, principal
			FROM saved_query_star
			WHERE saved_query = $1
			ORDER BY saved_query, principal
			FOR UPDATE
		)
	`, resourceID); err != nil {
		return errors.Wrapf(err, "failed to delete stars for saved query %s", resourceID)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM saved_query WHERE resource_id = $1`, resourceID); err != nil {
		return err
	}
	return tx.Commit()
}

// SetSavedQueryStar stars or unstars a saved query for a principal, and
// reports whether the saved query was still there to star. Lock order per the
// store row-lock rules: toggling or removing an existing star locks that child
// row directly; only the first star for a (query, principal) inserts a child
// that cannot be locked in advance — that case alone takes the parent fence
// (lock the saved_query row), and its inserted key is novel so it never
// contends with a purge's existing-child locks. The parent fence is also the
// only path that can observe a concurrent delete, which it reports as false
// rather than an error so the caller decides what a vanished row means.
func (s *Store) SetSavedQueryStar(ctx context.Context, savedQueryResourceID, principal string, starred bool) (bool, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return false, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	var exists bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM saved_query_star
			WHERE saved_query = $1 AND principal = $2
			FOR UPDATE
		)
	`, savedQueryResourceID, principal).Scan(&exists); err != nil {
		return false, errors.Wrap(err, "failed to lock star row")
	}

	switch {
	case starred && !exists:
		var one int
		if err := tx.QueryRowContext(ctx, `
			SELECT 1
			FROM saved_query
			WHERE resource_id = $1
			FOR UPDATE
		`, savedQueryResourceID).Scan(&one); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return false, nil
			}
			return false, errors.Wrapf(err, "failed to lock saved query %s", savedQueryResourceID)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO saved_query_star (saved_query, principal)
			VALUES ($1, $2)
			ON CONFLICT (saved_query, principal) DO NOTHING
		`, savedQueryResourceID, principal); err != nil {
			return false, errors.Wrap(err, "failed to insert star")
		}
	case !starred && exists:
		if _, err := tx.ExecContext(ctx, `
			DELETE FROM saved_query_star
			WHERE saved_query = $1 AND principal = $2
		`, savedQueryResourceID, principal); err != nil {
			return false, errors.Wrap(err, "failed to delete star")
		}
	default:
		// Already in the requested state: an unstar of a row that is gone
		// needs no parent, so it is reported as applied.
	}

	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}

// BatchUpdateSavedQueryFolder re-files the given saved queries into folder.
// The rows are locked in full primary-key order (resource_id) before the
// update, so two overlapping batches can never lock in opposing orders; a
// row deleted mid-batch simply updates zero rows.
func (s *Store) BatchUpdateSavedQueryFolder(ctx context.Context, resourceIDs []string, folder string) (int, error) {
	if len(resourceIDs) == 0 {
		return 0, nil
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return 0, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT resource_id
		FROM saved_query
		WHERE resource_id = ANY($1)
		ORDER BY resource_id
		FOR UPDATE
	`, resourceIDs)
	if err != nil {
		return 0, errors.Wrap(err, "failed to lock saved queries")
	}
	defer rows.Close()
	var locked []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		locked = append(locked, id)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	// Free the cursor before the UPDATE below: statements on the same
	// transaction cannot interleave with an open one.
	rows.Close()

	if len(locked) > 0 {
		if _, err := tx.ExecContext(ctx, `
			UPDATE saved_query
			SET folder = $1, updated_at = now()
			WHERE resource_id = ANY($2)
		`, folder, locked); err != nil {
			return 0, errors.Wrap(err, "failed to update folders")
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, errors.Wrap(err, "failed to commit transaction")
	}
	return len(locked), nil
}

// ListSavedQueryFolderPaths returns the distinct folder paths of the saved
// queries in a project that match filterQ. A non-nil creator restricts the
// paths to that creator's rows, which is how the API layer applies the same
// read rule the rows themselves carry: your own always, everyone's only with
// the admin backstop.
// ListSavedQueryFolderPaths returns the distinct folder paths of the saved
// queries a caller can read, narrowed by filterQ. Access is the same predicate
// SearchSavedQueries applies, and for the same reason: a folder is only useful
// if its contents are reachable. Scoping these to the caller's *own* rows
// instead would hide a shared saved query filed by its creator — the tree is
// seeded from here, and a row whose folder has no node can never be expanded
// into.
func (s *Store) ListSavedQueryFolderPaths(ctx context.Context, projectID string, principalEmail string, accessMembers []string, filterQ *qb.Query) ([]string, error) {
	q := qb.Q().Space(`
		SELECT DISTINCT folder
		FROM saved_query
		WHERE TRUE`)
	q.And("saved_query.project = ?", projectID)
	q.And("saved_query.folder <> ''")
	if accessMembers != nil {
		access := qb.Q().Space("saved_query.creator = ?", principalEmail)
		for _, member := range accessMembers {
			probe, err := bindingProbe(member)
			if err != nil {
				return nil, err
			}
			access.Or("saved_query.bindings @> ?", probe)
		}
		q.And("(?)", access)
	}
	if filterQ != nil {
		q.And("?", filterQ)
	}
	q.Space("ORDER BY folder")

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var folders []string
	for rows.Next() {
		var folder string
		if err := rows.Scan(&folder); err != nil {
			return nil, err
		}
		folders = append(folders, folder)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return folders, nil
}

// NormalizeSavedQueryFolder puts a client-supplied folder path into the one
// form that is stored and matched. Boundary slashes are trimmed, so "/team/"
// and "team" name the same folder, and an empty path segment is rejected
// rather than stored: every write path and the `folder ==` filter share this
// function, so a folder that can be stored is always a folder that can be
// found. "" means unfiled.
func NormalizeSavedQueryFolder(folder string) (string, error) {
	normalized := strings.Trim(folder, "/")
	if strings.Contains(normalized, "//") {
		return "", errors.Errorf("invalid folder %q: empty path segment", folder)
	}
	return normalized, nil
}

func GetSearchSavedQueryFilter(ctx context.Context, s *Store, workspaceID, caller string, filter string, allowTitleContains bool) (*qb.Query, error) {
	if filter == "" {
		return nil, nil
	}

	e, err := cel.NewEnv()
	if err != nil {
		return nil, errors.New("failed to create cel env")
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return nil, errors.Errorf("failed to parse filter %v, error: %v", filter, iss.String())
	}

	var getFilter func(expr celast.Expr) (*qb.Query, error)

	getUserID := func(name string) (string, error) {
		creatorEmail := strings.TrimPrefix(name, "users/")
		if creatorEmail == "" {
			return "", errors.New("invalid empty creator identifier")
		}
		user, err := s.GetUserByEmail(ctx, creatorEmail)
		if err != nil {
			return "", errors.Errorf("failed to get user: %v", err)
		}
		if user == nil {
			return "", errors.Errorf("user with email %s not found", creatorEmail)
		}
		return user.Email, nil
	}
	getSavedQueryID := func(name string) (string, error) {
		_, savedQueryID, err := common.GetProjectIDSavedQueryID(name)
		if err != nil {
			return "", errors.Errorf("invalid saved query name %q", name)
		}
		return savedQueryID, nil
	}

	parseToSQL := func(variable, value any) (*qb.Query, error) {
		switch variable {
		case "name":
			name, ok := value.(string)
			if !ok {
				return nil, errors.Errorf("invalid name value %q", value)
			}
			savedQueryID, err := getSavedQueryID(name)
			if err != nil {
				return nil, err
			}
			return qb.Q().Space("saved_query.resource_id = ?", savedQueryID), nil
		case "creator":
			creator, ok := value.(string)
			if !ok {
				return nil, errors.Errorf("invalid creator value %v, expect a string", value)
			}
			userID, err := getUserID(creator)
			if err != nil {
				return nil, err
			}
			return qb.Q().Space("saved_query.creator = ?", userID), nil
		case "starred":
			if starred, ok := value.(bool); ok {
				if starred {
					return qb.Q().Space("EXISTS (SELECT 1 FROM saved_query_star WHERE saved_query_star.saved_query = saved_query.resource_id AND saved_query_star.principal = ?)", caller), nil
				}
				return qb.Q().Space("NOT EXISTS (SELECT 1 FROM saved_query_star WHERE saved_query_star.saved_query = saved_query.resource_id AND saved_query_star.principal = ?)", caller), nil
			}
			return nil, errors.Errorf("invalid starred value %v, expect true or false", value)
		case "shared":
			// Reached through a binding, which is narrower than "somebody else
			// created it": an admin sees saved queries nobody shared with them,
			// and those must not show up in a Shared view.
			shared, ok := value.(bool)
			if !ok {
				return nil, errors.Errorf("invalid shared value %v, expect true or false", value)
			}
			principals, err := s.SavedQueryPrincipals(ctx, workspaceID, caller)
			if err != nil {
				return nil, err
			}
			granted := qb.Q().Space("FALSE")
			for _, principal := range principals {
				probe, err := bindingProbe(principal)
				if err != nil {
					return nil, err
				}
				granted.Or("saved_query.bindings @> ?", probe)
			}
			if shared {
				return qb.Q().Space("(saved_query.creator <> ? AND (?))", caller, granted), nil
			}
			return qb.Q().Space("(NOT (saved_query.creator <> ? AND (?)))", caller, granted), nil
		case "folder":
			folder, ok := value.(string)
			if !ok {
				return nil, errors.Errorf("invalid folder value %q", value)
			}
			folder, err := NormalizeSavedQueryFolder(folder)
			if err != nil {
				return nil, err
			}
			return qb.Q().Space("saved_query.folder = ?", folder), nil
		default:
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
				if expr.AsCall().Target().Kind() != celast.IdentKind {
					return nil, errors.Errorf(`invalid args for %q`, celoverloads.Contains)
				}
				variable := expr.AsCall().Target().AsIdent()
				args := expr.AsCall().Args()
				if len(args) != 1 || args[0].Kind() != celast.LiteralKind {
					return nil, errors.Errorf(`invalid args for %q`, variable)
				}
				value := args[0].AsLiteral().Value()
				strValue, ok := value.(string)
				if !ok {
					return nil, errors.Errorf("expect string, got %T, hint: filter literals should be string", value)
				}
				if strValue == "" {
					return nil, errors.Errorf(`empty value for %q`, variable)
				}

				switch variable {
				case "title":
					if !allowTitleContains {
						return nil, errors.Errorf("unsupport variable %q", variable)
					}
					return qb.Q().Space("LOWER(saved_query.name) LIKE ? ESCAPE '\\'", "%"+escapeLikePattern(strings.ToLower(strValue))+"%"), nil
				default:
					return nil, errors.Errorf("unsupport variable %q", variable)
				}
			case celoperators.NotEquals:
				variable, value := getVariableAndValueFromExpr(expr)
				if variable != "creator" {
					return nil, errors.Errorf(`only "creator" support "!=" operator`)
				}
				creator, ok := value.(string)
				if !ok {
					return nil, errors.Errorf("invalid creator value %v, expect a string", value)
				}
				userID, err := getUserID(creator)
				if err != nil {
					return nil, err
				}
				return qb.Q().Space("saved_query.creator != ?", userID), nil
			case celoperators.In:
				variable, value := getVariableAndValueFromExpr(expr)
				rawList, ok := value.([]any)
				if !ok {
					return nil, errors.Errorf("invalid %s value %q", variable, value)
				}
				if len(rawList) == 0 {
					return nil, errors.Errorf("empty %s filter", variable)
				}
				switch variable {
				case "name":
					savedQueryIDs := []string{}
					for _, raw := range rawList {
						name, ok := raw.(string)
						if !ok {
							return nil, errors.Errorf("invalid name value %q", raw)
						}
						savedQueryID, err := getSavedQueryID(name)
						if err != nil {
							return nil, err
						}
						savedQueryIDs = append(savedQueryIDs, savedQueryID)
					}
					return qb.Q().Space("saved_query.resource_id = ANY(?)", savedQueryIDs), nil
				default:
					return nil, errors.Errorf(`only "name" supports "in" filter`)
				}
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

func escapeLikePattern(pattern string) string {
	return strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	).Replace(pattern)
}

func GetListSavedQueryFilter(filter string) (*qb.Query, error) {
	if filter == "" {
		return nil, nil
	}

	e, err := cel.NewEnv()
	if err != nil {
		return nil, errors.New("failed to create cel env")
	}
	ast, iss := e.Parse(filter)
	if iss != nil {
		return nil, errors.Errorf("failed to parse filter %v, error: %v", filter, iss.String())
	}

	var getFilter func(expr celast.Expr) (*qb.Query, error)
	getFilter = func(expr celast.Expr) (*qb.Query, error) {
		if expr.Kind() != celast.CallKind {
			return nil, errors.Errorf("unexpected expr kind %v", expr.Kind())
		}
		q := qb.Q()
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
		case celoperators.Equals, celoperators.NotEquals:
			variable, value := getVariableAndValueFromExpr(expr)
			if variable != "creator" {
				return nil, errors.Errorf("unsupported variable %q", variable)
			}
			creator, ok := value.(string)
			if !ok {
				return nil, errors.Errorf("invalid creator value %q", value)
			}
			creatorEmail := strings.TrimPrefix(creator, "users/")
			if creatorEmail == "" {
				return nil, errors.New("invalid empty creator identifier")
			}
			if functionName == celoperators.Equals {
				return qb.Q().Space("saved_query.creator = ?", creatorEmail), nil
			}
			return qb.Q().Space("saved_query.creator != ?", creatorEmail), nil
		default:
			return nil, errors.Errorf("unexpected function %v", functionName)
		}
	}

	q, err := getFilter(ast.NativeRep().Expr())
	if err != nil {
		return nil, err
	}
	return qb.Q().Space("(?)", q), nil
}
