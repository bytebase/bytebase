package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"

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

// GetSheetTruncated gets a sheet by SHA256 hash with truncated statement (max 2MB).
// Statement field will be truncated to MaxSheetSize (2MB).
// Results are cached by SHA256 hex string.
func (s *Store) GetSheetTruncated(ctx context.Context, sha256Hex string) (*SheetMessage, error) {
	sheet, err := s.getSheet(ctx, sha256Hex, false)
	if err != nil {
		return nil, err
	}
	if sheet == nil {
		return nil, nil
	}
	return sheet, nil
}

// GetSheetFull gets a sheet by SHA256 hash with the complete statement.
// Statement field contains the complete content regardless of size.
// Results are cached by SHA256 hex string.
func (s *Store) GetSheetFull(ctx context.Context, sha256Hex string) (*SheetMessage, error) {
	if v, ok := s.sheetFullCache.Get(sha256Hex); ok && s.enableCache {
		return v, nil
	}

	sheet, err := s.getSheet(ctx, sha256Hex, true)
	if err != nil {
		return nil, err
	}
	if sheet == nil {
		return nil, nil
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
		return nil, nil
	}

	return sheet, nil
}

// CreateSheets creates sheet blobs using content-addressed storage.
// Each sheet is identified by the SHA256 hash of its statement.
// Duplicate statements share the same blob (ON CONFLICT DO NOTHING).
func (s *Store) CreateSheets(ctx context.Context, creates ...*SheetMessage) ([]*SheetMessage, error) {
	var statements []string
	var sha256s [][]byte

	for _, c := range creates {
		statements = append(statements, c.Statement)
		h := sha256.Sum256([]byte(c.Statement))
		c.Sha256 = hex.EncodeToString(h[:])
		sha256s = append(sha256s, h[:])
		c.Size = int64(len(c.Statement))
	}

	q := qb.Q().Space(`
		INSERT INTO sheet_blob (
			sha256,
			content
		) SELECT
		 	unnest(CAST(? AS BYTEA[])),
			unnest(CAST(? AS TEXT[]))
		ON CONFLICT DO NOTHING
	`, sha256s, statements)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	if _, err := s.GetDB().ExecContext(ctx, query, args...); err != nil {
		return nil, errors.Wrapf(err, "failed to exec")
	}

	return creates, nil
}

// HasSheets checks if all sheets exist by SHA256 hashes.
func (s *Store) HasSheets(ctx context.Context, sha256Hexes ...string) (bool, error) {
	if len(sha256Hexes) == 0 {
		return true, nil
	}

	// Remove duplicates
	sha256Hexes = common.Uniq(sha256Hexes)

	q := qb.Q().Space(`
		SELECT COUNT(*)
		FROM sheet_blob
		WHERE sha256 IN (SELECT decode(unnest(CAST(? AS TEXT[])), 'hex'))`,
		sha256Hexes)

	query, args, err := q.ToSQL()
	if err != nil {
		return false, errors.Wrapf(err, "failed to build sql")
	}

	var count int
	if err := s.GetDB().QueryRowContext(ctx, query, args...).Scan(&count); err != nil {
		return false, err
	}

	return count == len(sha256Hexes), nil
}
