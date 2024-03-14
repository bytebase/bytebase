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

// SheetMessage is the message for a sheet.
type SheetMessage struct {
	ProjectUID int
	// The DatabaseUID is optional.
	// If not NULL, the sheet ProjectID should always be equal to the id of the database related project.
	// A project must remove all linked sheets for a particular database before that database can be transferred to a different project.
	DatabaseUID *int

	CreatorID int
	UpdaterID int

	Title     string
	Statement string
	Payload   *storepb.SheetPayload

	// Output only fields
	UID         int
	Size        int64
	CreatedTime time.Time
	UpdatedTime time.Time

	// Internal fields
	createdTs int64
	updatedTs int64
}

// FindSheetMessage is the API message for finding sheets.
type FindSheetMessage struct {
	UID *int

	// Used to find the creator's sheet list.
	// When finding shared PROJECT/PUBLIC sheets, this value should be empty.
	// It does not make sense to set both `CreatorID` and `ExcludedCreatorID`.
	CreatorID *int

	// LoadFull is used if we want to load the full sheet.
	LoadFull bool

	// Related fields
	ProjectUID *int
}

// PatchSheetMessage is the message to patch a sheet.
type PatchSheetMessage struct {
	UID       int
	UpdaterID int
	Statement *string
}

// GetSheetStatementByID gets the statement of a sheet by ID.
func (s *Store) GetSheetStatementByID(ctx context.Context, id int) (string, error) {
	if v, ok := s.sheetCache.Get(id); ok {
		return v, nil
	}

	sheet, err := s.GetSheet(ctx, &FindSheetMessage{UID: &id, LoadFull: true})
	if err != nil {
		return "", err
	}
	if sheet == nil {
		return "", errors.Errorf("sheet not found with id %d", id)
	}

	statement := sheet.Statement
	s.sheetCache.Add(id, statement)
	return statement, nil
}

// GetSheet gets a sheet.
func (s *Store) GetSheet(ctx context.Context, find *FindSheetMessage) (*SheetMessage, error) {
	sheets, err := s.listSheets(ctx, find)
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

// listSheets returns a list of sheets.
func (s *Store) listSheets(ctx context.Context, find *FindSheetMessage) ([]*SheetMessage, error) {
	where, args := []string{"TRUE"}, []any{}

	// Standard fields
	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("sheet.id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.CreatorID; v != nil {
		where, args = append(where, fmt.Sprintf("sheet.creator_id = $%d", len(args)+1)), append(args, *v)
	}

	// Related fields
	if v := find.ProjectUID; v != nil {
		where, args = append(where, fmt.Sprintf("sheet.project_id = $%d", len(args)+1)), append(args, *v)
	}

	// Domain fields
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
			sheet.payload,
			OCTET_LENGTH(sheet.statement)
		FROM sheet
		WHERE %s`, statementField, strings.Join(where, " AND ")),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sheets []*SheetMessage
	for rows.Next() {
		var sheet SheetMessage
		var payload []byte
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
			&payload,
			&sheet.Size,
		); err != nil {
			return nil, err
		}
		sheetPayload := &storepb.SheetPayload{}
		if err := protojsonUnmarshaler.Unmarshal(payload, sheetPayload); err != nil {
			return nil, err
		}
		sheet.Payload = sheetPayload

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
	if create.Payload == nil {
		create.Payload = &storepb.SheetPayload{}
	}
	payload, err := protojson.Marshal(create.Payload)
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
			payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
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
	if v := patch.Statement; v != nil {
		s.sheetCache.Add(patch.UID, *v)
	}
	return sheet, nil
}

// patchSheetImpl updates a sheet's name/statement/payload/database_id/project_id.
func patchSheetImpl(ctx context.Context, tx *Tx, patch *PatchSheetMessage) (*SheetMessage, error) {
	set, args := []string{"updater_id = $1"}, []any{patch.UpdaterID}
	if v := patch.Statement; v != nil {
		set, args = append(set, fmt.Sprintf("statement = $%d", len(args)+1)), append(args, *v)
	}

	args = append(args, patch.UID)

	var sheet SheetMessage
	var payload []byte
	databaseID := sql.NullInt32{}

	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE sheet
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, LEFT(statement, %d), payload, OCTET_LENGTH(statement)
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
		&payload,
		&sheet.Size,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("sheet ID not found: %d", patch.UID)}
		}
		return nil, err
	}
	sheetPayload := &storepb.SheetPayload{}
	if err := protojsonUnmarshaler.Unmarshal(payload, sheetPayload); err != nil {
		return nil, err
	}
	sheet.Payload = sheetPayload

	if databaseID.Valid {
		value := int(databaseID.Int32)
		sheet.DatabaseUID = &value
	}
	sheet.CreatedTime = time.Unix(sheet.createdTs, 0)
	sheet.UpdatedTime = time.Unix(sheet.updatedTs, 0)

	return &sheet, nil
}
