package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/pkg/errors"
)

// migration3_25_0ApprovalOnlyWorkspaces must match approval_only_workspace in
// migration/3.25/0000##sql_export_permission.sql.
const migration3_25_0ApprovalOnlyWorkspaces = `
	SELECT workspace FROM policy
	WHERE resource_type = 'WORKSPACE' AND type = 'QUERY_DATA' AND enforce
	  AND COALESCE((payload ->> 'disableExport')::boolean, false)`

// sqlExportMigrationSummary is what the 3.25.0 SQL migration changes in one workspace.
type sqlExportMigrationSummary struct {
	Workspace    string
	ApprovalOnly bool
	// ExporterBindings maps an IAM policy resource, such as workspaces/{id} or
	// projects/{id}, to the number of SQL Editor Exporter bindings added to it.
	ExporterBindings map[string]int
	// RolesGainingExport are custom role ids that gain bb.sql.export.
	RolesGainingExport []string
	// RemovedDisableExport maps a query data policy resource to the JSON value
	// of the disableExport key removed from it.
	RemovedDisableExport map[string]string
}

type migration3_25_0Queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// migrate3_25_0 only logs; migration/3.25/0000##sql_export_permission.sql makes the changes.
func migrate3_25_0(ctx context.Context, conn *sql.Conn) error {
	summaries, err := summarizeMigration3_25_0(ctx, conn)
	if err != nil {
		return err
	}
	for _, s := range summaries {
		slog.Info(s.String())
	}
	return nil
}

// summarizeMigration3_25_0 returns one summary per workspace the SQL migration
// will change, sorted by workspace.
func summarizeMigration3_25_0(ctx context.Context, q migration3_25_0Queryer) ([]*sqlExportMigrationSummary, error) {
	byWorkspace := map[string]*sqlExportMigrationSummary{}
	get := func(workspace string) *sqlExportMigrationSummary {
		s, ok := byWorkspace[workspace]
		if !ok {
			s = &sqlExportMigrationSummary{
				Workspace:            workspace,
				ExporterBindings:     map[string]int{},
				RemovedDisableExport: map[string]string{},
			}
			byWorkspace[workspace] = s
		}
		return s
	}

	approvalOnly := map[string]bool{}
	if err := queryMigration3_25_0(ctx, q, migration3_25_0ApprovalOnlyWorkspaces, func(rows *sql.Rows) error {
		var workspace string
		if err := rows.Scan(&workspace); err != nil {
			return err
		}
		approvalOnly[workspace] = true
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "failed to list approval-only workspaces")
	}

	if err := queryMigration3_25_0(ctx, q, `
		SELECT workspace, resource_id FROM role
		WHERE permissions -> 'permissions' @> '["bb.sql.select"]'
		  AND NOT permissions -> 'permissions' @> '["bb.sql.export"]'
		  AND workspace NOT IN (`+migration3_25_0ApprovalOnlyWorkspaces+`)
		ORDER BY workspace, resource_id`, func(rows *sql.Rows) error {
		var workspace, role string
		if err := rows.Scan(&workspace, &role); err != nil {
			return err
		}
		s := get(workspace)
		s.RolesGainingExport = append(s.RolesGainingExport, role)
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "failed to list custom roles gaining bb.sql.export")
	}

	if err := queryMigration3_25_0(ctx, q, `
		SELECT p.workspace, p.resource, count(*)
		FROM policy p, jsonb_array_elements(p.payload -> 'bindings') AS b
		WHERE p.type = 'IAM'
		  AND p.resource_type IN ('WORKSPACE', 'PROJECT')
		  AND p.workspace NOT IN (`+migration3_25_0ApprovalOnlyWorkspaces+`)
		  AND b ->> 'role' IN ('roles/sqlEditorUser', 'roles/sqlEditorReadUser')
		GROUP BY p.workspace, p.resource`, func(rows *sql.Rows) error {
		var workspace, resource string
		var count int
		if err := rows.Scan(&workspace, &resource, &count); err != nil {
			return err
		}
		get(workspace).ExporterBindings[resource] = count
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "failed to count SQL Editor Exporter bindings")
	}

	if err := queryMigration3_25_0(ctx, q, `
		SELECT workspace, resource, (payload -> 'disableExport')::text
		FROM policy
		WHERE type = 'QUERY_DATA' AND payload ? 'disableExport'`, func(rows *sql.Rows) error {
		var workspace, resource, value string
		if err := rows.Scan(&workspace, &resource, &value); err != nil {
			return err
		}
		get(workspace).RemovedDisableExport[resource] = value
		return nil
	}); err != nil {
		return nil, errors.Wrap(err, "failed to list query data policies with disableExport")
	}

	summaries := make([]*sqlExportMigrationSummary, 0, len(byWorkspace))
	for workspace, s := range byWorkspace {
		s.ApprovalOnly = approvalOnly[workspace]
		summaries = append(summaries, s)
	}
	slices.SortFunc(summaries, func(a, b *sqlExportMigrationSummary) int {
		return strings.Compare(a.Workspace, b.Workspace)
	})
	return summaries, nil
}

func queryMigration3_25_0(ctx context.Context, q migration3_25_0Queryer, query string, scan func(*sql.Rows) error) error {
	rows, err := q.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := scan(rows); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (s *sqlExportMigrationSummary) String() string {
	mode := "export-enabled"
	if s.ApprovalOnly {
		mode = "approval-only"
	}
	return fmt.Sprintf(
		"SQL export migration: workspace %q is %s; SQL Editor Exporter bindings added: %s; custom roles gaining bb.sql.export: %s; disableExport removed: %s.",
		s.Workspace,
		mode,
		formatMigration3_25_0Map(s.ExporterBindings, func(v int) string { return fmt.Sprint(v) }),
		formatMigration3_25_0List(s.RolesGainingExport),
		formatMigration3_25_0Map(s.RemovedDisableExport, func(v string) string { return v }),
	)
}

func formatMigration3_25_0Map[V any](m map[string]V, format func(V) string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, format(m[k])))
	}
	return formatMigration3_25_0List(parts)
}

func formatMigration3_25_0List(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}
