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
	// ProjectWorkSheet is the sheet visibility for PROJECT. Both sheet OWNER and project OWNER can read/write, and project DEVELOPER can read.
	ProjectWorkSheet WorkSheetVisibility = "PROJECT"
	// PublicWorkSheet is the sheet visibility for PUBLIC. Sheet OWNER can read/write, and all others can read.
	PublicWorkSheet WorkSheetVisibility = "PUBLIC"
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
	Pinned      bool

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

	// Used to find (un)starred/pinned sheet list, could be PRIVATE/PROJECT/PUBLIC sheet.
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
	where, args = append(where, fmt.Sprintf("sheet.source = $%d", len(args)+1)), append(args, SheetFromBytebase)
	where, args = append(where, fmt.Sprintf("sheet.type = $%d", len(args)+1)), append(args, SheetForSQL)

	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("sheet.id = $%d", len(args)+1)), append(args, *v)
	}

	// Standard fields
	if v := find.CreatorID; v != nil {
		where, args = append(where, fmt.Sprintf("sheet.creator_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ExcludedCreatorID; v != nil {
		where, args = append(where, fmt.Sprintf("sheet.creator_id != $%d", len(args)+1)), append(args, *v)
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
		where, args = append(where, fmt.Sprintf("sheet.project_id IN (SELECT project_id FROM project_member WHERE principal_id = $%d)", len(args)+1)), append(args, *v)
	}
	if v := find.OrganizerPrincipalIDStarred; v != nil {
		where, args = append(where, fmt.Sprintf("sheet.id IN (SELECT sheet_id FROM sheet_organizer WHERE principal_id = $%d AND starred = true)", len(args)+1)), append(args, *v)
	}
	if v := find.OrganizerPrincipalIDNotStarred; v != nil {
		where, args = append(where, fmt.Sprintf("sheet.id IN (SELECT sheet_id FROM sheet_organizer WHERE principal_id = $%d AND starred = false)", len(args)+1)), append(args, *v)
	}
	statementField := fmt.Sprintf("LEFT(sheet.statement, %d)", common.MaxSheetSize)
	if find.LoadFull {
		statementField = "sheet.statement"
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			sheet.id,
			sheet.creator_id,
			sheet.created_ts,
			sheet.updater_id,
			sheet.updated_ts,
			sheet.project_id,
			sheet.database_id,
			sheet.name,
			%s,
			sheet.visibility,
			OCTET_LENGTH(sheet.statement),
			COALESCE(sheet_organizer.starred, FALSE),
			COALESCE(sheet_organizer.pinned, FALSE)
		FROM sheet
		LEFT JOIN sheet_organizer ON sheet_organizer.sheet_id = sheet.id AND sheet_organizer.principal_id = %d
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
			&sheet.Pinned,
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
		INSERT INTO sheet (
			creator_id,
			updater_id,
			project_id,
			database_id,
			name,
			statement,
			visibility,
			source,
			type,
			payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
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
		SheetFromBytebase,
		SheetForSQL,
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
func (s *Store) PatchWorkSheet(ctx context.Context, patch *PatchWorkSheetMessage) (*WorkSheetMessage, error) {
	// return s.PatchSheet(ctx, &PatchSheetMessage{})
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}

	sheet, err := patchWorkSheetImpl(ctx, tx, patch)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	s.sheetCache.Remove(patch.UID)
	return sheet, nil
}

// DeleteWorkSheet deletes an existing sheet by ID.
func (s *Store) DeleteWorkSheet(ctx context.Context, sheetUID int) error {
	return s.DeleteSheet(ctx, sheetUID)
}

// patchWorkSheetImpl updates a sheet's name/statement/visibility/database_id/project_id.
func patchWorkSheetImpl(ctx context.Context, tx *Tx, patch *PatchWorkSheetMessage) (*WorkSheetMessage, error) {
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

	var sheet WorkSheetMessage
	databaseID := sql.NullInt32{}

	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE sheet
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, LEFT(statement, %d), visibility, OCTET_LENGTH(statement)
	`, len(args), common.MaxSheetSize),
		args...,
	).Scan(
		&sheet.UID,
		&sheet.CreatorID,
		&sheet.createdTs,
		&sheet.UpdaterID,
		&sheet.updatedTs,
		&sheet.ProjectUID,
		&databaseID,
		&sheet.Title,
		&sheet.Statement,
		&sheet.Visibility,
		&sheet.Size,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("sheet ID not found: %d", patch.UID)}
		}
		return nil, err
	}

	if databaseID.Valid {
		value := int(databaseID.Int32)
		sheet.DatabaseUID = &value
	}
	sheet.CreatedTime = time.Unix(sheet.createdTs, 0)
	sheet.UpdatedTime = time.Unix(sheet.updatedTs, 0)

	return &sheet, nil
}
