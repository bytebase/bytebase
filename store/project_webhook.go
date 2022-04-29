package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgtype"

	"github.com/bytebase/bytebase/api"
	"github.com/bytebase/bytebase/common"
)

// projectWebhookRaw is the store model for an ProjectWebhook.
// Fields have exactly the same meanings as ProjectWebhook.
type projectWebhookRaw struct {
	ID int

	// Standard fields
	CreatorID int
	CreatedTs int64
	UpdaterID int
	UpdatedTs int64

	// Related fields
	ProjectID int

	// Domain specific fields
	Type         string
	Name         string
	URL          string
	ActivityList []string
}

// toProjectWebhook creates an instance of ProjectWebhook based on the projectWebhookRaw.
// This is intended to be called when we need to compose an ProjectWebhook relationship.
func (raw *projectWebhookRaw) toProjectWebhook() *api.ProjectWebhook {
	projectWebhook := api.ProjectWebhook{
		ID: raw.ID,

		// Standard fields
		CreatorID: raw.CreatorID,
		CreatedTs: raw.CreatedTs,
		UpdaterID: raw.UpdaterID,
		UpdatedTs: raw.UpdatedTs,

		// Related fields
		ProjectID: raw.ProjectID,

		// Domain specific fields
		Type: raw.Type,
		Name: raw.Name,
		URL:  raw.URL,
	}
	projectWebhook.ActivityList = append(projectWebhook.ActivityList, raw.ActivityList...)
	return &projectWebhook
}

// CreateProjectWebhook creates an instance of ProjectWebhook
func (s *Store) CreateProjectWebhook(ctx context.Context, create *api.ProjectWebhookCreate) (*api.ProjectWebhook, error) {
	projectWebhookRaw, err := s.createProjectWebhookRaw(ctx, create)
	if err != nil {
		return nil, fmt.Errorf("failed to create ProjectWebhook with ProjectWebhookCreate[%+v], error[%w]", create, err)
	}
	projectWebhook, err := s.composeProjectWebhook(ctx, projectWebhookRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose ProjectWebhook with projectWebhookRaw[%+v], error[%w]", projectWebhookRaw, err)
	}
	return projectWebhook, nil
}

// GetProjectWebhookByID gets an instance of ProjectWebhook
func (s *Store) GetProjectWebhookByID(ctx context.Context, id int) (*api.ProjectWebhook, error) {
	find := &api.ProjectWebhookFind{ID: &id}
	projectWebhookRaw, err := s.getProjectWebhookRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to get ProjectWebhook with ID[%d], error[%w]", id, err)
	}
	if projectWebhookRaw == nil {
		return nil, nil
	}
	projectWebhook, err := s.composeProjectWebhook(ctx, projectWebhookRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose ProjectWebhook with projectWebhookRaw[%+v], error[%w]", projectWebhookRaw, err)
	}
	return projectWebhook, nil
}

// FindProjectWebhook finds a list of ProjectWebhook instances
func (s *Store) FindProjectWebhook(ctx context.Context, find *api.ProjectWebhookFind) ([]*api.ProjectWebhook, error) {
	projectWebhookRawList, err := s.findProjectWebhookRaw(ctx, find)
	if err != nil {
		return nil, fmt.Errorf("failed to find ProjectWebhook list, error[%w]", err)
	}
	var projectWebhookList []*api.ProjectWebhook
	for _, raw := range projectWebhookRawList {
		projectWebhook, err := s.composeProjectWebhook(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("failed to compose ProjectWebhook with projectWebhookRaw[%+v], error[%w]", raw, err)
		}
		projectWebhookList = append(projectWebhookList, projectWebhook)
	}
	return projectWebhookList, nil
}

// PatchProjectWebhook patches an instance of ProjectWebhook
func (s *Store) PatchProjectWebhook(ctx context.Context, patch *api.ProjectWebhookPatch) (*api.ProjectWebhook, error) {
	projectWebhookRaw, err := s.patchProjectWebhookRaw(ctx, patch)
	if err != nil {
		return nil, fmt.Errorf("failed to patch ProjectWebhook with ProjectWebhookPatch[%+v], error[%w]", patch, err)
	}
	projectWebhook, err := s.composeProjectWebhook(ctx, projectWebhookRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to compose ProjectWebhook with projectWebhookRaw[%+v], error[%w]", projectWebhookRaw, err)
	}
	return projectWebhook, nil
}

// DeleteProjectWebhook deletes an existing projectWebhook by ID.
func (s *Store) DeleteProjectWebhook(ctx context.Context, delete *api.ProjectWebhookDelete) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return FormatError(err)
	}
	defer tx.PTx.Rollback()

	if err := deleteProjectWebhookImpl(ctx, tx.PTx, delete); err != nil {
		return FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return FormatError(err)
	}

	return nil
}

//
// private functions
//

func (s *Store) composeProjectWebhook(ctx context.Context, raw *projectWebhookRaw) (*api.ProjectWebhook, error) {
	webhook := raw.toProjectWebhook()

	creator, err := s.GetPrincipalByID(ctx, webhook.CreatorID)
	if err != nil {
		return nil, err
	}
	webhook.Creator = creator

	updater, err := s.GetPrincipalByID(ctx, webhook.UpdaterID)
	if err != nil {
		return nil, err
	}
	webhook.Updater = updater

	return webhook, nil
}

// createProjectWebhookRaw creates a new projectWebhook.
func (s *Store) createProjectWebhookRaw(ctx context.Context, create *api.ProjectWebhookCreate) (*projectWebhookRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	projectWebhook, err := createProjectWebhookImpl(ctx, tx.PTx, create)
	if err != nil {
		return nil, err
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return projectWebhook, nil
}

// findProjectWebhookRaw retrieves a list of projectWebhooks based on find.
func (s *Store) findProjectWebhookRaw(ctx context.Context, find *api.ProjectWebhookFind) ([]*projectWebhookRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findProjectWebhookImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	return list, nil
}

// getProjectWebhookRaw retrieves a single projectWebhook based on find.
// Returns ECONFLICT if finding more than 1 matching records.
func (s *Store) getProjectWebhookRaw(ctx context.Context, find *api.ProjectWebhookFind) (*projectWebhookRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	list, err := findProjectWebhookImpl(ctx, tx.PTx, find)
	if err != nil {
		return nil, err
	}

	if len(list) == 0 {
		return nil, nil
	} else if len(list) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: fmt.Errorf("found %d project hooks with filter %+v, expect 1", len(list), find)}
	}
	return list[0], nil
}

// patchProjectWebhookRaw updates an existing projectWebhook by ID.
// Returns ENOTFOUND if projectWebhook does not exist.
func (s *Store) patchProjectWebhookRaw(ctx context.Context, patch *api.ProjectWebhookPatch) (*projectWebhookRaw, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, FormatError(err)
	}
	defer tx.PTx.Rollback()

	projectWebhook, err := patchProjectWebhookImpl(ctx, tx.PTx, patch)
	if err != nil {
		return nil, FormatError(err)
	}

	if err := tx.PTx.Commit(); err != nil {
		return nil, FormatError(err)
	}

	return projectWebhook, nil
}

// createProjectWebhookImpl creates a new projectWebhook.
func createProjectWebhookImpl(ctx context.Context, tx *sql.Tx, create *api.ProjectWebhookCreate) (*projectWebhookRaw, error) {
	// Insert row into database.
	row, err := tx.QueryContext(ctx, `
		INSERT INTO project_webhook (
			creator_id,
			updater_id,
			project_id,
			type,
			name,
			url,
			activity_list
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, type, name, url, activity_list
	`,
		create.CreatorID,
		create.CreatorID,
		create.ProjectID,
		create.Type,
		create.Name,
		create.URL,
		create.ActivityList,
	)

	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	row.Next()
	var projectWebhookRaw projectWebhookRaw
	var txtArray pgtype.TextArray

	if err := row.Scan(
		&projectWebhookRaw.ID,
		&projectWebhookRaw.CreatorID,
		&projectWebhookRaw.CreatedTs,
		&projectWebhookRaw.UpdaterID,
		&projectWebhookRaw.UpdatedTs,
		&projectWebhookRaw.ProjectID,
		&projectWebhookRaw.Type,
		&projectWebhookRaw.Name,
		&projectWebhookRaw.URL,
		&txtArray,
	); err != nil {
		return nil, FormatError(err)
	}

	if err := txtArray.AssignTo(&projectWebhookRaw.ActivityList); err != nil {
		return nil, FormatError(err)
	}

	return &projectWebhookRaw, nil
}

func findProjectWebhookImpl(ctx context.Context, tx *sql.Tx, find *api.ProjectWebhookFind) ([]*projectWebhookRaw, error) {
	// Build WHERE clause.
	where, args := []string{"1 = 1"}, []interface{}{}
	if v := find.ID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.ProjectID; v != nil {
		where, args = append(where, fmt.Sprintf("project_id = $%d", len(args)+1)), append(args, *v)
	}

	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			creator_id,
			created_ts,
			updater_id,
			updated_ts,
			project_id,
			type,
			name,
			url,
			activity_list
		FROM project_webhook
		WHERE `+strings.Join(where, " AND "),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer rows.Close()

	// Iterate over result set and deserialize rows into projectWebhookRawList.
	var projectWebhookRawList []*projectWebhookRaw
	for rows.Next() {
		var projectWebhookRaw projectWebhookRaw
		var txtArray pgtype.TextArray

		if err := rows.Scan(
			&projectWebhookRaw.ID,
			&projectWebhookRaw.CreatorID,
			&projectWebhookRaw.CreatedTs,
			&projectWebhookRaw.UpdaterID,
			&projectWebhookRaw.UpdatedTs,
			&projectWebhookRaw.ProjectID,
			&projectWebhookRaw.Type,
			&projectWebhookRaw.Name,
			&projectWebhookRaw.URL,
			&txtArray,
		); err != nil {
			return nil, FormatError(err)
		}

		if err := txtArray.AssignTo(&projectWebhookRaw.ActivityList); err != nil {
			return nil, FormatError(err)
		}

		if v := find.ActivityType; v != nil {
			for _, activity := range projectWebhookRaw.ActivityList {
				if api.ActivityType(activity) == *v {
					projectWebhookRawList = append(projectWebhookRawList, &projectWebhookRaw)
					break
				}
			}
		} else {
			projectWebhookRawList = append(projectWebhookRawList, &projectWebhookRaw)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, FormatError(err)
	}

	return projectWebhookRawList, nil
}

// patchProjectWebhookImpl updates a projectWebhook by ID. Returns the new state of the projectWebhook after update.
func patchProjectWebhookImpl(ctx context.Context, tx *sql.Tx, patch *api.ProjectWebhookPatch) (*projectWebhookRaw, error) {
	// Build UPDATE clause.
	set, args := []string{"updater_id = $1"}, []interface{}{patch.UpdaterID}
	if v := patch.Name; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.URL; v != nil {
		set, args = append(set, fmt.Sprintf("url = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.ActivityList; v != nil {
		activities := strings.Split(*v, ",")
		set, args = append(set, fmt.Sprintf("activity_list = $%d", len(args)+1)), append(args, activities)
	}

	args = append(args, patch.ID)

	// Execute update query with RETURNING.
	row, err := tx.QueryContext(ctx, fmt.Sprintf(`
		UPDATE project_webhook
		SET `+strings.Join(set, ", ")+`
		WHERE id = $%d
		RETURNING id, creator_id, created_ts, updater_id, updated_ts, project_id, type, name, url, activity_list
	`, len(args)),
		args...,
	)
	if err != nil {
		return nil, FormatError(err)
	}
	defer row.Close()

	if row.Next() {
		var projectWebhookRaw projectWebhookRaw
		var txtArray pgtype.TextArray

		if err := row.Scan(
			&projectWebhookRaw.ID,
			&projectWebhookRaw.CreatorID,
			&projectWebhookRaw.CreatedTs,
			&projectWebhookRaw.UpdaterID,
			&projectWebhookRaw.UpdatedTs,
			&projectWebhookRaw.ProjectID,
			&projectWebhookRaw.Type,
			&projectWebhookRaw.Name,
			&projectWebhookRaw.URL,
			&txtArray,
		); err != nil {
			return nil, FormatError(err)
		}

		if err := txtArray.AssignTo(&projectWebhookRaw.ActivityList); err != nil {
			return nil, FormatError(err)
		}

		return &projectWebhookRaw, nil
	}

	return nil, &common.Error{Code: common.NotFound, Err: fmt.Errorf("project hook ID not found: %d", patch.ID)}
}

// deleteProjectWebhookImpl permanently deletes a projectWebhook by ID.
func deleteProjectWebhookImpl(ctx context.Context, tx *sql.Tx, delete *api.ProjectWebhookDelete) error {
	// Remove row from database.
	if _, err := tx.ExecContext(ctx, `DELETE FROM project_webhook WHERE id = $1`, delete.ID); err != nil {
		return FormatError(err)
	}
	return nil
}
