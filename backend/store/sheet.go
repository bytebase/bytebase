package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	"github.com/bytebase/bytebase/backend/metric"
)

// sheetRaw is the store model for an Sheet.
// Fields have exactly the same meanings as Sheet.
type sheetRaw struct {
	ID int

	// Standard fields
	RowStatus api.RowStatus
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	ProjectID int
	// The DatabaseID is optional.
	// If not NULL, the sheet ProjectID should always be equal to the id of the database related project.
	// A project must remove all linked sheets for a particular database before that database can be transferred to a different project.
	DatabaseID *int

	// Domain specific fields
	Name       string
	Statement  string
	Visibility api.SheetVisibility
	Source     api.SheetSource
	Type       api.SheetType
	Payload    string
	Size       int64
}

// toSheet creates an instance of Sheet based on the sheetRaw.
// This is intended to be called when we need to compose an Sheet relationship.
func (raw *sheetRaw) toSheet() *api.Sheet {
	return &api.Sheet{
		ID: raw.ID,

		// Standard fields
		RowStatus: raw.RowStatus,
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		ProjectID: raw.ProjectID,
		// The DatabaseID is optional.
		// If not NULL, the sheet ProjectID should always be equal to the id of the database related project.
		// A project must remove all linked sheets for a particular database before that database can be transferred to a different project.
		DatabaseID: raw.DatabaseID,

		// Domain specific fields
		Name:       raw.Name,
		Statement:  raw.Statement,
		Visibility: raw.Visibility,
		Source:     raw.Source,
		Type:       raw.Type,
		Payload:    raw.Payload,
		Size:       raw.Size,
	}
}

// CreateSheet creates an instance of Sheet.
func (s *Store) CreateSheet(ctx context.Context, create *api.SheetCreate) (*api.Sheet, error) {
	sheetRaw, err := s.createSheetRaw(ctx, create)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create Sheet with SheetCreate[%+v]", create)
	}
	sheet, err := s.composeSheet(ctx, sheetRaw, create.CreatorID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Sheet with sheetRaw[%+v]", sheetRaw)
	}
	return sheet, nil
}

// GetSheet returns a sheet.
func (s *Store) GetSheet(ctx context.Context, find *api.SheetFind, currentPrincipalID int) (*api.Sheet, error) {
	sheet, err := s.GetSheetV2(ctx, find, currentPrincipalID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get Sheet with SheetFind[%+v]", find)
	}
	if sheet == nil {
		return nil, nil
	}
	composedSheet, err := s.composeSheetMessage(ctx, sheet)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Sheet with sheetMessage[%+v]", sheet)
	}
	return composedSheet, nil
}

// FindSheet finds a list of Sheet instances.
func (s *Store) FindSheet(ctx context.Context, find *api.SheetFind, currentPrincipalID int) ([]*api.Sheet, error) {
	sheetRawList, err := s.findSheetRaw(ctx, find)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find Sheet list")
	}
	var sheetList []*api.Sheet
	for _, raw := range sheetRawList {
		sheet, err := s.composeSheet(ctx, raw, currentPrincipalID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compose Sheet with sheetRaw[%+v]", raw)
		}
		sheetList = append(sheetList, sheet)
	}
	return sheetList, nil
}

// PatchSheet patches an instance of Sheet.
func (s *Store) PatchSheet(ctx context.Context, patch *api.SheetPatch) (*api.Sheet, error) {
	sheetRaw, err := s.patchSheetRaw(ctx, patch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to patch Sheet with SheetPatch[%+v]", patch)
	}
	s.sheetStatementCache.Invalidate(patch.ID)
	sheet, err := s.composeSheet(ctx, sheetRaw, patch.UpdaterID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to compose Sheet with sheetRaw[%+v]", sheetRaw)
	}
	return sheet, nil
}

// DeleteSheet deletes an existing sheet by ID.
// Returns ENOTFOUND if sheet does not exist.
func (s *Store) DeleteSheet(ctx context.Context, delete *api.SheetDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := deleteSheet(ctx, tx, delete); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	s.sheetStatementCache.Invalidate(delete.ID)

	return nil
}

// CountSheetGroupByRowstatusVisibilitySourceAndType counts the number of sheets group by row_status, visibility, source and type.
func (s *Store) CountSheetGroupByRowstatusVisibilitySourceAndType(ctx context.Context) ([]*metric.SheetCountMetric, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, `
		SELECT row_status, visibility, source, type, COUNT(*) AS count
		FROM sheet
		GROUP BY row_status, visibility, source, type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []*metric.SheetCountMetric
	for rows.Next() {
		var sheetCount metric.SheetCountMetric
		if err := rows.Scan(
			&sheetCount.RowStatus,
			&sheetCount.Visibility,
			&sheetCount.Source,
			&sheetCount.Type,
			&sheetCount.Count,
		); err != nil {
			return nil, err
		}
		res = append(res, &sheetCount)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return res, nil
}

//
// private functions
//

// composeSheet composes sheet relationships.
func (s *Store) composeSheet(ctx context.Context, raw *sheetRaw, currentPrincipalID int) (*api.Sheet, error) {
	sheet := raw.toSheet()

	creator, err := s.GetPrincipalByID(ctx, sheet.CreatorID)
	if err != nil {
		return nil, err
	}
	sheet.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, sheet.UpdaterID)
	if err != nil {
		return nil, err
	}
	sheet.Updater = updater

	project, err := s.GetProjectByID(ctx, sheet.ProjectID)
	if err != nil {
		return nil, err
	}
	sheet.Project = project

	if sheet.DatabaseID != nil {
		database, err := s.GetDatabase(ctx, &api.DatabaseFind{ID: sheet.DatabaseID})
		if err != nil {
			return nil, err
		}
		sheet.Database = database
	}

	sheetOrganizer, err := s.FindSheetOrganizer(ctx, &api.SheetOrganizerFind{
		SheetID:     sheet.ID,
		PrincipalID: currentPrincipalID,
	})
	if err != nil {
		return nil, err
	}
	if sheetOrganizer != nil {
		sheet.Starred = sheetOrganizer.Starred
		sheet.Pinned = sheetOrganizer.Pinned
	}

	return sheet, nil
}

// createSheetRaw creates a new sheet.
func (s *Store) createSheetRaw(ctx context.Context, create *api.SheetCreate) (*sheetRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	sheet, err := createSheetImpl(ctx, tx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return sheet, nil
}

// patchSheetRaw updates an existing sheet by ID.
func (s *Store) patchSheetRaw(ctx context.Context, patch *api.SheetPatch) (*sheetRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	sheet, err := patchSheetImpl(ctx, tx, patch)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return sheet, nil
}

// findSheetRaw retrieves a list of sheet based on find.
func (s *Store) findSheetRaw(ctx context.Context, find *api.SheetFind) ([]*sheetRaw, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	list, err := findSheetImpl(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// createSheetImpl creates a new sheet.
func createSheetImpl(ctx context.Context, tx *Tx, create *api.SheetCreate) (*sheetRaw, error) {
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
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, LEFT(statement, %d), visibility, source, type, payload, LENGTH(statement)
	`, common.MaxSheetSize)
	var sheetRaw sheetRaw
	databaseID := sql.NullInt32{}
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.CreatorID,
		create.ProjectID,
		create.DatabaseID,
		create.Name,
		create.Statement,
		create.Visibility,
		create.Source,
		create.Type,
		create.Payload,
	).Scan(
		&sheetRaw.ID,
		&sheetRaw.RowStatus,
		&sheetRaw.CreatorID,
		&sheetRaw.CreatedTs,
		&sheetRaw.UpdaterID,
		&sheetRaw.UpdatedTs,
		&sheetRaw.ProjectID,
		&databaseID,
		&sheetRaw.Name,
		&sheetRaw.Statement,
		&sheetRaw.Visibility,
		&sheetRaw.Source,
		&sheetRaw.Type,
		&sheetRaw.Payload,
		&sheetRaw.Size,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}
	if databaseID.Valid {
		value := int(databaseID.Int32)
		sheetRaw.DatabaseID = &value
	}
	return &sheetRaw, nil
}

// patchSheetImpl updates a sheet's name/statement/visibility.
func patchSheetImpl(ctx context.Context, tx *Tx, patch *api.SheetPatch) (*sheetRaw, error) {
	set, args := []string{"updater_id = $1"}, []any{patch.UpdaterID}
	if v := patch.RowStatus; v != nil {
		set, args = append(set, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, api.RowStatus(*v))
	}
	if v := patch.ProjectID; v != nil {
		set, args = append(set, fmt.Sprintf("project_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.DatabaseID; v != nil {
		set, args = append(set, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, *v)
	}
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

	args = append(args, patch.ID)

	var sheetRaw sheetRaw
	databaseID := sql.NullInt32{}
	if err := tx.QueryRowContext(ctx, fmt.Sprintf(`
		UPDATE sheet
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, LEFT(statement, %d), visibility, source, type, payload, LENGTH(statement)
	`, len(args), common.MaxSheetSize),
		args...,
	).Scan(
		&sheetRaw.ID,
		&sheetRaw.RowStatus,
		&sheetRaw.CreatorID,
		&sheetRaw.CreatedTs,
		&sheetRaw.UpdaterID,
		&sheetRaw.UpdatedTs,
		&sheetRaw.ProjectID,
		&databaseID,
		&sheetRaw.Name,
		&sheetRaw.Statement,
		&sheetRaw.Visibility,
		&sheetRaw.Source,
		&sheetRaw.Type,
		&sheetRaw.Payload,
		&sheetRaw.Size,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("sheet ID not found: %d", patch.ID)}
		}
		return nil, err
	}
	if databaseID.Valid {
		value := int(databaseID.Int32)
		sheetRaw.DatabaseID = &value
	}
	return &sheetRaw, nil
}

func findSheetImpl(ctx context.Context, tx *Tx, find *api.SheetFind) ([]*sheetRaw, error) {
	where, args := []string{"TRUE"}, []any{}

	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}

	// Standard fields
	if v := find.RowStatus; v != nil {
		where, args = append(where, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.CreatorID; v != nil {
		where, args = append(where, fmt.Sprintf("creator_id = $%d", len(args)+1)), append(args, *v)
	}

	// Related fields
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("project_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.DatabaseID; v != nil {
		where, args = append(where, fmt.Sprintf("database_id = $%d", len(args)+1)), append(args, *v)
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
		where, args = append(where, fmt.Sprintf("project_id IN (SELECT project_id FROM project_member WHERE principal_id = $%d)", len(args)+1)), append(args, *v)
	}
	if v := find.OrganizerPrincipalIDStarred; v != nil {
		// For now, we only need the starred sheets.
		where, args = append(where, fmt.Sprintf("id IN (SELECT sheet_id FROM sheet_organizer WHERE principal_id = $%d AND starred = true)", len(args)+1)), append(args, *v)
	}
	if v := find.Source; v != nil {
		where, args = append(where, fmt.Sprintf("source = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Type; v != nil {
		where, args = append(where, fmt.Sprintf("type = $%d", len(args)+1)), append(args, *v)
	}
	statementField := fmt.Sprintf("LEFT(statement, %d)", common.MaxSheetSize)
	if find.LoadFull {
		statementField = "statement"
	}

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			id,
			row_status,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			project_id,
			database_id,
			name,
			%s,
			visibility,
			source,
			type,
			payload,
			LENGTH(statement)
		FROM sheet
		WHERE %s`, statementField, strings.Join(where, " AND ")),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sheetRawList []*sheetRaw
	for rows.Next() {
		var sheetRaw sheetRaw
		databaseID := sql.NullInt32{}
		if err := rows.Scan(
			&sheetRaw.ID,
			&sheetRaw.RowStatus,
			&sheetRaw.CreatorID,
			&sheetRaw.CreatedTs,
			&sheetRaw.UpdaterID,
			&sheetRaw.UpdatedTs,
			&sheetRaw.ProjectID,
			&databaseID,
			&sheetRaw.Name,
			&sheetRaw.Statement,
			&sheetRaw.Visibility,
			&sheetRaw.Source,
			&sheetRaw.Type,
			&sheetRaw.Payload,
			&sheetRaw.Size,
		); err != nil {
			return nil, err
		}

		if databaseID.Valid {
			value := int(databaseID.Int32)
			sheetRaw.DatabaseID = &value
		}

		sheetRawList = append(sheetRawList, &sheetRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sheetRawList, nil
}

// deleteSheet permanently deletes a sheet by ID.
func deleteSheet(ctx context.Context, tx *Tx, delete *api.SheetDelete) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM sheet WHERE id = $1`, delete.ID); err != nil {
		return err
	}
	return nil
}

// SheetMessage is the message for a sheet.
type SheetMessage struct {
	ProjectUID int
	// The DatabaseID is optional.
	// If not NULL, the sheet ProjectID should always be equal to the id of the database related project.
	// A project must remove all linked sheets for a particular database before that database can be transferred to a different project.
	DatabaseID *int

	CreatorID int
	UpdaterID int

	Name       string
	Statement  string
	Visibility api.SheetVisibility
	Source     api.SheetSource
	Type       api.SheetType
	Payload    string

	// Output only fields
	UID         int
	Size        int64
	CreatedTime time.Time
	UpdatedTime time.Time
	Starred     bool
	Pinned      bool

	// Internal fields
	rowStatus api.RowStatus
	createdTs int64
	updatedTs int64
}

// GetSheetStatementByID gets the statement of a sheet by ID.
func (s *Store) GetSheetStatementByID(ctx context.Context, id int) (string, error) {
	if statement, ok := s.sheetStatementCache.Get(id); ok {
		return statement, nil
	}

	sheets, err := s.ListSheetsV2(ctx, &api.SheetFind{ID: &id, LoadFull: true}, api.SystemBotID)
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

func (s *Store) composeSheetMessage(ctx context.Context, sheetMessage *SheetMessage) (*api.Sheet, error) {
	creator, err := s.GetPrincipalByID(ctx, sheetMessage.CreatorID)
	if err != nil {
		return nil, err
	}
	updater, err := s.GetPrincipalByID(ctx, sheetMessage.UpdaterID)
	if err != nil {
		return nil, err
	}
	project, err := s.GetProjectByID(ctx, sheetMessage.ProjectUID)
	if err != nil {
		return nil, err
	}

	sheet := &api.Sheet{
		ID: sheetMessage.UID,

		RowStatus: sheetMessage.rowStatus,
		CreatorID: sheetMessage.CreatorID,
		Creator:   creator,
		CreatedTs: sheetMessage.createdTs,
		UpdaterID: sheetMessage.UpdaterID,
		Updater:   updater,
		UpdatedTs: sheetMessage.updatedTs,

		ProjectID: sheetMessage.ProjectUID,
		Project:   project,

		DatabaseID: sheetMessage.DatabaseID,
		Database:   nil,

		Name:       sheetMessage.Name,
		Statement:  sheetMessage.Statement,
		Visibility: sheetMessage.Visibility,
		Source:     sheetMessage.Source,
		Type:       sheetMessage.Type,
		Starred:    sheetMessage.Starred,
		Pinned:     sheetMessage.Pinned,

		Size: sheetMessage.Size,
	}

	if sheetMessage.DatabaseID != nil {
		database, err := s.GetDatabase(ctx, &api.DatabaseFind{ID: sheetMessage.DatabaseID})
		if err != nil {
			return nil, err
		}
		sheet.Database = database
	}

	return sheet, nil
}

// GetSheetV2 gets a sheet.
func (s *Store) GetSheetV2(ctx context.Context, find *api.SheetFind, currentPrincipalID int) (*SheetMessage, error) {
	sheets, err := s.ListSheetsV2(ctx, find, currentPrincipalID)
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

// GetSheetUsedByIssues returns a list of issues that have tasks that are using the sheet.
func (s *Store) GetSheetUsedByIssues(ctx context.Context, sheetID int) ([]int, error) {
	query := `
		SELECT ARRAY_AGG(issue.id)
		FROM issue
		JOIN task ON task.pipeline_id = issue.pipeline_id
		WHERE task.payload ? 'sheetId' AND (task.payload->>'sheetId')::INT = $1
	`

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var issueIDs []sql.NullInt32
	if err := tx.QueryRowContext(ctx, query, sheetID).Scan(pq.Array(&issueIDs)); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	var ids []int
	for _, id := range issueIDs {
		if id.Valid {
			ids = append(ids, int(id.Int32))
		}
	}

	return ids, nil
}

// ListSheetsV2 returns a list of sheets.
func (s *Store) ListSheetsV2(ctx context.Context, find *api.SheetFind, currentPrincipalID int) ([]*SheetMessage, error) {
	where, args := []string{"TRUE"}, []any{}

	if v := find.ID; v != nil {
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
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("sheet.project_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.DatabaseID; v != nil {
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
			&sheet.DatabaseID,
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

// CreateSheetV2 creates a new sheet.
func (s *Store) CreateSheetV2(ctx context.Context, create *SheetMessage) (*SheetMessage, error) {
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
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, LEFT(statement, %d), visibility, source, type, LENGTH(statement)
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
		create.DatabaseID,
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
		sheet.DatabaseID = &value
	}
	sheet.CreatedTime = time.Unix(sheet.createdTs, 0)
	sheet.UpdatedTime = time.Unix(sheet.updatedTs, 0)

	return &sheet, nil
}

// PatchSheetMessage is the message to patch a sheet.
type PatchSheetMessage struct {
	ID         int
	UpdaterID  int
	Name       *string
	Statement  *string
	Visibility *string
	// TODO(zp): update the payload.
	Payload *string
}

// PatchSheetV2 updates a sheet.
func (s *Store) PatchSheetV2(ctx context.Context, patch *PatchSheetMessage) (*SheetMessage, error) {
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

	args = append(args, patch.ID)

	var sheet SheetMessage
	databaseID := sql.NullInt32{}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin transaction")
	}
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
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("sheet ID not found: %d", patch.ID)}
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}
	s.sheetStatementCache.Invalidate(patch.ID)

	if databaseID.Valid {
		value := int(databaseID.Int32)
		sheet.DatabaseID = &value
	}
	sheet.CreatedTime = time.Unix(sheet.createdTs, 0)
	sheet.UpdatedTime = time.Unix(sheet.updatedTs, 0)
	return &sheet, nil
}
