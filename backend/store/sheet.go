package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

// SheetVisibility is the visibility of a sheet.
type SheetVisibility string

const (
	// PrivateSheet is the sheet visibility for PRIVATE. Only sheet OWNER can read/write.
	PrivateSheet SheetVisibility = "PRIVATE"
	// ProjectSheet is the sheet visibility for PROJECT. Both sheet OWNER and project OWNER can read/write, and project DEVELOPER can read.
	ProjectSheet SheetVisibility = "PROJECT"
	// PublicSheet is the sheet visibility for PUBLIC. Sheet OWNER can read/write, and all others can read.
	PublicSheet SheetVisibility = "PUBLIC"
)

// SheetSource is the type of sheet origin source.
type SheetSource string

const (
	// SheetFromBytebase is the sheet created by Bytebase. e.g. SQL Editor.
	SheetFromBytebase SheetSource = "BYTEBASE"
	// SheetFromBytebaseArtifact is the artifact sheet.
	SheetFromBytebaseArtifact SheetSource = "BYTEBASE_ARTIFACT"
	// SheetFromGitLab is the sheet synced from GitLab (for both GitLab.com and self-hosted GitLab).
	SheetFromGitLab SheetSource = "GITLAB"
	// SheetFromGitHub is the sheet synced from GitHub (for both GitHub.com and GitHub Enterprise).
	SheetFromGitHub SheetSource = "GITHUB"
	// SheetFromBitbucket is the sheet synced from Bitbucket.
	SheetFromBitbucket SheetSource = "BITBUCKET"
)

// SheetType is the type of sheet.
type SheetType string

const (
	// SheetForSQL is the sheet that used for saving SQL statements.
	SheetForSQL SheetType = "SQL"
)

// SheetMessage is the message for a sheet.
type SheetMessage struct {
	ProjectUID int
	// The DatabaseUID is optional.
	// If not NULL, the sheet ProjectID should always be equal to the id of the database related project.
	// A project must remove all linked sheets for a particular database before that database can be transferred to a different project.
	DatabaseUID *int

	CreatorID int
	UpdaterID int

	Name       string
	Statement  string
	Visibility SheetVisibility
	Source     SheetSource
	Type       SheetType
	Payload    string

	// Output only fields
	UID         int
	Size        int64
	CreatedTime time.Time
	UpdatedTime time.Time
	Starred     bool
	Pinned      bool

	// Internal fields
	rowStatus RowStatus
	createdTs int64
	updatedTs int64
}

// FindSheetMessage is the API message for finding sheets.
type FindSheetMessage struct {
	UID *int

	// Standard fields
	RowStatus *RowStatus

	// Used to find the creator's sheet list.
	// When finding shared PROJECT/PUBLIC sheets, this value should be empty.
	// It does not make sense to set both `CreatorID` and `ExcludedCreatorID`.
	CreatorID *int
	// Used to find the sheets that are not created by the creator.
	ExcludedCreatorID *int

	// LoadFull is used if we want to load the full sheet.
	LoadFull bool

	// Related fields
	ProjectUID  *int
	DatabaseUID *int

	// Domain fields
	Name         *string
	Visibilities []SheetVisibility
	Source       *SheetSource
	Type         *SheetType
	PayloadType  *string
	// Used to find (un)starred/pinned sheet list, could be PRIVATE/PROJECT/PUBLIC sheet.
	// For now, we only need the starred sheets.
	OrganizerPrincipalIDStarred    *int
	OrganizerPrincipalIDNotStarred *int
	// Used to find a sheet list from projects containing PrincipalID as an active member.
	// When finding a shared PROJECT/PUBLIC sheets, this value should be present.
	PrincipalID *int
}

// PatchSheetMessage is the message to patch a sheet.
type PatchSheetMessage struct {
	UID         int
	UpdaterID   int
	Name        *string
	Statement   *string
	Visibility  *string
	ProjectUID  *int
	DatabaseUID *int
	// TODO(zp): update the payload.
	Payload *string
}

// GetSheetStatementByID gets the statement of a sheet by ID.
func (s *Store) GetSheetStatementByID(ctx context.Context, id int) (string, error) {
	if statement, ok := s.sheetStatementCache.Get(id); ok {
		return statement, nil
	}

	sheets, err := s.ListSheets(ctx, &FindSheetMessage{UID: &id, LoadFull: true}, SystemBotID)
	if err != nil {
		return "", err
	}
	if len(sheets) == 0 {
		return "", errors.Errorf("sheet not found with id %d", id)
	}
	if len(sheets) > 1 {
		return "", errors.Errorf("expected 1 sheet, got %d", len(sheets))
	}
	statement := sheets[0].Statement
	s.sheetStatementCache.Set(id, statement, 10*time.Minute)
	return statement, nil
}

// GetSheet gets a sheet.
func (s *Store) GetSheet(ctx context.Context, find *FindSheetMessage, currentPrincipalID int) (*SheetMessage, error) {
	sheets, err := s.ListSheets(ctx, find, currentPrincipalID)
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

// ListSheets returns a list of sheets.
func (s *Store) ListSheets(ctx context.Context, find *FindSheetMessage, currentPrincipalID int) ([]*SheetMessage, error) {
	where, args := []string{"TRUE"}, []any{}

	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("sheet.id = $%d", len(args)+1)), append(args, *v)
	}

	// Standard fields
	if v := find.RowStatus; v != nil {
		where, args = append(where, fmt.Sprintf("sheet.row_status = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.CreatorID; v != nil {
		where, args = append(where, fmt.Sprintf("sheet.creator_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ExcludedCreatorID; v != nil {
		where, args = append(where, fmt.Sprintf("sheet.creator_id != $%d", len(args)+1)), append(args, *v)
	}

	// Related fields
	if v := find.ProjectUID; v != nil {
		where, args = append(where, fmt.Sprintf("sheet.project_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.DatabaseUID; v != nil {
		where, args = append(where, fmt.Sprintf("sheet.database_id = $%d", len(args)+1)), append(args, *v)
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

	if v := find.Source; v != nil {
		where, args = append(where, fmt.Sprintf("sheet.source = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Type; v != nil {
		where, args = append(where, fmt.Sprintf("sheet.type = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PayloadType; v != nil {
		where, args = append(where, fmt.Sprintf("sheet.payload->'type' = $%d", len(args)+1)), append(args, *v)
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
			sheet.row_status,
			sheet.creator_id,
			sheet.created_ts,
			sheet.updater_id,
			sheet.updated_ts,
			sheet.project_id,
			sheet.database_id,
			sheet.name,
			%s,
			sheet.visibility,
			sheet.source,
			sheet.type,
			sheet.payload,
			LENGTH(sheet.statement),
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

	var sheets []*SheetMessage
	for rows.Next() {
		var sheet SheetMessage
		if err := rows.Scan(
			&sheet.UID,
			&sheet.rowStatus,
			&sheet.CreatorID,
			&sheet.createdTs,
			&sheet.UpdaterID,
			&sheet.updatedTs,
			&sheet.ProjectUID,
			&sheet.DatabaseUID,
			&sheet.Name,
			&sheet.Statement,
			&sheet.Visibility,
			&sheet.Source,
			&sheet.Type,
			&sheet.Payload,
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

// CreateSheet creates a new sheet.
func (s *Store) CreateSheet(ctx context.Context, create *SheetMessage) (*SheetMessage, error) {
	if create.Payload == "" {
		create.Payload = "{}"
	}

	query := fmt.Sprintf(`
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
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, LEFT(statement, %d), visibility, source, type, LENGTH(statement), payload
	`, common.MaxSheetSize)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	databaseID := sql.NullInt32{}
	var sheet SheetMessage

	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.CreatorID,
		create.ProjectUID,
		create.DatabaseUID,
		create.Name,
		create.Statement,
		create.Visibility,
		create.Source,
		create.Type,
		create.Payload,
	).Scan(
		&sheet.UID,
		&sheet.rowStatus,
		&sheet.CreatorID,
		&sheet.createdTs,
		&sheet.UpdaterID,
		&sheet.updatedTs,
		&sheet.ProjectUID,
		&databaseID,
		&sheet.Name,
		&sheet.Statement,
		&sheet.Visibility,
		&sheet.Source,
		&sheet.Type,
		&sheet.Size,
		&sheet.Payload,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	if databaseID.Valid {
		value := int(databaseID.Int32)
		sheet.DatabaseUID = &value
	}
	sheet.CreatedTime = time.Unix(sheet.createdTs, 0)
	sheet.UpdatedTime = time.Unix(sheet.updatedTs, 0)

	return &sheet, nil
}

// PatchSheet updates a sheet.
func (s *Store) PatchSheet(ctx context.Context, patch *PatchSheetMessage) (*SheetMessage, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}

	sheet, err := patchSheetImpl(ctx, tx, patch)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	s.sheetStatementCache.Invalidate(patch.UID)

	return sheet, nil
}

// DeleteSheet deletes an existing sheet by ID.
func (s *Store) DeleteSheet(ctx context.Context, sheetUID int) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM sheet WHERE id = $1`, sheetUID); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.sheetStatementCache.Invalidate(sheetUID)

	return nil
}

// patchSheetImpl updates a sheet's name/statement/visibility/payload/database_id/project_id.
func patchSheetImpl(ctx context.Context, tx *Tx, patch *PatchSheetMessage) (*SheetMessage, error) {
	set, args := []string{"updater_id = $1"}, []any{patch.UpdaterID}
	if v := patch.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Statement; v != nil {
		set, args = append(set, fmt.Sprintf("statement = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Visibility; v != nil {
		set, args = append(set, fmt.Sprintf("visibility = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Payload; v != nil {
		set, args = append(set, fmt.Sprintf("payload = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.DatabaseUID; v != nil {
		set, args = append(set, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.ProjectUID; v != nil {
		set, args = append(set, fmt.Sprintf("project_id = $%d", len(args)+1)), append(args, *v)
	}

	args = append(args, patch.UID)

	var sheet SheetMessage
	databaseID := sql.NullInt32{}

	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE sheet
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, LEFT(statement, %d), visibility, source, type, payload, LENGTH(statement)
	`, len(args), common.MaxSheetSize),
		args...,
	).Scan(
		&sheet.UID,
		&sheet.rowStatus,
		&sheet.CreatorID,
		&sheet.createdTs,
		&sheet.UpdaterID,
		&sheet.updatedTs,
		&sheet.ProjectUID,
		&databaseID,
		&sheet.Name,
		&sheet.Statement,
		&sheet.Visibility,
		&sheet.Source,
		&sheet.Type,
		&sheet.Payload,
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
