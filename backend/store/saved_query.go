package store

import (
	"context"
	"database/sql"
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
			%s,
			OCTET_LENGTH(saved_query.statement),
			EXISTS (SELECT 1 FROM saved_query_star WHERE saved_query_star.saved_query = saved_query.resource_id AND saved_query_star.principal = ?)
		FROM saved_query
		WHERE TRUE`, statementField), find.PrincipalEmail)

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
		var payloadBytes []byte
		if err := rows.Scan(
			&sheet.ResourceID,
			&sheet.Creator,
			&sheet.CreatedAt,
			&sheet.UpdatedAt,
			&sheet.ProjectID,
			&payloadBytes,
			&sheet.Folder,
			&sheet.Title,
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

		sheets = append(sheets, &sheet)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sheets, nil
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

	// Callers reach this only after the API layer has resolved the parent
	// project, so a missing or deleted project here means it was purged
	// mid-request. That is a lost race, not a caller mistake.
	var deleted bool
	if err := tx.QueryRowContext(ctx, `
		SELECT deleted
		FROM project
		WHERE resource_id = $1
		FOR UPDATE
	`, create.ProjectID).Scan(&deleted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.Errorf("project %s was deleted while creating the saved query", create.ProjectID)
		}
		return nil, errors.Wrapf(err, "failed to lock project %s", create.ProjectID)
	}
	if deleted {
		return nil, errors.Errorf("project %s was deleted while creating the saved query", create.ProjectID)
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
func (s *Store) ListSavedQueryFolderPaths(ctx context.Context, projectID string, creator *string, filterQ *qb.Query) ([]string, error) {
	q := qb.Q().Space(`
		SELECT DISTINCT folder
		FROM saved_query
		WHERE TRUE`)
	q.And("saved_query.project = ?", projectID)
	q.And("saved_query.folder <> ''")
	if creator != nil {
		q.And("saved_query.creator = ?", *creator)
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

func GetSearchSavedQueryFilter(ctx context.Context, s *Store, caller string, filter string, allowTitleContains bool) (*qb.Query, error) {
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
			userID, err := getUserID(value.(string))
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
			return qb.Q().Space("TRUE"), nil
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
				userID, err := getUserID(value.(string))
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
