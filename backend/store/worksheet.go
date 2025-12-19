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
	ProjectID string
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
	UID       int
	Size      int64
	CreatedAt time.Time
	UpdatedAt time.Time
	Starred   bool
	Folders   []string
}

// FindWorkSheetMessage is the API message for finding sheets.
type FindWorkSheetMessage struct {
	UID *int

	// LoadFull is used if we want to load the full sheet.
	LoadFull bool

	FilterQ *qb.Query
}

// PatchWorkSheetMessage is the message to patch a sheet.
type PatchWorkSheetMessage struct {
	UID          int
	Title        *string
	Statement    *string
	Visibility   *string
	InstanceID   *string
	DatabaseName *string
}

// GetWorkSheet gets a sheet.
func (s *Store) GetWorkSheet(ctx context.Context, find *FindWorkSheetMessage, currentPrincipal string) (*WorkSheetMessage, error) {
	sheets, err := s.ListWorkSheets(ctx, find, currentPrincipal)
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
func (s *Store) ListWorkSheets(ctx context.Context, find *FindWorkSheetMessage, currentPrincipal string) ([]*WorkSheetMessage, error) {
	statementField := fmt.Sprintf("LEFT(worksheet.statement, %d)", common.MaxSheetSize)
	if find.LoadFull {
		statementField = "worksheet.statement"
	}

	q := qb.Q().Space(fmt.Sprintf(`
		SELECT
			worksheet.id,
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
		LEFT JOIN worksheet_organizer ON worksheet_organizer.worksheet_id = worksheet.id AND worksheet_organizer.principal = '%s'
		WHERE TRUE`, statementField, currentPrincipal))

	if filterQ := find.FilterQ; filterQ != nil {
		q.And("?", filterQ)
	}

	if v := find.UID; v != nil {
		q.And("worksheet.id = ?", *v)
	}

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, args...)
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
			&sheet.UID,
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
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return sheets, nil
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
		RETURNING id, created_at, updated_at, OCTET_LENGTH(statement)
	`, create.Creator, create.ProjectID, create.InstanceID, create.DatabaseName, create.Title, create.Statement, create.Visibility)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := tx.QueryRowContext(ctx, query, args...).Scan(
		&create.UID,
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
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	return create, nil
}

// PatchWorkSheet updates a sheet.
func (s *Store) PatchWorkSheet(ctx context.Context, patch *PatchWorkSheetMessage) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}

	if err := patchWorkSheetImpl(ctx, tx, patch); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit transaction")
	}
	return nil
}

// DeleteWorkSheet deletes an existing sheet by ID.
func (s *Store) DeleteWorkSheet(ctx context.Context, sheetUID int) error {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	q1 := qb.Q().Space(`DELETE FROM worksheet_organizer WHERE worksheet_id = ?`, sheetUID)
	query1, args1, err := q1.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	if _, err := tx.ExecContext(ctx, query1, args1...); err != nil {
		return err
	}

	q2 := qb.Q().Space(`DELETE FROM worksheet WHERE id = ?`, sheetUID)
	query2, args2, err := q2.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	if _, err := tx.ExecContext(ctx, query2, args2...); err != nil {
		return err
	}

	return tx.Commit()
}

// patchWorkSheetImpl updates a sheet's name/statement/visibility/instance/db_name/project.
func patchWorkSheetImpl(ctx context.Context, txn *sql.Tx, patch *PatchWorkSheetMessage) error {
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

	query, args, err := qb.Q().Space("UPDATE worksheet SET ? WHERE id = ?", set, patch.UID).ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}
	if _, err := txn.ExecContext(ctx, query, args...); err != nil {
		return err
	}
	return nil
}

// WorksheetOrganizerMessage is the store message for worksheet organizer.
type WorksheetOrganizerMessage struct {
	UID int

	// Related fields
	WorksheetUID int
	Principal    string
	Payload      *storepb.WorkSheetOrganizerPayload
}

func (s *Store) GetWorksheetOrganizer(ctx context.Context, worksheetUID int, principal string) (*WorksheetOrganizerMessage, error) {
	q := qb.Q().Space(`
		SELECT
			id,
			payload
		FROM worksheet_organizer
		WHERE worksheet_id = ? AND principal = ?
	`, worksheetUID, principal)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	worksheetOrganizer := WorksheetOrganizerMessage{
		WorksheetUID: worksheetUID,
		Principal:    principal,
		Payload:      &storepb.WorkSheetOrganizerPayload{},
	}
	var payload []byte
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(
		&worksheetOrganizer.UID,
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
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	payloadStr, err := protojson.Marshal(patch.Payload)
	if err != nil {
		return nil, err
	}
	q := qb.Q().Space(`
	  INSERT INTO worksheet_organizer (
			worksheet_id,
			principal,
			payload
		)
		VALUES (?, ?, ?)
		ON CONFLICT(worksheet_id, principal) DO UPDATE SET
			payload = EXCLUDED.payload
		RETURNING
			id,
			worksheet_id,
			principal,
			payload
	`, patch.WorksheetUID, patch.Principal, payloadStr)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var worksheetOrganizer WorksheetOrganizerMessage
	var payload []byte
	if err := tx.QueryRowContext(ctx, query, args...).Scan(
		&worksheetOrganizer.UID,
		&worksheetOrganizer.WorksheetUID,
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

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &worksheetOrganizer, nil
}

func GetListSheetFilter(ctx context.Context, s *Store, caller string, filter string) (*qb.Query, error) {
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
		// Assuming we trust the email or validate existence elsewhere, or we can keep GetUserByEmail if strict validation needed.
		// For now, let's keep validation but return email.
		user, err := s.GetUserByEmail(ctx, creatorEmail)
		if err != nil {
			return "", errors.Errorf("failed to get user: %v", err)
		}
		if user == nil {
			return "", errors.Errorf("user with email %s not found", creatorEmail)
		}
		return user.Email, nil
	}

	parseToSQL := func(variable, value any) (*qb.Query, error) {
		switch variable {
		case "creator":
			userID, err := getUserID(value.(string))
			if err != nil {
				return nil, err
			}
			return qb.Q().Space("worksheet.creator = ?", userID), nil
		case "starred":
			if starred, ok := value.(bool); ok {
				return qb.Q().Space("worksheet.id IN (SELECT worksheet_id FROM worksheet_organizer WHERE principal = ? AND (payload->>'starred')::boolean = ?)", caller, starred), nil
			}
			return qb.Q().Space("TRUE"), nil
		case "visibility":
			visibility := WorkSheetVisibility(value.(string))
			return qb.Q().Space("worksheet.visibility = ?", visibility), nil
		case "project":
			projectID, err := common.GetProjectID(value.(string))
			if err != nil {
				return nil, errors.Errorf("invalid project filter %q", value)
			}
			return qb.Q().Space("worksheet.project = ?", projectID), nil
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
				if variable != "visibility" {
					return nil, errors.Errorf(`only "visibility" support "visibility in [xx]" filter`)
				}
				rawList, ok := value.([]any)
				if !ok {
					return nil, errors.Errorf("invalid visibility value %q", value)
				}
				if len(rawList) == 0 {
					return nil, errors.New("empty visibility filter")
				}
				visibilityList := []string{}
				for _, raw := range rawList {
					visibilityList = append(visibilityList, raw.(string))
				}
				return qb.Q().Space("worksheet.visibility = ANY(?)", visibilityList), nil
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
