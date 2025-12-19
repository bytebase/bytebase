package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/qb"
)

// SheetMessage is the message for a sheet.
type SheetMessage struct {
	// SHA256 hash of the statement (hex-encoded)
	Sha256 string
	// SQL statement content
	Statement string
	// Size of the statement in bytes
	Size int64
}

// GetSheetMetadata gets a sheet with truncated statement (max 2MB).
// Use this when you need to check sheet.Size or other metadata before processing.
// Statement field will be truncated to MaxSheetSize (2MB).
func (s *Store) GetSheetMetadata(ctx context.Context, sha256Hex string) (*SheetMessage, error) {
	if v, ok := s.sheetMetadataCache.Get(sha256Hex); ok && s.enableCache {
		return v, nil
	}

	sheet, err := s.getSheet(ctx, sha256Hex, false)
	if err != nil {
		return nil, err
	}

	s.sheetMetadataCache.Add(sha256Hex, sheet)
	return sheet, nil
}

// GetSheetFull gets a sheet with the complete statement.
// Use this when you need the full statement for execution or processing.
// Statement field contains the complete content regardless of size.
func (s *Store) GetSheetFull(ctx context.Context, sha256Hex string) (*SheetMessage, error) {
	if v, ok := s.sheetFullCache.Get(sha256Hex); ok && s.enableCache {
		return v, nil
	}

	sheet, err := s.getSheet(ctx, sha256Hex, true)
	if err != nil {
		return nil, err
	}

	s.sheetFullCache.Add(sha256Hex, sheet)
	return sheet, nil
}

// getSheet is the internal helper for fetching a single sheet by SHA256.
func (s *Store) getSheet(ctx context.Context, sha256Hex string, loadFull bool) (*SheetMessage, error) {
	statementField := fmt.Sprintf("LEFT(content, %d)", common.MaxSheetSize)
	if loadFull {
		statementField = "content"
	}

	q := qb.Q().Space(fmt.Sprintf(`
		SELECT
			%s,
			OCTET_LENGTH(content)
		FROM sheet_blob
		WHERE sha256 = decode(?, 'hex')`, statementField), sha256Hex)

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
		sheet = &SheetMessage{
			Sha256: sha256Hex,
		}
		if err := rows.Scan(
			&sheet.Statement,
			&sheet.Size,
		); err != nil {
			return nil, err
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	if sheet == nil {
		return nil, errors.Errorf("sheet not found with sha256 %s", sha256Hex)
	}

	return sheet, nil
}

// CreateSheets creates new sheets.
// You should not use this function directly to create sheets.
// Use CreateSheets in component/sheet instead.
func (s *Store) CreateSheets(ctx context.Context, projectID string, _ string, creates ...*SheetMessage) ([]*SheetMessage, error) {
	var names []string
	var statements []string
	var sha256s [][]byte

	for _, c := range creates {
		c.ProjectID = projectID
		c.Creator = SystemBotUser.Email
		names = append(names, "")
		statements = append(statements, c.Statement)
		h := sha256.Sum256([]byte(c.Statement))
		c.Sha256 = h[:]
		sha256s = append(sha256s, c.Sha256)
	}

	if err := s.batchCreateSheetBlob(ctx, sha256s, statements); err != nil {
		return nil, errors.Wrapf(err, "failed to create sheet blobs")
	}

	q := qb.Q().Space(`
		INSERT INTO sheet (
			creator,
			project,
			name,
			sha256
		) SELECT
			?,
			?,
			unnest(CAST(? AS TEXT[])),
			unnest(CAST(? AS BYTEA[]))
		RETURNING id, created_at
	`, SystemBotUser.Email, projectID, names, sha256s)

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
