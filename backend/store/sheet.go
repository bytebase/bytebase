package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// SheetMessage is the message for a sheet.
type SheetMessage struct {
	ProjectID string

	Creator string

	Title     string
	Statement string
	Payload   *storepb.SheetPayload

	// Sha256 is the Sha256 hash of the statement.
	Sha256 []byte

	// Output only fields
	UID       int
	Size      int64
	CreatedAt time.Time
}

func (s *SheetMessage) GetSha256Hex() string {
	return hex.EncodeToString(s.Sha256)
}

// FindSheetMessage is the API message for finding sheets.
type FindSheetMessage struct {
	ProjectID *string
	UID       *int
	// LoadFull is used if we want to load the full sheet.
	LoadFull bool
}

// GetSheetMetadata gets a sheet with truncated statement (max 2MB).
// Use this when you need to check sheet.Size or other metadata before processing.
// Statement field will be truncated to MaxSheetSize (2MB).
func (s *Store) GetSheetMetadata(ctx context.Context, id int) (*SheetMessage, error) {
	if v, ok := s.sheetMetadataCache.Get(id); ok && s.enableCache {
		return v, nil
	}

	sheet, err := s.getSheet(ctx, id, false)
	if err != nil {
		return nil, err
	}

	s.sheetMetadataCache.Add(id, sheet)
	return sheet, nil
}

// GetSheetFull gets a sheet with the complete statement.
// Use this when you need the full statement for execution or processing.
// Statement field contains the complete content regardless of size.
func (s *Store) GetSheetFull(ctx context.Context, id int) (*SheetMessage, error) {
	if v, ok := s.sheetFullCache.Get(id); ok && s.enableCache {
		return v, nil
	}

	sheet, err := s.getSheet(ctx, id, true)
	if err != nil {
		return nil, err
	}

	s.sheetFullCache.Add(id, sheet)
	return sheet, nil
}

// GetSheetStatementByID gets the statement of a sheet by ID.
func (s *Store) GetSheetStatementByID(ctx context.Context, id int) (string, error) {
	if v, ok := s.sheetStatementCache.Get(id); ok && s.enableCache {
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
		if v, ok := s.sheetCache.Get(*find.UID); ok && s.enableCache {
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

// getSheet is the internal helper for fetching a single sheet by ID.
func (s *Store) getSheet(ctx context.Context, id int, loadFull bool) (*SheetMessage, error) {
	statementField := fmt.Sprintf("LEFT(sheet_blob.content, %d)", common.MaxSheetSize)
	if loadFull {
		statementField = "sheet_blob.content"
	}

	q := qb.Q().Space(fmt.Sprintf(`
		SELECT
			sheet.id,
			sheet.creator,
			sheet.created_at,
			sheet.project,
			sheet.name,
			%s,
			sheet.sha256,
			sheet.payload,
			OCTET_LENGTH(sheet_blob.content)
		FROM sheet
		LEFT JOIN sheet_blob ON sheet.sha256 = sheet_blob.sha256
		WHERE sheet.id = ?`, statementField), id)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sheet *SheetMessage
	if rows.Next() {
		sheet = &SheetMessage{}
		var payload []byte
		if err := rows.Scan(
			&sheet.UID,
			&sheet.Creator,
			&sheet.CreatedAt,
			&sheet.ProjectID,
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
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if sheet == nil {
		return nil, errors.Errorf("sheet not found with id %d", id)
	}

	return sheet, nil
}

// CreateSheets creates new sheets.
// You should not use this function directly to create sheets.
// Use CreateSheets in component/sheet instead.
func (s *Store) CreateSheets(ctx context.Context, projectID string, creator string, creates ...*SheetMessage) ([]*SheetMessage, error) {
	var names []string
	var statements []string
	var sha256s [][]byte
	var payloads [][]byte

	for _, c := range creates {
		c.ProjectID = projectID
		c.Creator = creator
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

	if err := s.batchCreateSheetBlob(ctx, sha256s, statements); err != nil {
		return nil, errors.Wrapf(err, "failed to create sheet blobs")
	}

	q := qb.Q().Space(`
		INSERT INTO sheet (
			creator,
			project,
			name,
			sha256,
			payload
		) SELECT
			?,
			?,
			unnest(CAST(? AS TEXT[])),
			unnest(CAST(? AS BYTEA[])),
			unnest(CAST(? AS JSONB[]))
		RETURNING id, created_at
	`, creator, projectID, names, sha256s, payloads)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to begin tx")
	}
	defer tx.Rollback()

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query")
	}
	defer rows.Close()

	for i := 0; rows.Next(); i++ {
		if err := rows.Scan(
			&creates[i].UID,
			&creates[i].CreatedAt,
		); err != nil {
			return nil, errors.Wrapf(err, "failed to scan")
		}

		creates[i].Size = int64(len(creates[i].Statement))
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrapf(err, "rows err")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrapf(err, "failed to commit tx")
	}

	return creates, nil
}

func (s *Store) batchCreateSheetBlob(ctx context.Context, sha256s [][]byte, contents []string) error {
	q := qb.Q().Space(`
		INSERT INTO sheet_blob (
			sha256,
			content
		) SELECT
		 	unnest(CAST(? AS BYTEA[])),
			unnest(CAST(? AS TEXT[]))
		ON CONFLICT DO NOTHING
	`, sha256s, contents)

	query, args, err := q.ToSQL()
	if err != nil {
		return errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrapf(err, "failed to exec")
	}

	return nil
}
