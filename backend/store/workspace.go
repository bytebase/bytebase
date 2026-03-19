package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/genproto/googleapis/type/expr"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// WorkspaceMessage is the message for a workspace.
type WorkspaceMessage struct {
	ResourceID string
	Name       string
}

// getWorkspace returns the workspace. Returns (nil, nil) if no workspace exists.
func (s *Store) getWorkspace(ctx context.Context) (*WorkspaceMessage, error) {
	var workspace WorkspaceMessage
	if err := s.GetDB().QueryRowContext(ctx,
		`SELECT resource_id, name FROM workspace WHERE deleted = FALSE LIMIT 1`,
	).Scan(&workspace.ResourceID, &workspace.Name); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get workspace")
	}
	return &workspace, nil
}

// GetWorkspaceID returns the workspace resource ID.
// Returns ("", nil) if no workspace exists.
func (s *Store) GetWorkspaceID(ctx context.Context) (string, error) {
	ws, err := s.getWorkspace(ctx)
	if err != nil {
		return "", err
	}
	if ws == nil {
		return "", nil
	}
	return ws.ResourceID, nil
}

// CreateWorkspace creates a new workspace and initializes its default settings and IAM policy.
func (s *Store) CreateWorkspace(ctx context.Context, create *WorkspaceMessage, adminEmail string) (*WorkspaceMessage, error) {
	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	// Create workspace.
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO workspace (resource_id, name) VALUES ($1, $2)`,
		create.ResourceID, create.Name,
	); err != nil {
		return nil, errors.Wrap(err, "failed to create workspace")
	}

	// Initialize default settings for the workspace.
	type defaultSetting struct {
		name    storepb.SettingName
		payload proto.Message
	}
	settings := []defaultSetting{
		{storepb.SettingName_SYSTEM, &storepb.SystemSetting{}},
		{storepb.SettingName_APP_IM, &storepb.AppIMSetting{}},
		{storepb.SettingName_DATA_CLASSIFICATION, &storepb.DataClassificationSetting{}},
		{storepb.SettingName_WORKSPACE_APPROVAL, &storepb.WorkspaceApprovalSetting{
			Rules: []*storepb.WorkspaceApprovalSetting_Rule{
				{
					Template: &storepb.ApprovalTemplate{
						Flow:        &storepb.ApprovalFlow{Roles: []string{"roles/projectOwner"}},
						Title:       "Fallback Rule",
						Description: "Requires project owner approval when no other rules match.",
					},
					Condition: &expr.Expr{Expression: "true"},
				},
			},
		}},
		{storepb.SettingName_WORKSPACE_PROFILE, &storepb.WorkspaceProfileSetting{
			EnableMetricCollection: true,
			DirectorySyncToken:     uuid.New().String(),
			DisallowSignup:         true,
			PasswordRestriction:    &storepb.WorkspaceProfileSetting_PasswordRestriction{MinLength: 8},
		}},
		{storepb.SettingName_ENVIRONMENT, &storepb.EnvironmentSetting{
			Environments: []*storepb.EnvironmentSetting_Environment{
				{Title: "Test", Id: "test"},
				{Title: "Prod", Id: "prod"},
			},
		}},
	}
	for _, s := range settings {
		value, err := protojson.Marshal(s.payload)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal setting %s", s.name)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO setting (name, workspace, value) VALUES ($1, $2, $3)`,
			s.name.String(), create.ResourceID, string(value),
		); err != nil {
			return nil, errors.Wrapf(err, "failed to create setting %s", s.name)
		}
	}

	// Initialize workspace IAM policy — add the creator as workspace admin.
	iamPolicy := &storepb.IamPolicy{
		Bindings: []*storepb.Binding{
			{
				Role:    "roles/workspaceAdmin",
				Members: []string{fmt.Sprintf("users/%s", adminEmail)},
			},
		},
	}
	iamPayload, err := protojson.Marshal(iamPolicy)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal IAM policy")
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO policy (workspace, resource_type, resource, type, payload, inherit_from_parent, enforce)
		 VALUES ($1, 'WORKSPACE', 'workspaces/' || $1, 'IAM', $2, FALSE, TRUE)`,
		create.ResourceID, string(iamPayload),
	); err != nil {
		return nil, errors.Wrap(err, "failed to create workspace IAM policy")
	}

	// Create sample project with a globally unique resource_id.
	projectID, err := common.RandomString(8)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate project ID")
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO project (resource_id, workspace, name, setting) VALUES ($1, $2, 'Sample Project', '{}')`,
		fmt.Sprintf("sample-project-%s", projectID), create.ResourceID,
	); err != nil {
		return nil, errors.Wrap(err, "failed to create sample project")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit workspace creation")
	}

	return create, nil
}

func (s *Store) FindWorkspaceIDByMemberEmail(ctx context.Context, memberName string, includeAllUser bool) (string, error) {
	workspaces, err := s.FindWorkspacesByMemberEmail(ctx, memberName, includeAllUser)
	if err != nil {
		return "", errors.Wrap(err, "failed to find workspaces for user")
	}
	if len(workspaces) == 0 {
		return "", errors.Errorf("%q is not a member of any workspace", memberName)
	}
	// TODO(ed): In SaaS mode with multiple workspaces, return a workspace picker
	// instead of auto-selecting the first one. For now, use the first workspace.
	return workspaces[0].ResourceID, nil
}

// FindWorkspacesByMemberEmail finds all workspaces where the given email is a member
// in the workspace IAM policy bindings. The memberName should be in the format
// "users/{email}", "serviceAccounts/{email}", etc.
// Returns workspaces sorted by name.
func (s *Store) FindWorkspacesByMemberEmail(ctx context.Context, memberName string, includeAllUser bool) ([]*WorkspaceMessage, error) {
	memberFilter := qb.Q().Space("member = ?", memberName)
	if includeAllUser {
		memberFilter.Or("member = ?", common.AllUsers)
	}

	q := qb.Q().Space(`
		SELECT DISTINCT w.resource_id, w.name
		FROM workspace w
		JOIN policy p ON p.workspace = w.resource_id
		WHERE p.resource_type = 'WORKSPACE'
		  AND p.type = 'IAM'
		  AND w.deleted = FALSE
		  AND EXISTS (
			SELECT 1
			FROM jsonb_array_elements(p.payload->'bindings') AS binding,
			     jsonb_array_elements_text(binding->'members') AS member
			WHERE ?
		  )
		ORDER BY w.name
	`, memberFilter)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find workspaces by member")
	}
	defer rows.Close()

	var workspaces []*WorkspaceMessage
	for rows.Next() {
		var ws WorkspaceMessage
		if err := rows.Scan(&ws.ResourceID, &ws.Name); err != nil {
			return nil, errors.Wrap(err, "failed to scan workspace")
		}
		workspaces = append(workspaces, &ws)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to iterate workspaces")
	}
	return workspaces, nil
}
