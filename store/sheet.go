package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
	"github.com/bytebase/bytebase/metric"
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
	// Payload is in the json string format of SheetVCSPayload.
	Payload string
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
	}
}

// CreateSheet creates an instance of Sheet
func (s *Store) CreateSheet(ctx context.Context, create *api.SheetCreate) (*api.Sheet, error) {
	sheetRaw, err := s.createSheetRaw(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("failed to create Sheet with SheetCreate[%+v], error[%w]", create, err)
	}
	sheet, err := s.composeSheet(ctx, sheetRaw, create.CreatorID)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Sheet with sheetRaw[%+v], error[%w]", sheetRaw, err)
	}
	return sheet, nil
}

// GetSheet gets an instance of Sheet
func (s *Store) GetSheet(ctx context.Context, find *api.SheetFind, currentPrincipalID int) (*api.Sheet, error) {
	sheetRaw, err := s.getSheetRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to get Sheet with SheetFind[%+v], error[%w]", find, err)
	}
	if sheetRaw == nil {
		return nil, nil
	}
	sheet, err := s.composeSheet(ctx, sheetRaw, currentPrincipalID)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Sheet with sheetRaw[%+v], error[%w]", sheetRaw, err)
	}
	return sheet, nil
}

// FindSheet finds a list of Sheet instances
func (s *Store) FindSheet(ctx context.Context, find *api.SheetFind, currentPrincipalID int) ([]*api.Sheet, error) {
	sheetRawList, err := s.findSheetRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to find Sheet list, error[%w]", err)
	}
	var sheetList []*api.Sheet
	for _, raw := range sheetRawList {
		sheet, err := s.composeSheet(ctx, raw, currentPrincipalID)
		if err != nil {
			return nil, fmt.Errorf("failed to compose Sheet with sheetRaw[%+v], error[%w]", raw, err)
		}
		sheetList = append(sheetList, sheet)
	}
	return sheetList, nil
}

// PatchSheet patches an instance of Sheet
func (s *Store) PatchSheet(ctx context.Context, patch *api.SheetPatch) (*api.Sheet, error) {
	sheetRaw, err := s.patchSheetRaw(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("failed to patch Sheet with SheetPatch[%+v], error[%w]", patch, err)
	}
	sheet, err := s.composeSheet(ctx, sheetRaw, patch.UpdaterID)
	if err != nil {
		return nil, fmt.Errorf("failed to compose Sheet with sheetRaw[%+v], error[%w]", sheetRaw, err)
	}
	return sheet, nil
}

// DeleteSheet deletes an existing sheet by ID.
// Returns ENOTFOUND if sheet does not exist.
func (s *Store) DeleteSheet(ctx context.Context, delete *api.SheetDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.PTx.Rollback()

	if err := deleteSheet(ctx, tx.PTx, delete); err != nil {
		return FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

// CountSheetGroupByRowstatusVisibilitySourceAndType counts the number of sheets group by row_status, visibility, source and type.
func (s *Store) CountSheetGroupByRowstatusVisibilitySourceAndType(ctx context.Context) ([]*metric.SheetCountMetric, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	rows, err := tx.PTx.QueryContext(ctx, `
		SELECT row_status, visibility, source, type, COUNT(*) AS count
		FROM sheet
		GROUP BY row_status, visibility, source, type`)
	if err != nil {
		return nil, FormatError(err)
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
			return nil, FormatError(err)
		}
		res = append(res, &sheetCount)
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
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	sheet, err := createSheetImpl(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return sheet, nil
}

// patchSheetRaw updates an existing sheet by ID.
func (s *Store) patchSheetRaw(ctx context.Context, patch *api.SheetPatch) (*sheetRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	sheet, err := patchSheetImpl(ctx, tx.PTx, patch)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return sheet, nil
}

// findSheetRaw retrieves a list of sheet based on find.
func (s *Store) findSheetRaw(ctx context.Context, find *api.SheetFind) ([]*sheetRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findSheetImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// getSheetRaw retrieves a single sheet based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getSheetRaw(ctx context.Context, find *api.SheetFind) (*sheetRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findSheetImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d sheet with filter %+v, expect 1. ", len(list), find)}
	}
	return list[0], nil
}

// createSheetImpl creates a new sheet.
func createSheetImpl(ctx context.Context, tx *sql.Tx, create *api.SheetCreate) (*sheetRaw, error) {
	if create.Payload == "" {
		create.Payload = "{}"
	}

	row, err := tx.QueryContext(ctx, `
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
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, statement, visibility, source, type, payload
	`,
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
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var sheetRaw sheetRaw
	databaseID := sql.NullInt32{}
	if err := row.Scan(
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
	); err != nil {
		return nil, FormatError(err)
	}

	if databaseID.Valid {
		value := int(databaseID.Int32)
		sheetRaw.DatabaseID = &value
	}

	return &sheetRaw, nil
}

// patchSheetImpl updates a sheet's name/statement/visibility.
func patchSheetImpl(ctx context.Context, tx *sql.Tx, patch *api.SheetPatch) (*sheetRaw, error) {
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.RowStatus; v != nil {
		set, args = append(set, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, api.RowStatus(*v))
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

	row, err := tx.QueryContext(ctx, fmt.Sprintf(`
		UPDATE sheet
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, row_status, creator_id, created_ts, updater_id, updated_ts, project_id, database_id, name, statement, visibility, source, type, payload
	`, len(args)),
		args...,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var sheetRaw sheetRaw
		databaseID := sql.NullInt32{}
		if err := row.Scan(
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
		); err != nil {
			return nil, FormatError(err)
		}

		if databaseID.Valid {
			value := int(databaseID.Int32)
			sheetRaw.DatabaseID = &value
		}

		return &sheetRaw, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("sheet ID not found: %d", patch.ID)}

}

func findSheetImpl(ctx context.Context, tx *sql.Tx, find *api.SheetFind) ([]*sheetRaw, error) {
	where, args := []string{"1 = 1"}, []interface{}{}

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
	if v := find.Visibility; v != nil {
		where, args = append(where, fmt.Sprintf("visibility = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.PrincipalID; v != nil {
		where, args = append(where, fmt.Sprintf("project_id IN (SELECT project_id FROM project_member WHERE principal_id = $%d)", len(args)+1)), append(args, *v)
	}
	if v := find.OrganizerID; v != nil {
		// For now, we only need the starred sheets.
		where, args = append(where, fmt.Sprintf("id IN (SELECT sheet_id FROM sheet_organizer WHERE principal_id = $%d AND starred = true)", len(args)+1)), append(args, *v)
	}
	if v := find.Source; v != nil {
		where, args = append(where, fmt.Sprintf("source = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.Type; v != nil {
		where, args = append(where, fmt.Sprintf("type = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
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
			statement,
			visibility,
			source,
			type,
			payload
		FROM sheet
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
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
		); err != nil {
			return nil, FormatError(err)
		}

		if databaseID.Valid {
			value := int(databaseID.Int32)
			sheetRaw.DatabaseID = &value
		}

		sheetRawList = append(sheetRawList, &sheetRaw)
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return sheetRawList, nil
}

// deleteSheet permanently deletes a sheet by ID.
func deleteSheet(ctx context.Context, tx *sql.Tx, delete *api.SheetDelete) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM sheet WHERE id = $1`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}
