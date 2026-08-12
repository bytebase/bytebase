package store

import (
	"context"
	"database/sql"
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

	// Output only fields
	Size      int64
	CreatedAt time.Time
	UpdatedAt time.Time
	Starred   bool
	Folders   []string
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
			saved_query.name,
			%s,
			OCTET_LENGTH(saved_query.statement),
			COALESCE(saved_query_organizer.payload, '{}')
		FROM saved_query
		LEFT JOIN saved_query_organizer ON saved_query_organizer.saved_query = saved_query.resource_id AND saved_query_organizer.principal = ?
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
		var payloadBytes, organizerPayloadBytes []byte
		if err := rows.Scan(
			&sheet.ResourceID,
			&sheet.Creator,
			&sheet.CreatedAt,
			&sheet.UpdatedAt,
			&sheet.ProjectID,
			&payloadBytes,
			&sheet.Title,
			&sheet.Statement,
			&sheet.Size,
			&organizerPayloadBytes,
		); err != nil {
			return nil, err
		}

		var payload storepb.SavedQueryPayload
		if err := common.ProtojsonUnmarshaler.Unmarshal(payloadBytes, &payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal saved query payload")
		}
		sheet.Database = payload.Database

		var organizerPayload storepb.SavedQueryOrganizerPayload
		if err := common.ProtojsonUnmarshaler.Unmarshal(organizerPayloadBytes, &organizerPayload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal saved query organizer payload")
		}
		sheet.Folders = organizerPayload.Folders
		sheet.Starred = organizerPayload.Starred

		sheets = append(sheets, &sheet)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sheets, nil
}

// ListSavedQueryOrganizers returns the caller's savedQuery organizers.
func (s *Store) ListSavedQueryOrganizers(ctx context.Context, find *FindSavedQueryMessage) ([]*SavedQueryOrganizerMessage, error) {
	if len(find.ProjectIDs) == 0 && find.Workspace == "" {
		return nil, errors.Errorf("empty project filter")
	}
	if find.PrincipalEmail == "" {
		return nil, errors.Errorf("empty principal")
	}
	q := qb.Q().Space(`
		SELECT
			saved_query.resource_id,
			saved_query.creator,
			saved_query_organizer.principal,
			saved_query_organizer.payload
		FROM saved_query_organizer
		JOIN savedQuery ON saved_query.resource_id = saved_query_organizer.saved_query
		WHERE saved_query_organizer.principal = ?`, find.PrincipalEmail)

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

	q.Space("ORDER BY saved_query.project, saved_query.resource_id")

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var savedQueryOrganizers []*SavedQueryOrganizerMessage
	for rows.Next() {
		var savedQueryOrganizer SavedQueryOrganizerMessage
		var payloadBytes []byte
		if err := rows.Scan(
			&savedQueryOrganizer.SavedQueryResourceID,
			&savedQueryOrganizer.SavedQueryCreator,
			&savedQueryOrganizer.Principal,
			&payloadBytes,
		); err != nil {
			return nil, err
		}

		var payload storepb.SavedQueryOrganizerPayload
		if err := common.ProtojsonUnmarshaler.Unmarshal(payloadBytes, &payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal savedQuery organizer payload")
		}
		savedQueryOrganizer.Payload = &payload

		savedQueryOrganizers = append(savedQueryOrganizers, &savedQueryOrganizer)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return savedQueryOrganizers, nil
}

// CreateSavedQuery creates a new saved query.
func (s *Store) CreateSavedQuery(ctx context.Context, create *SavedQueryMessage) (*SavedQueryMessage, error) {
	payloadStr, err := protojson.Marshal(&storepb.SavedQueryPayload{Database: create.Database})
	if err != nil {
		return nil, err
	}
	q := qb.Q().Space(`
		INSERT INTO saved_query (
			creator,
			project,
			name,
			statement,
			payload
		)
		VALUES (?, ?, ?, ?, ?)
		RETURNING resource_id, created_at, updated_at, OCTET_LENGTH(statement)
	`, create.Creator, create.ProjectID, create.Title, create.Statement, payloadStr)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
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

// DeleteSavedQuery deletes an existing saved query by resource ID.
// The saved_query_organizer rows are cascade-deleted via FK on saved_query(resource_id).
func (s *Store) DeleteSavedQuery(ctx context.Context, resourceID string) error {
	q := qb.Q().Space(`DELETE FROM saved_query WHERE resource_id = ?`, resourceID)
	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return err
	}
	return nil
}

// SavedQueryOrganizerMessage is the store message for savedQuery organizer.
type SavedQueryOrganizerMessage struct {
	SavedQueryResourceID string
	SavedQueryCreator    string
	Principal            string
	Payload              *storepb.SavedQueryOrganizerPayload
}

// BatchUpdateSavedQueryOrganizerPatch is the patch for updating savedQuery organizers in batch.
type BatchUpdateSavedQueryOrganizerPatch struct {
	SavedQueryResourceIDs []string
	Principal             string
	Starred               *bool
	Folders               *[]string
}

func (s *Store) GetSavedQueryOrganizer(ctx context.Context, savedQueryResourceID string, principal string) (*SavedQueryOrganizerMessage, error) {
	q := qb.Q().Space(`
		SELECT
			payload
		FROM saved_query_organizer
		WHERE saved_query = ? AND principal = ?
	`, savedQueryResourceID, principal)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	savedQueryOrganizer := SavedQueryOrganizerMessage{
		SavedQueryResourceID: savedQueryResourceID,
		Principal:            principal,
		Payload:              &storepb.SavedQueryOrganizerPayload{},
	}
	var payload []byte
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&payload,
	); err != nil {
		if err == sql.ErrNoRows {
			return &savedQueryOrganizer, nil
		}
		return nil, errors.Wrapf(err, "failed to scan")
	}
	workSheetPayload := &storepb.SavedQueryOrganizerPayload{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(payload, workSheetPayload); err != nil {
		return nil, err
	}
	savedQueryOrganizer.Payload = workSheetPayload

	return &savedQueryOrganizer, nil
}

// UpsertSavedQueryOrganizer upserts a new SheetOrganizerMessage.
func (s *Store) UpsertSavedQueryOrganizer(ctx context.Context, patch *SavedQueryOrganizerMessage) (*SavedQueryOrganizerMessage, error) {
	payloadStr, err := protojson.Marshal(patch.Payload)
	if err != nil {
		return nil, err
	}
	q := qb.Q().Space(`
	  INSERT INTO saved_query_organizer (
			savedQuery,
			principal,
			payload
		)
		VALUES (?, ?, ?)
		ON CONFLICT(saved_query, principal) DO UPDATE SET
			payload = EXCLUDED.payload
		RETURNING
			savedQuery,
			principal,
			payload
	`, patch.SavedQueryResourceID, patch.Principal, payloadStr)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var savedQueryOrganizer SavedQueryOrganizerMessage
	var payload []byte
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&savedQueryOrganizer.SavedQueryResourceID,
		&savedQueryOrganizer.Principal,
		&payload,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}
	workSheetPayload := &storepb.SavedQueryOrganizerPayload{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(payload, workSheetPayload); err != nil {
		return nil, err
	}
	savedQueryOrganizer.Payload = workSheetPayload

	return &savedQueryOrganizer, nil
}

// BatchUpdateSavedQueryOrganizer updates savedQuery organizer payload fields in batch.
func (s *Store) BatchUpdateSavedQueryOrganizer(ctx context.Context, patch *BatchUpdateSavedQueryOrganizerPatch) ([]*SavedQueryOrganizerMessage, error) {
	if len(patch.SavedQueryResourceIDs) == 0 {
		return nil, nil
	}
	if patch.Principal == "" {
		return nil, errors.New("principal is empty")
	}
	if patch.Starred == nil && patch.Folders == nil {
		return nil, errors.New("empty savedQuery organizer patch")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	insertQuery, insertArgs, err := qb.Q().Space(`
		INSERT INTO saved_query_organizer (
			savedQuery,
			principal,
			payload
		)
		SELECT saved_query.resource_id, ?, '{}'::jsonb
		FROM saved_query
		WHERE saved_query.resource_id = ANY(?)
		ON CONFLICT (saved_query, principal) DO NOTHING
	`, patch.Principal, patch.SavedQueryResourceIDs).ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build savedQuery organizer insert query")
	}
	if _, err := tx.ExecContext(ctx, insertQuery, insertArgs...); err != nil {
		return nil, errors.Wrap(err, "failed to insert missing savedQuery organizers")
	}

	payloadExpr := "payload"
	var updateArgs []any
	if patch.Starred != nil {
		payloadExpr = fmt.Sprintf("jsonb_set(%s, '{starred}', to_jsonb(?::boolean), true)", payloadExpr)
		updateArgs = append(updateArgs, *patch.Starred)
	}
	if patch.Folders != nil {
		folderBytes, err := json.Marshal(*patch.Folders)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal folders")
		}
		payloadExpr = fmt.Sprintf("jsonb_set(%s, '{folders}', ?::jsonb, true)", payloadExpr)
		updateArgs = append(updateArgs, string(folderBytes))
	}
	updateArgs = append(updateArgs, patch.Principal, patch.SavedQueryResourceIDs)

	updateQuery, args, err := qb.Q().Space(fmt.Sprintf(`
		UPDATE saved_query_organizer
		SET payload = %s
		WHERE principal = ? AND savedQuery = ANY(?)
		RETURNING savedQuery, principal, payload
	`, payloadExpr), updateArgs...).ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build savedQuery organizer update query")
	}
	rows, err := tx.QueryContext(ctx, updateQuery, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update savedQuery organizers")
	}
	defer rows.Close()

	var savedQueryOrganizers []*SavedQueryOrganizerMessage
	for rows.Next() {
		var savedQueryOrganizer SavedQueryOrganizerMessage
		var payload []byte
		if err := rows.Scan(
			&savedQueryOrganizer.SavedQueryResourceID,
			&savedQueryOrganizer.Principal,
			&payload,
		); err != nil {
			return nil, err
		}
		workSheetPayload := &storepb.SavedQueryOrganizerPayload{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, workSheetPayload); err != nil {
			return nil, err
		}
		savedQueryOrganizer.Payload = workSheetPayload
		savedQueryOrganizers = append(savedQueryOrganizers, &savedQueryOrganizer)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}
	return savedQueryOrganizers, nil
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
				return qb.Q().Space("saved_query.resource_id IN (SELECT saved_query FROM saved_query_organizer WHERE principal = ? AND (payload->>'starred')::boolean = ?)", caller, starred), nil
			}
			return qb.Q().Space("TRUE"), nil
		case "folder":
			folder, ok := value.(string)
			if !ok {
				return nil, errors.Errorf("invalid folder value %q", value)
			}
			folder = strings.Trim(folder, "/")
			if folder == "" {
				return qb.Q().Space("COALESCE(jsonb_array_length(saved_query_organizer.payload->'folders'), 0) = 0"), nil
			}
			q := qb.Q()
			segments := strings.Split(folder, "/")
			for i, segment := range segments {
				if segment == "" {
					return nil, errors.Errorf("invalid folder %q", value)
				}
				q.And(fmt.Sprintf("saved_query_organizer.payload->'folders'->>%d = ?", i), segment)
			}
			q.And("jsonb_array_length(saved_query_organizer.payload->'folders') = ?", len(segments))
			return qb.Q().Space("(?)", q), nil
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
