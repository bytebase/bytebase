package migrator

import (
	"context"
	"database/sql"
	"strconv"
	"time"

	"github.com/lib/pq"
	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/store"
)

const migration3_21_1BatchSize = 100

type uiPlanDraftCandidate struct {
	planID      int64
	creator     string
	projectID   string
	title       string
	description string
}

type migration3_21_1Queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func migrate3_21_1(ctx context.Context, conn *sql.Conn) error {
	var migrationTime time.Time
	if err := conn.QueryRowContext(ctx, `SELECT CURRENT_TIMESTAMP`).Scan(&migrationTime); err != nil {
		return errors.Wrap(err, "failed to get migration time")
	}
	return migrate3_21_1At(ctx, conn, migrationTime)
}

func migrate3_21_1At(ctx context.Context, conn *sql.Conn, migrationTime time.Time) error {
	afterProjectID := ""
	for {
		projectIDs, err := listMigration3_21_1ProjectPage(ctx, conn, afterProjectID)
		if err != nil {
			return err
		}
		if len(projectIDs) == 0 {
			return nil
		}
		highWaterByProject, err := captureMigration3_21_1HighWater(ctx, conn, projectIDs)
		if err != nil {
			return err
		}
		for _, projectID := range projectIDs {
			highWater, ok := highWaterByProject[projectID]
			if !ok {
				continue
			}
			var afterPlanID int64
			for {
				candidates, err := findMigration3_21_1Candidates(
					ctx,
					conn,
					migrationTime,
					projectID,
					nil,
					&afterPlanID,
					&highWater,
					migration3_21_1BatchSize,
					false,
				)
				if err != nil {
					return err
				}
				if len(candidates) == 0 {
					break
				}
				if err := migrate3_21_1Batch(ctx, conn, migrationTime, candidates); err != nil {
					return err
				}
				afterPlanID = candidates[len(candidates)-1].planID
			}
		}
		afterProjectID = projectIDs[len(projectIDs)-1]
	}
}

func listMigration3_21_1ProjectPage(ctx context.Context, conn *sql.Conn, afterProjectID string) ([]string, error) {
	rows, err := conn.QueryContext(ctx, `
		SELECT resource_id
		FROM project
		WHERE resource_id > $1
		ORDER BY resource_id
		LIMIT $2`, afterProjectID, migration3_21_1BatchSize)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list projects for UI Plan draft backfill")
	}
	defer rows.Close()

	var projectIDs []string
	for rows.Next() {
		var projectID string
		if err := rows.Scan(&projectID); err != nil {
			return nil, errors.Wrap(err, "failed to scan project for UI Plan draft backfill")
		}
		projectIDs = append(projectIDs, projectID)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to iterate projects for UI Plan draft backfill")
	}
	return projectIDs, nil
}

func captureMigration3_21_1HighWater(ctx context.Context, conn *sql.Conn, projectIDs []string) (map[string]int64, error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin UI Plan high-water capture")
	}
	defer tx.Rollback()

	var lockedProjectIDs []string
	if err := func() error {
		rows, err := tx.QueryContext(ctx, `
			SELECT resource_id
			FROM project
			WHERE resource_id = ANY($1)
			ORDER BY resource_id
			FOR UPDATE`, pq.Array(projectIDs))
		if err != nil {
			return errors.Wrap(err, "failed to lock projects for UI Plan high-water capture")
		}
		defer rows.Close()
		for rows.Next() {
			var projectID string
			if err := rows.Scan(&projectID); err != nil {
				return errors.Wrap(err, "failed to scan locked project")
			}
			lockedProjectIDs = append(lockedProjectIDs, projectID)
		}
		return errors.Wrap(rows.Err(), "failed to iterate locked projects")
	}(); err != nil {
		return nil, err
	}

	highWaterByProject := make(map[string]int64)
	if len(lockedProjectIDs) > 0 {
		rows, err := tx.QueryContext(ctx, `
			SELECT locked_project.project_id, latest_plan.id
			FROM unnest($1::TEXT[]) AS locked_project(project_id)
			CROSS JOIN LATERAL (
				SELECT plan.id
				FROM plan
				WHERE plan.project = locked_project.project_id
				ORDER BY plan.id DESC
				LIMIT 1
			) AS latest_plan
			ORDER BY locked_project.project_id`, pq.Array(lockedProjectIDs))
		if err != nil {
			return nil, errors.Wrap(err, "failed to capture UI Plan high-water marks")
		}
		defer rows.Close()
		for rows.Next() {
			var projectID string
			var highWater int64
			if err := rows.Scan(&projectID, &highWater); err != nil {
				return nil, errors.Wrap(err, "failed to scan UI Plan high-water mark")
			}
			highWaterByProject[projectID] = highWater
		}
		if err := rows.Err(); err != nil {
			return nil, errors.Wrap(err, "failed to iterate UI Plan high-water marks")
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit UI Plan high-water capture")
	}
	return highWaterByProject, nil
}

func migrate3_21_1Batch(ctx context.Context, conn *sql.Conn, migrationTime time.Time, candidates []uiPlanDraftCandidate) error {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin UI Plan draft backfill batch")
	}
	defer tx.Rollback()

	projectID := candidates[0].projectID
	planIDs := make([]int64, 0, len(candidates))
	for _, candidate := range candidates {
		key := candidate.projectID + "/" + strconv.FormatInt(candidate.planID, 10)
		if err := store.AcquireAdvisoryXactLockWithStringKey(
			ctx,
			tx,
			store.AdvisoryLockKeyPlanIssueRollout,
			key,
		); err != nil {
			return errors.Wrapf(err, "failed to lock Plan %s", key)
		}
		planIDs = append(planIDs, candidate.planID)
	}

	eligible, err := findMigration3_21_1Candidates(ctx, tx, migrationTime, projectID, planIDs, nil, nil, 0, true)
	if err != nil {
		return err
	}
	if len(eligible) == 0 {
		if err := tx.Commit(); err != nil {
			return errors.Wrap(err, "failed to commit empty UI Plan draft backfill batch")
		}
		return nil
	}

	var lockedProjectID string
	if err := tx.QueryRowContext(ctx, `
		SELECT resource_id
		FROM project
		WHERE resource_id = $1
		FOR UPDATE`, projectID).Scan(&lockedProjectID); err != nil {
		return errors.Wrapf(err, "failed to lock project %s", projectID)
	}

	var issueID int64
	if err := tx.QueryRowContext(ctx, `
		SELECT GREATEST(COALESCE(MAX(id), 0), 100)
		FROM issue
		WHERE project = $1`, projectID).Scan(&issueID); err != nil {
		return errors.Wrapf(err, "failed to get maximum issue ID for project %s", projectID)
	}

	statement, err := tx.PrepareContext(ctx, `
		INSERT INTO issue (
			id, creator, created_at, updated_at, project, plan_id,
			name, status, type, description, payload, ts_vector
		) VALUES ($1, $2, $3, $3, $4, $5, $6, 'OPEN', 'DATABASE_CHANGE', $7, '{"draft":true}', $8)`)
	if err != nil {
		return errors.Wrap(err, "failed to prepare UI Plan draft backfill")
	}
	defer statement.Close()

	for _, candidate := range eligible {
		issueID++
		if _, err := statement.ExecContext(
			ctx,
			issueID,
			candidate.creator,
			migrationTime,
			candidate.projectID,
			candidate.planID,
			candidate.title,
			candidate.description,
			store.IssueSearchVector(candidate.title, candidate.description),
		); err != nil {
			return errors.Wrapf(err, "failed to backfill draft issue for Plan %s/%d", candidate.projectID, candidate.planID)
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit UI Plan draft backfill batch")
	}
	return nil
}

func findMigration3_21_1Candidates(
	ctx context.Context,
	queryer migration3_21_1Queryer,
	migrationTime time.Time,
	projectID string,
	planIDs []int64,
	afterPlanID *int64,
	highWater *int64,
	limit int,
	lock bool,
) ([]uiPlanDraftCandidate, error) {
	query := `
		SELECT plan.id, plan.creator, plan.project, plan.name, plan.description
		FROM plan
		WHERE NOT plan.deleted
		  AND COALESCE(plan.config->>'hasRollout', 'false') = 'false'
		  AND plan.created_at >= $1::TIMESTAMPTZ - INTERVAL '30 days'
		  AND plan.project = $2
		  AND ($3::BIGINT[] IS NULL OR plan.id = ANY($3))
		  AND ($4::BIGINT IS NULL OR plan.id > $4)
		  AND ($5::BIGINT IS NULL OR plan.id <= $5)
		  AND NOT EXISTS (
		      SELECT 1
		      FROM issue linked_issue
		      WHERE linked_issue.project = plan.project
		        AND linked_issue.plan_id = plan.id
		  )
		  AND jsonb_typeof(plan.config->'specs') = 'array'
		  AND jsonb_array_length(plan.config->'specs') > 0
		  AND (
		      NOT EXISTS (
		          SELECT 1
		          FROM jsonb_array_elements(plan.config->'specs') AS spec
		          WHERE jsonb_typeof(spec->'createDatabaseConfig') IS DISTINCT FROM 'object'
		             OR spec ? 'changeDatabaseConfig'
		             OR spec ? 'exportDataConfig'
		      )
		      OR NOT EXISTS (
		          SELECT 1
		          FROM jsonb_array_elements(plan.config->'specs') AS spec
		          WHERE jsonb_typeof(spec->'changeDatabaseConfig') IS DISTINCT FROM 'object'
		             OR spec ? 'createDatabaseConfig'
		             OR spec ? 'exportDataConfig'
		             OR NULLIF(spec->'changeDatabaseConfig'->>'release', '') IS NOT NULL
		      )
		  )
		ORDER BY plan.project, plan.id`
	if limit > 0 {
		query += " LIMIT $6"
	}
	if lock {
		query += " FOR UPDATE OF plan"
	}

	var planIDsArg any
	if len(planIDs) > 0 {
		planIDsArg = pq.Array(planIDs)
	}
	args := []any{migrationTime, projectID, planIDsArg, afterPlanID, highWater}
	if limit > 0 {
		args = append(args, limit)
	}
	rows, err := queryer.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list UI Plan draft candidates")
	}
	defer rows.Close()

	var candidates []uiPlanDraftCandidate
	for rows.Next() {
		var candidate uiPlanDraftCandidate
		if err := rows.Scan(
			&candidate.planID,
			&candidate.creator,
			&candidate.projectID,
			&candidate.title,
			&candidate.description,
		); err != nil {
			return nil, errors.Wrap(err, "failed to scan UI Plan draft candidate")
		}
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to iterate UI Plan draft candidates")
	}
	return candidates, nil
}
