package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

// BranchMessage is the message for a branch.
type BranchMessage struct {
	ProjectID  string
	ResourceID string

	Engine storepb.Engine
	Config *storepb.BranchConfig
	Base   *storepb.BranchSnapshot
	Head   *storepb.BranchSnapshot

	// Output only fields
	UID         int
	CreatorID   int
	UpdaterID   int
	CreatedTime time.Time
	UpdatedTime time.Time
}

// FindBranchMessage is the API message for finding branches.
type FindBranchMessage struct {
	ProjectID  *string
	ResourceID *string
	// TODO(d): handle LoadFull.
	LoadFull bool
}

// UpdateBranchMessage is the message to update a branch.
type UpdateBranchMessage struct {
	ProjectID        string
	ResourceID       string
	UpdaterID        int
	Config           *storepb.BranchConfig
	Base             *storepb.BranchSnapshot
	Head             *storepb.BranchSnapshot
	UpdateResourceID *string
}

// GetBranch gets a branch.
func (s *Store) GetBranch(ctx context.Context, find *FindBranchMessage) (*BranchMessage, error) {
	branches, err := s.ListBranches(ctx, find)
	if err != nil {
		return nil, err
	}
	if len(branches) == 0 {
		return nil, nil
	}
	if len(branches) > 1 {
		return nil, errors.Errorf("expected 1 branch, got %d", len(branches))
	}
	return branches[0], nil
}

// ListBranches returns a list of branches.
func (s *Store) ListBranches(ctx context.Context, find *FindBranchMessage) ([]*BranchMessage, error) {
	where, args := []string{"TRUE"}, []any{}

	if v := find.ProjectID; v != nil {
		project, err := s.GetProjectV2(ctx, &FindProjectMessage{ResourceID: v})
		if err != nil {
			return nil, err
		}
		where, args = append(where, fmt.Sprintf("branch.project_id = $%d", len(args)+1)), append(args, project.UID)
	}
	if v := find.ResourceID; v != nil {
		where, args = append(where, fmt.Sprintf("branch.name = $%d", len(args)+1)), append(args, *v)
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT
			branch.id,
			branch.creator_id,
			branch.created_ts,
			branch.updater_id,
			branch.updated_ts,
			project.resource_id AS project_id,
			branch.name,
			branch.engine,
			branch.base,
			branch.head,
			branch.config
		FROM branch
		LEFT JOIN project ON branch.project_id = project.id
		WHERE %s`, strings.Join(where, " AND ")),
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var branches []*BranchMessage
	for rows.Next() {
		var branch BranchMessage
		var createdTs, updatedTs int64
		var base, head, config []byte
		var engine string
		if err := rows.Scan(
			&branch.UID,
			&branch.CreatorID,
			&createdTs,
			&branch.UpdaterID,
			&updatedTs,
			&branch.ProjectID,
			&branch.ResourceID,
			&engine,
			&base,
			&head,
			&config,
		); err != nil {
			return nil, err
		}
		engineTypeValue, ok := storepb.Engine_value[engine]
		if !ok {
			return nil, errors.Errorf("invalid engine %s", engine)
		}
		branch.Engine = storepb.Engine(engineTypeValue)

		branchBase := &storepb.BranchSnapshot{}
		if err := protojsonUnmarshaler.Unmarshal(base, branchBase); err != nil {
			return nil, err
		}
		branch.Base = branchBase
		branchHead := &storepb.BranchSnapshot{}
		if err := protojsonUnmarshaler.Unmarshal(head, branchHead); err != nil {
			return nil, err
		}
		branch.Head = branchHead
		branchConfig := &storepb.BranchConfig{}
		if err := protojsonUnmarshaler.Unmarshal(config, branchConfig); err != nil {
			return nil, err
		}
		branch.Config = branchConfig
		branch.CreatedTime = time.Unix(createdTs, 0)
		branch.UpdatedTime = time.Unix(updatedTs, 0)

		branches = append(branches, &branch)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return branches, nil
}

// CreateBranch creates a branch.
func (s *Store) CreateBranch(ctx context.Context, create *BranchMessage) (*BranchMessage, error) {
	project, err := s.GetProjectV2(ctx, &FindProjectMessage{ResourceID: &create.ProjectID})
	if err != nil {
		return nil, err
	}
	if create.Base == nil {
		create.Base = &storepb.BranchSnapshot{}
	}
	if create.Head == nil {
		create.Head = &storepb.BranchSnapshot{}
	}
	if create.Config == nil {
		create.Config = &storepb.BranchConfig{}
	}
	create.UpdaterID = create.CreatorID
	base, err := protojson.Marshal(create.Base)
	if err != nil {
		return nil, err
	}
	head, err := protojson.Marshal(create.Head)
	if err != nil {
		return nil, err
	}
	config, err := protojson.Marshal(create.Config)
	if err != nil {
		return nil, err
	}

	query := `
		INSERT INTO branch (
			creator_id,
			updater_id,
			project_id,
			name,
			engine,
			base,
			head,
			config
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_ts, updated_ts;
	`

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var createdTs, updatedTs int64
	if err := tx.QueryRowContext(ctx, query,
		create.CreatorID,
		create.CreatorID,
		project.UID,
		create.ResourceID,
		create.Engine.String(),
		base,
		head,
		config,
	).Scan(
		&create.UID,
		&createdTs,
		&updatedTs,
	); err != nil {
		return nil, err
	}
	create.CreatedTime = time.Unix(createdTs, 0)
	create.UpdatedTime = time.Unix(updatedTs, 0)
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return create, nil
}

// UpdateBranch updates a branch.
func (s *Store) UpdateBranch(ctx context.Context, update *UpdateBranchMessage) error {
	project, err := s.GetProjectV2(ctx, &FindProjectMessage{ResourceID: &update.ProjectID})
	if err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to begin transaction")
	}

	set, args := []string{"updater_id = $1"}, []any{update.UpdaterID}
	if v := update.Base; v != nil {
		base, err := protojson.Marshal(v)
		if err != nil {
			return err
		}
		set, args = append(set, fmt.Sprintf("base = $%d", len(args)+1)), append(args, base)
	}
	if v := update.Head; v != nil {
		head, err := protojson.Marshal(v)
		if err != nil {
			return err
		}
		set, args = append(set, fmt.Sprintf("head = $%d", len(args)+1)), append(args, head)
	}
	if v := update.Config; v != nil {
		config, err := protojson.Marshal(v)
		if err != nil {
			return err
		}
		set, args = append(set, fmt.Sprintf("config = $%d", len(args)+1)), append(args, config)
	}
	if v := update.UpdateResourceID; v != nil {
		set, args = append(set, fmt.Sprintf("name = $%d", len(args)+1)), append(args, *v)
	}
	args = append(args, project.UID, update.ResourceID)

	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
		UPDATE branch
		SET `+strings.Join(set, ", ")+`
		WHERE branch.project_id = $%d AND branch.name = $%d`, len(set)+1, len(set)+2), args...); err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteBranch deletes a branch.
func (s *Store) DeleteBranch(ctx context.Context, projectID, resourceID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM branch
		USING project
		WHERE branch.project_id = project.id AND project.resource_id = $1 AND branch.name = $2;`,
		projectID, resourceID); err != nil {
		return err
	}

	return tx.Commit()
}
