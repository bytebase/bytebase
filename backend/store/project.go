package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	api "github.com/bytebase/bytebase/backend/legacyapi"
	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// ProjectMessage is the message for project.
type ProjectMessage struct {
	ResourceID                 string
	Title                      string
	Key                        string
	TenantMode                 api.ProjectTenantMode
	Webhooks                   []*ProjectWebhookMessage
	DataClassificationConfigID string
	Setting                    *storepb.Project
	VCSConnectorsCount         int
	// The following fields are output only and not used for create().
	UID     int
	Deleted bool
}

func (p *ProjectMessage) GetName() string {
	return fmt.Sprintf("projects/%s", p.ResourceID)
}

// FindProjectMessage is the message for finding projects.
type FindProjectMessage struct {
	// We should only set either UID or ResourceID.
	// Deprecate UID later once we fully migrate to ResourceID.
	UID         *int
	ResourceID  *string
	ShowDeleted bool
}

// UpdateProjectMessage is the message for updating a project.
type UpdateProjectMessage struct {
	UpdaterID  int
	ResourceID string

	Title                      *string
	Key                        *string
	TenantMode                 *api.ProjectTenantMode
	DataClassificationConfigID *string
	Setting                    *storepb.Project
	Delete                     *bool
}

// GetProjectV2 gets project by resource ID.
func (s *Store) GetProjectV2(ctx context.Context, find *FindProjectMessage) (*ProjectMessage, error) {
	if find.ResourceID != nil {
		if v, ok := s.projectCache.Get(*find.ResourceID); ok {
			return v, nil
		}
	}
	if find.UID != nil {
		if v, ok := s.projectIDCache.Get(*find.UID); ok {
			return v, nil
		}
	}

	// We will always return the resource regardless of its deleted state.
	find.ShowDeleted = true

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	projects, err := s.listProjectImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}
	if len(projects) == 0 {
		return nil, nil
	}
	if len(projects) > 1 {
		return nil, &common.Error{Code: common.Conflict, Err: errors.Errorf("found %d projects with filter %+v, expect 1", len(projects), find)}
	}
	project := projects[0]

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.storeProjectCache(project)
	return projects[0], nil
}

// ListProjectV2 lists all projects.
func (s *Store) ListProjectV2(ctx context.Context, find *FindProjectMessage) ([]*ProjectMessage, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	projects, err := s.listProjectImplV2(ctx, tx, find)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	for _, project := range projects {
		s.storeProjectCache(project)
	}
	return projects, nil
}

// CreateProjectV2 creates a project.
func (s *Store) CreateProjectV2(ctx context.Context, create *ProjectMessage, creatorID int) (*ProjectMessage, error) {
	user, err := s.GetUserByID(ctx, creatorID)
	if err != nil {
		return nil, err
	}
	if create.Setting == nil {
		create.Setting = &storepb.Project{}
	}
	payload, err := protojson.Marshal(create.Setting)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	project := &ProjectMessage{
		ResourceID:                 create.ResourceID,
		Title:                      create.Title,
		Key:                        create.Key,
		TenantMode:                 create.TenantMode,
		DataClassificationConfigID: create.DataClassificationConfigID,
		Setting:                    create.Setting,
	}
	if err := tx.QueryRowContext(ctx, `
			INSERT INTO project (
				creator_id,
				updater_id,
				resource_id,
				name,
				key,
				tenant_mode,
				data_classification_config_id,
				setting
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id
		`,
		creatorID,
		creatorID,
		create.ResourceID,
		create.Title,
		create.Key,
		create.TenantMode,
		create.DataClassificationConfigID,
		payload,
	).Scan(
		&project.UID,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery("failed to create project")
		}
		return nil, err
	}

	policy := &IAMPolicyMessage{
		Bindings: []*PolicyBinding{
			{
				Role: api.ProjectOwner,
				Members: []*UserMessage{
					user,
				},
			},
		},
	}
	if err := s.setProjectIAMPolicyImpl(ctx, tx, policy, creatorID, project.UID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.storeProjectCache(project)
	return project, nil
}

// UpdateProjectV2 updates a project.
func (s *Store) UpdateProjectV2(ctx context.Context, patch *UpdateProjectMessage) (*ProjectMessage, error) {
	s.removeProjectCache(patch.ResourceID)

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := updateProjectImplV2(ctx, tx, patch); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetProjectV2(ctx, &FindProjectMessage{ResourceID: &patch.ResourceID})
}

func updateProjectImplV2(ctx context.Context, tx *Tx, patch *UpdateProjectMessage) error {
	set, args := []string{"updater_id = $1"}, []any{fmt.Sprintf("%d", patch.UpdaterID)}
	if v := patch.Title; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Key; v != nil {
		set, args = append(set, fmt.Sprintf("key = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Delete; v != nil {
		rowStatus := api.Normal
		if *patch.Delete {
			rowStatus = api.Archived
		}
		set, args = append(set, fmt.Sprintf(`"row_status" = $%d`, len(args)+1)), append(args, rowStatus)
	}
	if v := patch.TenantMode; v != nil {
		set, args = append(set, fmt.Sprintf("tenant_mode = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.DataClassificationConfigID; v != nil {
		set, args = append(set, fmt.Sprintf("data_classification_config_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := patch.Setting; v != nil {
		payload, err := protojson.Marshal(patch.Setting)
		if err != nil {
			return err
		}
		set, args = append(set, fmt.Sprintf("setting = $%d", len(args)+1)), append(args, payload)
	}

	args = append(args, patch.ResourceID)
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
		UPDATE project
		SET `+strings.Join(set, ", ")+`
		WHERE resource_id = $%d`, len(args)),
		args...,
	); err != nil {
		return err
	}
	return nil
}

func (s *Store) listProjectImplV2(ctx context.Context, tx *Tx, find *FindProjectMessage) ([]*ProjectMessage, error) {
	where, args := []string{"TRUE"}, []any{}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("resource_id = $%d", len(args)+1)), append(args, *v)
	}
	if v := find.UID; v != nil {
		where, args = append(where, fmt.Sprintf("id = $%d", len(args)+1)), append(args, *v)
	}
	if !find.ShowDeleted {
		where, args = append(where, fmt.Sprintf("row_status = $%d", len(args)+1)), append(args, api.Normal)
	}

	var projectMessages []*ProjectMessage
	rows, err := tx.QueryContext(ctx, `
		SELECT
			id,
			resource_id,
			name,
			key,
			tenant_mode,
			data_classification_config_id,
			(SELECT COUNT(1) FROM repository WHERE project.id = repository.project_id) AS connectors,
			setting,
			row_status
		FROM project
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY project.id`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var projectMessage ProjectMessage
		var payload []byte
		var rowStatus string
		if err := rows.Scan(
			&projectMessage.UID,
			&projectMessage.ResourceID,
			&projectMessage.Title,
			&projectMessage.Key,
			&projectMessage.TenantMode,
			&projectMessage.DataClassificationConfigID,
			&projectMessage.VCSConnectorsCount,
			&payload,
			&rowStatus,
		); err != nil {
			return nil, err
		}
		setting := &storepb.Project{}
		if err := protojsonUnmarshaler.Unmarshal(payload, setting); err != nil {
			return nil, err
		}
		projectMessage.Setting = setting
		projectMessage.Deleted = convertRowStatusToDeleted(rowStatus)
		projectMessages = append(projectMessages, &projectMessage)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, project := range projectMessages {
		projectWebhooks, err := s.findProjectWebhookImplV2(ctx, tx, &FindProjectWebhookMessage{ProjectID: &project.UID})
		if err != nil {
			return nil, err
		}
		project.Webhooks = projectWebhooks
	}

	return projectMessages, nil
}

func (s *Store) storeProjectCache(project *ProjectMessage) {
	s.projectCache.Add(project.ResourceID, project)
	s.projectIDCache.Add(project.UID, project)
}

func (s *Store) removeProjectCache(resourceID string) {
	if project, ok := s.projectCache.Get(resourceID); ok {
		s.projectIDCache.Remove(project.UID)
	}
	s.projectCache.Remove(resourceID)
}
