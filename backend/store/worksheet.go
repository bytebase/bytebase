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

// WorkSheetVisibility is the visibility of a sheet.
type WorkSheetVisibility string

const (
	// PrivateWorkSheet is the sheet visibility for PRIVATE. Only sheet OWNER can read/write.
	PrivateWorkSheet WorkSheetVisibility = "PRIVATE"
	// ProjectReadWorkSheet is the sheet visibility for PROJECT. Both sheet OWNER and project OWNER can read/write, and project DEVELOPER can read.
	ProjectReadWorkSheet WorkSheetVisibility = "PROJECT_READ"
	// ProjectWriteWorkSheet is the sheet visibility for PROJECT. Both sheet OWNER and project OWNER can read/write, and project DEVELOPER can read.
	ProjectWriteWorkSheet WorkSheetVisibility = "PROJECT_WRITE"
)

// WorkSheetMessage is the message for a sheet.
type WorkSheetMessage struct {
	ProjectID  string
	ResourceID string
	// The DatabaseUID is optional.
	// If not NULL, the sheet ProjectID should always be equal to the id of the database related project.
	// A project must remove all linked sheets for a particular database before that database can be transferred to a different project.
	InstanceID   *string
	DatabaseName *string

	Creator string

	Title      string
	Statement  string
	Visibility WorkSheetVisibility

	// Output only fields
	Size      int64
	CreatedAt time.Time
	UpdatedAt time.Time
	Starred   bool
	Folders   []string
}

// FindWorkSheetMessage is the API message for finding sheets.
type FindWorkSheetMessage struct {
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

// PatchWorkSheetMessage is the message to patch a sheet.
type PatchWorkSheetMessage struct {
	ResourceID   string
	Title        *string
	Statement    *string
	Visibility   *string
	InstanceID   *string
	DatabaseName *string
}

// GetWorkSheet gets a sheet.
func (s *Store) GetWorkSheet(ctx context.Context, find *FindWorkSheetMessage) (*WorkSheetMessage, error) {
	sheets, err := s.ListWorkSheets(ctx, find)
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

// ListWorkSheets returns a list of sheets.
func (s *Store) ListWorkSheets(ctx context.Context, find *FindWorkSheetMessage) ([]*WorkSheetMessage, error) {
	if len(find.ProjectIDs) == 0 && find.Workspace == "" {
		return nil, errors.Errorf("empty project filter")
	}
	statementField := fmt.Sprintf("LEFT(worksheet.statement, %d)", common.MaxSheetSize)
	if find.LoadFull {
		statementField = "worksheet.statement"
	}

	q := qb.Q().Space(fmt.Sprintf(`
		SELECT
			worksheet.resource_id,
			worksheet.creator,
			worksheet.created_at,
			worksheet.updated_at,
			worksheet.project,
			worksheet.instance,
			worksheet.db_name,
			worksheet.name,
			%s,
			worksheet.visibility,
			OCTET_LENGTH(worksheet.statement),
			COALESCE(worksheet_organizer.payload, '{}')
		FROM worksheet
		LEFT JOIN worksheet_organizer ON worksheet_organizer.worksheet = worksheet.resource_id AND worksheet_organizer.principal = ?
		WHERE TRUE`, statementField), find.PrincipalEmail)

	if find.Workspace != "" {
		q.And("EXISTS (SELECT 1 FROM project WHERE project.resource_id = worksheet.project AND project.workspace = ? AND project.deleted = FALSE)", find.Workspace)
	}
	if len(find.ProjectIDs) == 1 {
		q.And("worksheet.project = ?", find.ProjectIDs[0])
	} else if len(find.ProjectIDs) > 1 {
		q.And("worksheet.project = ANY(?)", find.ProjectIDs)
	}

	if filterQ := find.FilterQ; filterQ != nil {
		q.And("?", filterQ)
	}

	if v := find.ResourceID; v != nil {
		q.And("worksheet.resource_id = ?", *v)
	}

	q.Space("ORDER BY worksheet.name, worksheet.resource_id")
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

	var sheets []*WorkSheetMessage
	for rows.Next() {
		var sheet WorkSheetMessage
		var instanceID, databaseName sql.NullString
		var payloadBytes []byte
		if err := rows.Scan(
			&sheet.ResourceID,
			&sheet.Creator,
			&sheet.CreatedAt,
			&sheet.UpdatedAt,
			&sheet.ProjectID,
			&instanceID,
			&databaseName,
			&sheet.Title,
			&sheet.Statement,
			&sheet.Visibility,
			&sheet.Size,
			&payloadBytes,
		); err != nil {
			return nil, err
		}
		if instanceID.Valid {
			sheet.InstanceID = &instanceID.String
		}
		if databaseName.Valid {
			sheet.DatabaseName = &databaseName.String
		}

		var payload storepb.WorkSheetOrganizerPayload
		if err := common.ProtojsonUnmarshaler.Unmarshal(payloadBytes, &payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal worksheet organizer payload")
		}
		sheet.Folders = payload.Folders
		sheet.Starred = payload.Starred

		sheets = append(sheets, &sheet)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sheets, nil
}

// ListWorksheetOrganizers returns the caller's worksheet organizers.
func (s *Store) ListWorksheetOrganizers(ctx context.Context, find *FindWorkSheetMessage) ([]*WorksheetOrganizerMessage, error) {
	if len(find.ProjectIDs) == 0 && find.Workspace == "" {
		return nil, errors.Errorf("empty project filter")
	}
	if find.PrincipalEmail == "" {
		return nil, errors.Errorf("empty principal")
	}
	q := qb.Q().Space(`
		SELECT
			worksheet.resource_id,
			worksheet.creator,
			worksheet.visibility,
			worksheet_organizer.principal,
			worksheet_organizer.payload
		FROM worksheet_organizer
		JOIN worksheet ON worksheet.resource_id = worksheet_organizer.worksheet
		WHERE worksheet_organizer.principal = ?`, find.PrincipalEmail)

	if find.Workspace != "" {
		q.And("EXISTS (SELECT 1 FROM project WHERE project.resource_id = worksheet.project AND project.workspace = ? AND project.deleted = FALSE)", find.Workspace)
	}
	if len(find.ProjectIDs) == 1 {
		q.And("worksheet.project = ?", find.ProjectIDs[0])
	} else if len(find.ProjectIDs) > 1 {
		q.And("worksheet.project = ANY(?)", find.ProjectIDs)
	}

	if filterQ := find.FilterQ; filterQ != nil {
		q.And("?", filterQ)
	}

	q.Space("ORDER BY worksheet.project, worksheet.resource_id")

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var worksheetOrganizers []*WorksheetOrganizerMessage
	for rows.Next() {
		var worksheetOrganizer WorksheetOrganizerMessage
		var visibility string
		var payloadBytes []byte
		if err := rows.Scan(
			&worksheetOrganizer.WorksheetResourceID,
			&worksheetOrganizer.WorksheetCreator,
			&visibility,
			&worksheetOrganizer.Principal,
			&payloadBytes,
		); err != nil {
			return nil, err
		}
		worksheetOrganizer.WorksheetVisibility = WorkSheetVisibility(visibility)

		var payload storepb.WorkSheetOrganizerPayload
		if err := common.ProtojsonUnmarshaler.Unmarshal(payloadBytes, &payload); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal worksheet organizer payload")
		}
		worksheetOrganizer.Payload = &payload

		worksheetOrganizers = append(worksheetOrganizers, &worksheetOrganizer)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return worksheetOrganizers, nil
}

// CreateWorkSheet creates a new sheet.
func (s *Store) CreateWorkSheet(ctx context.Context, create *WorkSheetMessage) (*WorkSheetMessage, error) {
	q := qb.Q().Space(`
		INSERT INTO worksheet (
			creator,
			project,
			instance,
			db_name,
			name,
			statement,
			visibility,
			payload
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, '{}')
		RETURNING resource_id, created_at, updated_at, OCTET_LENGTH(statement)
	`, create.Creator, create.ProjectID, create.InstanceID, create.DatabaseName, create.Title, create.Statement, create.Visibility)

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

// PatchWorkSheet updates a sheet.
func (s *Store) PatchWorkSheet(ctx context.Context, patch *PatchWorkSheetMessage) error {
	set := qb.Q()
	set.Comma("updated_at = ?", time.Now())
	if v := patch.Title; v != nil {
		set.Comma("name = ?", *v)
	}
	if v := patch.Statement; v != nil {
		set.Comma("statement = ?", *v)
	}
	if v := patch.Visibility; v != nil {
		set.Comma("visibility = ?", *v)
	}
	if v := patch.InstanceID; v != nil {
		if *v == "" {
			set.Comma("instance = ?", nil)
		} else {
			set.Comma("instance = ?", *v)
		}
	}
	if v := patch.DatabaseName; v != nil {
		if *v == "" {
			set.Comma("db_name = ?", nil)
		} else {
			set.Comma("db_name = ?", *v)
		}
	}

	query, args, err := qb.Q().Space("UPDATE worksheet SET ? WHERE resource_id = ?", set, patch.ResourceID).ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return err
	}
	return nil
}

// DeleteWorkSheet deletes an existing sheet by resource ID.
// The worksheet_organizer rows are cascade-deleted via FK on worksheet(resource_id).
func (s *Store) DeleteWorkSheet(ctx context.Context, resourceID string) error {
	q := qb.Q().Space(`DELETE FROM worksheet WHERE resource_id = ?`, resourceID)
	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return err
	}
	return nil
}

// WorksheetOrganizerMessage is the store message for worksheet organizer.
type WorksheetOrganizerMessage struct {
	WorksheetResourceID string
	WorksheetCreator    string
	WorksheetVisibility WorkSheetVisibility
	Principal           string
	Payload             *storepb.WorkSheetOrganizerPayload
}

// BatchUpdateWorksheetOrganizerPatch is the patch for updating worksheet organizers in batch.
type BatchUpdateWorksheetOrganizerPatch struct {
	WorksheetResourceIDs []string
	Principal            string
	Starred              *bool
	Folders              *[]string
}

func (s *Store) GetWorksheetOrganizer(ctx context.Context, worksheetResourceID string, principal string) (*WorksheetOrganizerMessage, error) {
	q := qb.Q().Space(`
		SELECT
			payload
		FROM worksheet_organizer
		WHERE worksheet = ? AND principal = ?
	`, worksheetResourceID, principal)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	worksheetOrganizer := WorksheetOrganizerMessage{
		WorksheetResourceID: worksheetResourceID,
		Principal:           principal,
		Payload:             &storepb.WorkSheetOrganizerPayload{},
	}
	var payload []byte
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&payload,
	); err != nil {
		if err == sql.ErrNoRows {
			return &worksheetOrganizer, nil
		}
		return nil, errors.Wrapf(err, "failed to scan")
	}
	workSheetPayload := &storepb.WorkSheetOrganizerPayload{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(payload, workSheetPayload); err != nil {
		return nil, err
	}
	worksheetOrganizer.Payload = workSheetPayload

	return &worksheetOrganizer, nil
}

// UpsertWorksheetOrganizer upserts a new SheetOrganizerMessage.
func (s *Store) UpsertWorksheetOrganizer(ctx context.Context, patch *WorksheetOrganizerMessage) (*WorksheetOrganizerMessage, error) {
	payloadStr, err := protojson.Marshal(patch.Payload)
	if err != nil {
		return nil, err
	}
	q := qb.Q().Space(`
	  INSERT INTO worksheet_organizer (
			worksheet,
			principal,
			payload
		)
		VALUES (?, ?, ?)
		ON CONFLICT(worksheet, principal) DO UPDATE SET
			payload = EXCLUDED.payload
		RETURNING
			worksheet,
			principal,
			payload
	`, patch.WorksheetResourceID, patch.Principal, payloadStr)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var worksheetOrganizer WorksheetOrganizerMessage
	var payload []byte
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&worksheetOrganizer.WorksheetResourceID,
		&worksheetOrganizer.Principal,
		&payload,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}
	workSheetPayload := &storepb.WorkSheetOrganizerPayload{}
	if err := common.ProtojsonUnmarshaler.Unmarshal(payload, workSheetPayload); err != nil {
		return nil, err
	}
	worksheetOrganizer.Payload = workSheetPayload

	return &worksheetOrganizer, nil
}

// BatchUpdateWorksheetOrganizer updates worksheet organizer payload fields in batch.
func (s *Store) BatchUpdateWorksheetOrganizer(ctx context.Context, patch *BatchUpdateWorksheetOrganizerPatch) ([]*WorksheetOrganizerMessage, error) {
	if len(patch.WorksheetResourceIDs) == 0 {
		return nil, nil
	}
	if patch.Principal == "" {
		return nil, errors.New("principal is empty")
	}
	if patch.Starred == nil && patch.Folders == nil {
		return nil, errors.New("empty worksheet organizer patch")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	insertQuery, insertArgs, err := qb.Q().Space(`
		INSERT INTO worksheet_organizer (
			worksheet,
			principal,
			payload
		)
		SELECT worksheet.resource_id, ?, '{}'::jsonb
		FROM worksheet
		WHERE worksheet.resource_id = ANY(?)
		ON CONFLICT (worksheet, principal) DO NOTHING
	`, patch.Principal, patch.WorksheetResourceIDs).ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build worksheet organizer insert query")
	}
	if _, err := tx.ExecContext(ctx, insertQuery, insertArgs...); err != nil {
		return nil, errors.Wrap(err, "failed to insert missing worksheet organizers")
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
	updateArgs = append(updateArgs, patch.Principal, patch.WorksheetResourceIDs)

	updateQuery, args, err := qb.Q().Space(fmt.Sprintf(`
		UPDATE worksheet_organizer
		SET payload = %s
		WHERE principal = ? AND worksheet = ANY(?)
		RETURNING worksheet, principal, payload
	`, payloadExpr), updateArgs...).ToSQL()
	if err != nil {
		return nil, errors.Wrap(err, "failed to build worksheet organizer update query")
	}
	rows, err := tx.QueryContext(ctx, updateQuery, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update worksheet organizers")
	}
	defer rows.Close()

	var worksheetOrganizers []*WorksheetOrganizerMessage
	for rows.Next() {
		var worksheetOrganizer WorksheetOrganizerMessage
		var payload []byte
		if err := rows.Scan(
			&worksheetOrganizer.WorksheetResourceID,
			&worksheetOrganizer.Principal,
			&payload,
		); err != nil {
			return nil, err
		}
		workSheetPayload := &storepb.WorkSheetOrganizerPayload{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, workSheetPayload); err != nil {
			return nil, err
		}
		worksheetOrganizer.Payload = workSheetPayload
		worksheetOrganizers = append(worksheetOrganizers, &worksheetOrganizer)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}
	return worksheetOrganizers, nil
}

func GetListSheetFilter(ctx context.Context, s *Store, caller string, filter string, allowTitleContains bool) (*qb.Query, error) {
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
	getWorksheetID := func(name string) (string, error) {
		_, worksheetID, err := common.GetProjectIDWorksheetID(name)
		if err != nil {
			return "", errors.Errorf("invalid worksheet name %q", name)
		}
		return worksheetID, nil
	}

	parseToSQL := func(variable, value any) (*qb.Query, error) {
		switch variable {
		case "name":
			name, ok := value.(string)
			if !ok {
				return nil, errors.Errorf("invalid name value %q", value)
			}
			worksheetID, err := getWorksheetID(name)
			if err != nil {
				return nil, err
			}
			return qb.Q().Space("worksheet.resource_id = ?", worksheetID), nil
		case "creator":
			userID, err := getUserID(value.(string))
			if err != nil {
				return nil, err
			}
			return qb.Q().Space("worksheet.creator = ?", userID), nil
		case "starred":
			if starred, ok := value.(bool); ok {
				return qb.Q().Space("worksheet.resource_id IN (SELECT worksheet FROM worksheet_organizer WHERE principal = ? AND (payload->>'starred')::boolean = ?)", caller, starred), nil
			}
			return qb.Q().Space("TRUE"), nil
		case "visibility":
			visibility := WorkSheetVisibility(value.(string))
			return qb.Q().Space("worksheet.visibility = ?", visibility), nil
		case "folder":
			folder, ok := value.(string)
			if !ok {
				return nil, errors.Errorf("invalid folder value %q", value)
			}
			folder = strings.Trim(folder, "/")
			if folder == "" {
				return qb.Q().Space("COALESCE(jsonb_array_length(worksheet_organizer.payload->'folders'), 0) = 0"), nil
			}
			q := qb.Q()
			segments := strings.Split(folder, "/")
			for i, segment := range segments {
				if segment == "" {
					return nil, errors.Errorf("invalid folder %q", value)
				}
				q.And(fmt.Sprintf("worksheet_organizer.payload->'folders'->>%d = ?", i), segment)
			}
			q.And("jsonb_array_length(worksheet_organizer.payload->'folders') = ?", len(segments))
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
					return qb.Q().Space("LOWER(worksheet.name) LIKE ? ESCAPE '\\'", "%"+escapeLikePattern(strings.ToLower(strValue))+"%"), nil
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
				return qb.Q().Space("worksheet.creator != ?", userID), nil
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
					worksheetIDs := []string{}
					for _, raw := range rawList {
						name, ok := raw.(string)
						if !ok {
							return nil, errors.Errorf("invalid name value %q", raw)
						}
						worksheetID, err := getWorksheetID(name)
						if err != nil {
							return nil, err
						}
						worksheetIDs = append(worksheetIDs, worksheetID)
					}
					return qb.Q().Space("worksheet.resource_id = ANY(?)", worksheetIDs), nil
				case "visibility":
					visibilityList := []string{}
					for _, raw := range rawList {
						visibility, ok := raw.(string)
						if !ok {
							return nil, errors.Errorf("invalid visibility value %q", raw)
						}
						visibilityList = append(visibilityList, visibility)
					}
					return qb.Q().Space("worksheet.visibility = ANY(?)", visibilityList), nil
				default:
					return nil, errors.Errorf(`only "name" and "visibility" support "in" filter`)
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

func GetListWorksheetFilter(filter string) (*qb.Query, error) {
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
				return qb.Q().Space("worksheet.creator = ?", creatorEmail), nil
			}
			return qb.Q().Space("worksheet.creator != ?", creatorEmail), nil
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
