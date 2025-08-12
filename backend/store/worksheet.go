package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
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

	Filter *ListResourceFilter
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
	where, args := []string{"TRUE"}, []any{}
	if filter := find.Filter; filter != nil {
		where = append(where, filter.Where)
		args = append(args, filter.Args...)
	}

	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("worksheet.id = $%d", len(args)+1)), append(args, *v)
	}

	statementField := fmt.Sprintf("LEFT(worksheet.statement, %d)", common.MaxSheetSize)
	if find.LoadFull {
		statementField = "worksheet.statement"
	}

	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
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
		WHERE %s`, statementField, currentPrincipalID, strings.Join(where, " AND ")),
		args...,
	)
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

	query := `
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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at, OCTET_LENGTH(statement)
	`

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.ProjectID,
		create.InstanceID,
		create.DatabaseName,
		create.Title,
		create.Statement,
		create.Visibility,
		payload,
	).Scan(
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

	if _, err := tx.ExecContext(ctx, `DELETE FROM worksheet_organizer WHERE worksheet_id = $1`, sheetUID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM worksheet WHERE id = $1`, sheetUID); err != nil {
		return err
	}

	return tx.Commit()
}

// patchWorkSheetImpl updates a sheet's name/statement/visibility/instance/db_name/project.
func patchWorkSheetImpl(ctx context.Context, txn *sql.Tx, patch *PatchWorkSheetMessage) error {
	set, args := []string{"updated_at = $1"}, []any{time.Now()}
	if v := patch.Title; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Statement; v != nil {
		set, args = append(set, fmt.Sprintf("statement = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Visibility; v != nil {
		set, args = append(set, fmt.Sprintf("visibility = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.InstanceID; v != nil {
		set, args = append(set, fmt.Sprintf("instance = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.DatabaseName; v != nil {
		set, args = append(set, fmt.Sprintf("db_name = $%d", len(args)+1)), append(args, *v)
	}
	args = append(args, patch.UID)

	query := fmt.Sprintf(`
		UPDATE worksheet
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d`, len(args))
	if _, err := txn.ExecContext(ctx, query, args...,
	); err != nil {
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

	query := `
	  INSERT INTO worksheet_organizer (
			worksheet_id,
			principal_id,
			starred
		)
		VALUES ($1, $2, $3)
		ON CONFLICT(worksheet_id, principal_id) DO UPDATE SET
			starred = EXCLUDED.starred
		RETURNING id, worksheet_id, principal_id, starred
	`
	var worksheetOrganizer WorksheetOrganizerMessage
	if err := tx.QueryRowContext(ctx, query,
		organizer.WorksheetUID,
		organizer.PrincipalUID,
		organizer.Starred,
	).Scan(
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
