package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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

	CreatorID int
	UpdaterID int

	Title     string
	Statement string
	Payload   *storepb.SheetPayload

	// Sha256 is the Sha256 hash of the statement.
	Sha256 []byte

	// Output only fields
	UID         int
	Size        int64
	CreatedTime time.Time
	UpdatedTime time.Time

	// Internal fields
	createdTs int64
	updatedTs int64
}

func (s *SheetMessage) GetSha256Hex() string {
	return hex.EncodeToString(s.Sha256)
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
	if v, ok := s.sheetStatementCache.Get(id); ok {
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
	s.sheetStatementCache.Add(id, statement)
	return statement, nil
}

// GetSheet gets a sheet.
func (s *Store) GetSheet(ctx context.Context, find *FindSheetMessage) (*SheetMessage, error) {
	shouldCache := !find.LoadFull && find.UID != nil
	if shouldCache {
		if v, ok := s.sheetCache.Get(*find.UID); ok {
			return v, nil
		}
	}

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

	if shouldCache {
		s.sheetCache.Add(sheet.UID, sheet)
	}

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
	statementField := fmt.Sprintf("LEFT(sheet_blob.content, %d)", common.MaxSheetSize)
	if find.LoadFull {
		statementField = "sheet_blob.content"
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
			sheet.name,
			%s,
			sheet.sha256,
			sheet.payload,
			OCTET_LENGTH(sheet_blob.content)
		FROM sheet
		LEFT JOIN sheet_blob ON sheet.sha256 = sheet_blob.sha256
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
			&sheet.Title,
			&sheet.Statement,
			&sheet.Sha256,
			&payload,
			&sheet.Size,
		); err != nil {
			return nil, err
		}
		sheetPayload := &storepb.SheetPayload{}
		if err := common.ProtojsonUnmarshaler.Unmarshal(payload, sheetPayload); err != nil {
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
// You should not use this function directly to create sheets.
// Use CreateSheet in component/sheet instead.
func (s *Store) CreateSheet(ctx context.Context, create *SheetMessage) (*SheetMessage, error) {
	if create.Payload == nil {
		create.Payload = &storepb.SheetPayload{}
	}
	payload, err := protojson.Marshal(create.Payload)
	if err != nil {
		return nil, err
	}

	h := sha256.Sum256([]byte(create.Statement))
	create.Sha256 = h[:]

	if err := s.BatchCreateSheetBlob(ctx, [][]byte{create.Sha256}, []string{create.Statement}); err != nil {
		return nil, errors.Wrapf(err, "failed to create sheet blobs")
	}

	query := `
		INSERT INTO sheet (
			creator_id,
			updater_id,
			project_id,
			name,
			sha256,
			payload
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_ts, updated_ts
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
		create.Title,
		create.Sha256,
		payload,
	).Scan(
		&create.UID,
		&create.createdTs,
		&create.updatedTs,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, common.FormatDBErrorEmptyRowWithQuery(query)
		}
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit transaction")
	}

	create.Size = int64(len(create.Statement))
	create.CreatedTime = time.Unix(create.createdTs, 0)
	create.UpdatedTime = time.Unix(create.updatedTs, 0)

	return create, nil
}

// BatchCreateSheet creates a new sheet.
// You should not use this function directly to create sheets.
// Use BatchCreateSheet in component/sheet instead.
func (s *Store) BatchCreateSheet(ctx context.Context, projectUID int, creates []*SheetMessage, creatorUID int) ([]*SheetMessage, error) {
	var names []string
	var statements []string
	var sha256s [][]byte
	var payloads [][]byte

	for _, c := range creates {
		names = append(names, c.Title)
		statements = append(statements, c.Statement)
		h := sha256.Sum256([]byte(c.Statement))
		c.Sha256 = h[:]
		sha256s = append(sha256s, c.Sha256)
		if c.Payload == nil {
			c.Payload = &storepb.SheetPayload{}
		}
		payload, err := protojson.Marshal(c.Payload)
		if err != nil {
			return nil, err
		}
		payloads = append(payloads, payload)
	}

	if err := s.BatchCreateSheetBlob(ctx, sha256s, statements); err != nil {
		return nil, errors.Wrapf(err, "failed to create sheet blobs")
	}

	query := `
		INSERT INTO sheet (
			creator_id,
			updater_id,
			project_id,
			name,
			sha256,
			payload
		) SELECT
			$1,
			$2,
			$3,
			unnest(CAST($4 AS TEXT[])),
			unnest(CAST($5 AS BYTEA[])),
			unnest(CAST($6 AS JSONB[]))
		RETURNING id, created_ts, updated_ts
	`

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, creatorUID, creatorUID, projectUID, names, sha256s, payloads)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query")
	}
	defer rows.Close()

	for i := 0; rows.Next(); i++ {
		creates[i].UpdaterID = creatorUID
		creates[i].CreatorID = creatorUID

		if err := rows.Scan(
			&creates[i].UID,
			&creates[i].createdTs,
			&creates[i].updatedTs,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan")
		}

		creates[i].Size = int64(len(creates[i].Statement))
		creates[i].CreatedTime = time.Unix(creates[i].createdTs, 0)
		creates[i].UpdatedTime = time.Unix(creates[i].updatedTs, 0)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "rows err")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit tx")
	}

	return creates, nil
}

func (s *Store) BatchCreateSheetBlob(ctx context.Context, sha256s [][]byte, contents []string) error {
	query := `
		INSERT INTO sheet_blob (
			sha256,
			content
		) SELECT
		 	unnest(CAST($1 AS BYTEA[])),
			unnest(CAST($2 AS TEXT[]))
		ON CONFLICT DO NOTHING
	`

	if _, err := s.db.db.ExecContext(ctx, query, sha256s, contents); err != nil {
		return errors.Wrapf(err, "failed to exec")
	}

	return nil
}

// PatchSheet updates a sheet.
func (s *Store) PatchSheet(ctx context.Context, patch *PatchSheetMessage) (*SheetMessage, error) {
	if patch.Statement == nil {
		return s.GetSheet(ctx, &FindSheetMessage{UID: &patch.UID})
	}

	h := sha256.Sum256([]byte(*patch.Statement))

	if err := s.BatchCreateSheetBlob(ctx, [][]byte{h[:]}, []string{*patch.Statement}); err != nil {
		return nil, errors.Wrapf(err, "failed to create sheet blobs")
	}

	var uid int
	if err := s.db.db.QueryRowContext(ctx, `
		UPDATE sheet
		SET 
			updater_id = $1,
			updated_ts = $2,
			sha256 = $3
		WHERE id = $4
		RETURNING id
	`,
		patch.UpdaterID,
		time.Now().Unix(),
		h[:],
		patch.UID,
	).Scan(
		&uid,
	); err != nil {
		if err == sql.ErrNoRows {
			return nil, &common.Error{Code: common.NotFound, Err: errors.Errorf("sheet ID not found: %d", patch.UID)}
		}
		return nil, errors.Wrapf(err, "failed to update sheet")
	}

	s.sheetStatementCache.Add(patch.UID, *patch.Statement)
	s.sheetCache.Remove(patch.UID)

	return s.GetSheet(ctx, &FindSheetMessage{UID: &patch.UID})
}
