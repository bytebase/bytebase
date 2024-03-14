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
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
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
	ProjectUID int
	// The DatabaseUID is optional.
	// If not NULL, the sheet ProjectID should always be equal to the id of the database related project.
	// A project must remove all linked sheets for a particular database before that database can be transferred to a different project.
	DatabaseUID *int

	CreatorID int
	UpdaterID int

	Title      string
	Statement  string
	Visibility WorkSheetVisibility

	// Output only fields
	UID         int
	Size        int64
	CreatedTime time.Time
	UpdatedTime time.Time
	Starred     bool

	// Internal fields
	createdTs int64
	updatedTs int64
}

// FindWorkSheetMessage is the API message for finding sheets.
type FindWorkSheetMessage struct {
	UID *int

	// Used to find the creator's sheet list.
	// When finding shared PROJECT/PUBLIC sheets, this value should be empty.
	// It does not make sense to set both `CreatorID` and `ExcludedCreatorID`.
	CreatorID *int
	// Used to find the sheets that are not created by the creator.
	ExcludedCreatorID *int

	// LoadFull is used if we want to load the full sheet.
	LoadFull bool

	// Domain fields
	Visibilities []WorkSheetVisibility

	// Used to find (un)starred sheet list, could be PRIVATE/PROJECT/PUBLIC sheet.
	// For now, we only need the starred sheets.
	OrganizerPrincipalIDStarred    *int
	OrganizerPrincipalIDNotStarred *int
	// Used to find a sheet list from projects containing PrincipalID as an active member.
	// When finding a shared PROJECT/PUBLIC sheets, this value should be present.
	PrincipalID *int
}

// PatchWorkSheetMessage is the message to patch a sheet.
type PatchWorkSheetMessage struct {
	UID        int
	UpdaterID  int
	Title      *string
	Statement  *string
	Visibility *string
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

	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("worksheet.id = $%d", len(args)+1)), append(args, *v)
	}

	// Standard fields
	if v := find.CreatorID; v != nil {
		where, args = append(where, fmt.Sprintf("worksheet.creator_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ExcludedCreatorID; v != nil {
		where, args = append(where, fmt.Sprintf("worksheet.creator_id != $%d", len(args)+1)), append(args, *v)
	}

	// Domain fields
	visibilitiesWhere := []string{}
	for _, v := range find.Visibilities {
		visibilitiesWhere, args = append(visibilitiesWhere, fmt.Sprintf("visibility = $%d", len(args)+1)), append(args, v)
	}
	if len(visibilitiesWhere) > 0 {
		where = append(where, fmt.Sprintf("(%s)", strings.Join(visibilitiesWhere, " OR ")))
	}
	if v := find.PrincipalID; v != nil {
		where, args = append(where, fmt.Sprintf("worksheet.project_id IN (SELECT project_id FROM project_member WHERE principal_id = $%d)", len(args)+1)), append(args, *v)
	}
	if v := find.OrganizerPrincipalIDStarred; v != nil {
		where, args = append(where, fmt.Sprintf("worksheet.id IN (SELECT worksheet_id FROM worksheet_organizer WHERE principal_id = $%d AND starred = true)", len(args)+1)), append(args, *v)
	}
	if v := find.OrganizerPrincipalIDNotStarred; v != nil {
		where, args = append(where, fmt.Sprintf("worksheet.id IN (SELECT worksheet_id FROM worksheet_organizer WHERE principal_id = $%d AND starred = false)", len(args)+1)), append(args, *v)
	}
	statementField := fmt.Sprintf("LEFT(worksheet.statement, %d)", common.MaxSheetSize)
	if find.LoadFull {
		statementField = "worksheet.statement"
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			worksheet.id,
			worksheet.creator_id,
			worksheet.created_ts,
			worksheet.updater_id,
			worksheet.updated_ts,
			worksheet.project_id,
			worksheet.database_id,
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
			&sheet.createdTs,
			&sheet.UpdaterID,
			&sheet.updatedTs,
			&sheet.ProjectUID,
			&sheet.DatabaseUID,
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

	for _, sheet := range sheets {
		sheet.CreatedTime = time.Unix(sheet.createdTs, 0)
		sheet.UpdatedTime = time.Unix(sheet.updatedTs, 0)
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
			updater_id,
			project_id,
			database_id,
			name,
			statement,
			visibility,
			payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_ts, updated_ts, OCTET_LENGTH(statement)
	`

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	create.UpdaterID = create.CreatorID
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.CreatorID,
		create.ProjectUID,
		create.DatabaseUID,
		create.Title,
		create.Statement,
		create.Visibility,
		payload,
	).Scan(
		&create.UID,
		&create.createdTs,
		&create.updatedTs,
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

	create.CreatedTime = time.Unix(create.createdTs, 0)
	create.UpdatedTime = time.Unix(create.updatedTs, 0)

	return create, nil
}

// PatchWorkSheet updates a sheet.
func (s *Store) PatchWorkSheet(ctx context.Context, patch *PatchWorkSheetMessage) error {
	tx, err := s.db.BeginTx(ctx, nil)
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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM worksheet WHERE id = $1`, sheetUID); err != nil {
		return err
	}

	return tx.Commit()
}

// patchWorkSheetImpl updates a sheet's name/statement/visibility/database_id/project_id.
func patchWorkSheetImpl(ctx context.Context, tx *Tx, patch *PatchWorkSheetMessage) error {
	set, args := []string{"updater_id = $1"}, []any{patch.UpdaterID}
	if v := patch.Title; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Statement; v != nil {
		set, args = append(set, fmt.Sprintf("statement = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Visibility; v != nil {
		set, args = append(set, fmt.Sprintf("visibility = $%d", len(args)+1)), append(args, *v)
	}
	args = append(args, patch.UID)

	query := fmt.Sprintf(`
		UPDATE worksheet
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d`, len(args))
	if _, err := tx.ExecContext(ctx, query, args...,
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
	tx, err := s.db.BeginTx(ctx, nil)
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
