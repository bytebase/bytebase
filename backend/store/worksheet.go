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

	CreatorID int

	Title      string
	Statement  string
	Visibility WorkSheetVisibility

	// Output only fields
	UID       int
	Size      int64
	CreatedAt time.Time
	UpdatedAt time.Time
	Starred   bool
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
func (s *Store) GetWorkSheet(ctx context.Context, find *FindWorkSheetMessage, currentPrincipalID int) (*WorkSheetMessage, error) {
	sheets, err := s.ListWorkSheets(ctx, find, currentPrincipalID)
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
func (s *Store) ListWorkSheets(ctx context.Context, find *FindWorkSheetMessage, currentPrincipalID int) ([]*WorkSheetMessage, error) {
	statementField := fmt.Sprintf("LEFT(worksheet.statement, %d)", common.MaxSheetSize)
	if find.LoadFull {
		statementField = "worksheet.statement"
	}

	q := qb.Q().Space(fmt.Sprintf(`
		SELECT
			worksheet.id,
			worksheet.creator_id,
			worksheet.created_at,
			worksheet.updated_at,
			worksheet.project,
			worksheet.instance,
			worksheet.db_name,
			worksheet.name,
			%s,
			worksheet.visibility,
			OCTET_LENGTH(worksheet.statement),
			COALESCE(worksheet_organizer.starred, FALSE)
		FROM worksheet
		LEFT JOIN worksheet_organizer ON worksheet_organizer.worksheet_id = worksheet.id AND worksheet_organizer.principal_id = %d
		WHERE TRUE`, statementField, currentPrincipalID))

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
		if err := rows.Scan(
			&sheet.UID,
			&sheet.CreatorID,
			&sheet.CreatedAt,
			&sheet.UpdatedAt,
			&sheet.ProjectID,
			&sheet.InstanceID,
			&sheet.DatabaseName,
			&sheet.Title,
			&sheet.Statement,
			&sheet.Visibility,
			&sheet.Size,
			&sheet.Starred,
		); err != nil {
			return nil, err
		}

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
	payload, err := protojson.Marshal(&storepb.SheetPayload{})
	if err != nil {
		return nil, err
	}

	q := qb.Q().Space(`
		INSERT INTO worksheet (
			creator_id,
			project,
			instance,
			db_name,
			name,
			statement,
			visibility,
			payload
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, created_at, updated_at, OCTET_LENGTH(statement)
	`, create.CreatorID, create.ProjectID, create.InstanceID, create.DatabaseName, create.Title, create.Statement, create.Visibility, payload)

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
		set.Comma("instance = ?", *v)
	}
	if v := patch.DatabaseName; v != nil {
		set.Comma("db_name = ?", *v)
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
	PrincipalUID int
	Starred      bool
}

// UpsertWorksheetOrganizer upserts a new SheetOrganizerMessage.
func (s *Store) UpsertWorksheetOrganizer(ctx context.Context, organizer *WorksheetOrganizerMessage) (*WorksheetOrganizerMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	q := qb.Q().Space(`
	  INSERT INTO worksheet_organizer (
			worksheet_id,
			principal_id,
			starred
		)
		VALUES (?, ?, ?)
		ON CONFLICT(worksheet_id, principal_id) DO UPDATE SET
			starred = EXCLUDED.starred
		RETURNING id, worksheet_id, principal_id, starred
	`, organizer.WorksheetUID, organizer.PrincipalUID, organizer.Starred)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	var worksheetOrganizer WorksheetOrganizerMessage
	if err := tx.QueryRowContext(ctx, query, args...).Scan(
		&worksheetOrganizer.UID,
		&worksheetOrganizer.WorksheetUID,
		&worksheetOrganizer.PrincipalUID,
		&worksheetOrganizer.Starred,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &worksheetOrganizer, nil
}

func GetListSheetFilter(ctx context.Context, s *Store, callerID int, filter string) (*qb.Query, error) {
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

	getUserID := func(name string) (int, error) {
		creatorEmail := strings.TrimPrefix(name, "users/")
		if creatorEmail == "" {
			return 0, errors.New("invalid empty creator identifier")
		}
		user, err := s.GetUserByEmail(ctx, creatorEmail)
		if err != nil {
			return 0, errors.Errorf("failed to get user: %v", err)
		}
		if user == nil {
			return 0, errors.Errorf("user with email %s not found", creatorEmail)
		}
		return user.ID, nil
	}

	parseToSQL := func(variable, value any) (*qb.Query, error) {
		switch variable {
		case "creator":
			userID, err := getUserID(value.(string))
			if err != nil {
				return nil, err
			}
			return qb.Q().Space("worksheet.creator_id = ?", userID), nil
		case "starred":
			if starred, ok := value.(bool); ok {
				return qb.Q().Space("worksheet.id IN (SELECT worksheet_id FROM worksheet_organizer WHERE principal_id = ? AND starred = ?)", callerID, starred), nil
			}
			return qb.Q().Space("TRUE"), nil
		case "visibility":
			visibility := WorkSheetVisibility(value.(string))
			return qb.Q().Space("worksheet.visibility = ?", visibility), nil
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
				return qb.Q().Space("worksheet.creator_id != ?", userID), nil
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
