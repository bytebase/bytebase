package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"slices"
	"strings"

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

// validSheetSha256Hexes filters to well-formed 32-byte hex hashes and
// canonicalizes them to lowercase — the form encode(sha256, 'hex') returns —
// so result maps and the content cache key consistently no matter the case a
// caller or a legacy stored payload used. Stored payloads are not trusted to
// carry valid hex; a malformed value can never match a blob, so callers treat
// it as absent rather than letting decode() turn it into a SQL error.
func validSheetSha256Hexes(sha256Hexes []string) []string {
	valid := make([]string, 0, len(sha256Hexes))
	for _, sha256Hex := range sha256Hexes {
		if len(sha256Hex) != 64 {
			continue
		}
		if _, err := hex.DecodeString(sha256Hex); err != nil {
			continue
		}
		valid = append(valid, strings.ToLower(sha256Hex))
	}
	return valid
}

// GetSheetFull gets a sheet by SHA256 hash with the complete statement, with
// no project scope. It serves the runners and components, which execute work
// that was authorized when its plan or release was created; user-facing reads
// go through the scoped accessors below.
func (s *Store) GetSheetFull(ctx context.Context, sha256Hex string) (*SheetMessage, error) {
	sheets, err := s.getSheetsFull(ctx, sha256Hex)
	if err != nil {
		return nil, err
	}
	return sheets[strings.ToLower(sha256Hex)], nil
}

// GetSheetsForProject is the sheet gate: a project may access a sheet only
// when it holds a sheet_blob_ref for its hash. It filters the requested
// hashes through the project's refs and fetches content only for those that
// survive; a hash absent from the result does not exist as far as the caller
// is concerned (NotFound rather than PermissionDenied, so the response does
// not confirm the hash exists in some other project).
//
// The ref check runs before any content read on purpose: the full-content
// cache is keyed by hash alone and a cache hit issues no query, so a scope
// predicate placed with the content fetch would be skipped. Do not fuse the
// check and the fetch into one query, and do not reorder them. Both steps are
// batched: one ref query plus one content query for the cache misses,
// whatever the input size.
func (s *Store) GetSheetsForProject(ctx context.Context, projectID string, sha256Hexes []string, raw bool) (map[string]*SheetMessage, error) {
	allowed, err := s.filterSheetsForProject(ctx, projectID, common.Uniq(sha256Hexes))
	if err != nil {
		return nil, err
	}
	granted := slices.Collect(maps.Keys(allowed))
	if len(granted) == 0 {
		return map[string]*SheetMessage{}, nil
	}
	if raw {
		return s.getSheetsFull(ctx, granted...)
	}
	return s.getSheets(ctx, granted, false)
}

// GetSheetForProject is the single-sheet form of GetSheetsForProject. A nil
// result means the project may not read the hash (or no such blob exists).
func (s *Store) GetSheetForProject(ctx context.Context, projectID, sha256Hex string, raw bool) (*SheetMessage, error) {
	sheets, err := s.GetSheetsForProject(ctx, projectID, []string{sha256Hex}, raw)
	if err != nil {
		return nil, err
	}
	return sheets[strings.ToLower(sha256Hex)], nil
}

// MissingSheetsForProject returns the hashes, in input order, for which the
// project holds no ref — a ref implies its blob through the foreign key, and
// a malformed hash matches nothing, so it comes back missing. An empty result
// means every hash is readable. Run when a plan or revision is created, never
// per use: sheets are never deleted, and refs are deleted only by project
// purge, which deletes the referencing rows too.
func (s *Store) MissingSheetsForProject(ctx context.Context, projectID string, sha256Hexes ...string) ([]string, error) {
	sha256Hexes = common.Uniq(sha256Hexes)
	allowed, err := s.filterSheetsForProject(ctx, projectID, sha256Hexes)
	if err != nil {
		return nil, err
	}
	// Look up by the canonical lowercase form, but do NOT pre-filter the
	// inputs through validSheetSha256Hexes here: a malformed hash must be
	// reported missing, not silently dropped from the result.
	var missing []string
	for _, sha256Hex := range sha256Hexes {
		if !allowed[strings.ToLower(sha256Hex)] {
			missing = append(missing, sha256Hex)
		}
	}
	return missing, nil
}

// getSheetsFull gets sheets by SHA256 hashes with complete statements in one
// query for all cache misses. The result is keyed by hex hash; an absent key
// means no such blob. Content is a pure function of the hash, so the cache is
// keyed by hex hash alone; scoping runs before content retrieval (a cache hit
// issues no query).
func (s *Store) getSheetsFull(ctx context.Context, sha256Hexes ...string) (map[string]*SheetMessage, error) {
	// Canonicalize before the cache loop so cache keys, result keys, and the
	// keys encode() produces all agree, and mixed-case duplicates collapse.
	sha256Hexes = common.Uniq(validSheetSha256Hexes(sha256Hexes))
	result := make(map[string]*SheetMessage, len(sha256Hexes))
	misses := sha256Hexes
	if s.enableCache {
		misses = nil
		for _, sha256Hex := range sha256Hexes {
			if v, ok := s.sheetFullCache.Get(sha256Hex); ok {
				result[sha256Hex] = v
			} else {
				misses = append(misses, sha256Hex)
			}
		}
	}
	if len(misses) == 0 {
		return result, nil
	}

	fetched, err := s.getSheets(ctx, misses, true)
	if err != nil {
		return nil, err
	}
	for sha256Hex, sheet := range fetched {
		result[sha256Hex] = sheet
		if s.enableCache {
			s.sheetFullCache.Add(sha256Hex, sheet)
		}
	}
	return result, nil
}

// getSheets is the internal helper for fetching sheets by SHA256 in one query.
func (s *Store) getSheets(ctx context.Context, sha256Hexes []string, loadFull bool) (map[string]*SheetMessage, error) {
	result := map[string]*SheetMessage{}
	sha256Hexes = validSheetSha256Hexes(sha256Hexes)
	if len(sha256Hexes) == 0 {
		return result, nil
	}

	statementField := fmt.Sprintf("LEFT(content, %d)", common.MaxSheetSize)
	if loadFull {
		statementField = "content"
	}

	q := qb.Q().Space(fmt.Sprintf(`
		SELECT
			encode(sha256, 'hex'),
			%s,
			OCTET_LENGTH(content)
		FROM sheet_blob
		WHERE sha256 IN (SELECT decode(unnest(CAST(? AS TEXT[])), 'hex'))`, statementField), sha256Hexes)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		sheet := &SheetMessage{}
		if err := rows.Scan(
			&sheet.Sha256,
			&sheet.Statement,
			&sheet.Size,
		); err != nil {
			return nil, err
		}
		result[sheet.Sha256] = sheet
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// filterSheetsForProject returns which of the given hashes carry a
// sheet_blob_ref row for the project. This is the stored ACL fact behind the
// scoped accessors; it reads no content.
func (s *Store) filterSheetsForProject(ctx context.Context, projectID string, sha256Hexes []string) (map[string]bool, error) {
	result := map[string]bool{}
	sha256Hexes = validSheetSha256Hexes(sha256Hexes)
	if len(sha256Hexes) == 0 {
		return result, nil
	}

	q := qb.Q().Space(`
		SELECT encode(sha256, 'hex')
		FROM sheet_blob_ref
		WHERE project = ?
		AND sha256 IN (SELECT decode(unnest(CAST(? AS TEXT[])), 'hex'))`,
		projectID, sha256Hexes)

	query, args, err := q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}

	rows, err := s.GetDB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var sha256Hex string
		if err := rows.Scan(&sha256Hex); err != nil {
			return nil, err
		}
		result[sha256Hex] = true
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// CreateSheets creates sheet blobs using content-addressed storage and records
// a sheet_blob_ref row granting the project read access to each hash.
// Duplicate statements share the same blob (ON CONFLICT DO NOTHING); refs are
// per project.
//
// Lifecycle policy: creating a sheet requires an active project. The
// transaction takes the project purge fence before any row lock, then locks
// the project row and rejects a missing or deleted project with NotFound,
// serializing against project purge in both directions. Both inserts are
// new-row-only; the foreign-key checks on project and sheet_blob are covered
// by the fence and the project row lock.
func (s *Store) CreateSheets(ctx context.Context, projectID string, creates ...*SheetMessage) ([]*SheetMessage, error) {
	if projectID == "" {
		return nil, errors.New("project is required to create sheets")
	}
	if len(creates) == 0 {
		return creates, nil
	}

	var statements []string
	var sha256s [][]byte
	for _, c := range creates {
		statements = append(statements, c.Statement)
		h := sha256.Sum256([]byte(c.Statement))
		c.Sha256 = hex.EncodeToString(h[:])
		sha256s = append(sha256s, h[:])
		c.Size = int64(len(c.Statement))
	}

	tx, err := s.GetDB().BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.Rollback()

	if err := acquireProjectPurgeLock(ctx, tx, projectID); err != nil {
		return nil, errors.Wrapf(err, "failed to lock project purge fence for %s", projectID)
	}
	if err := lockActiveProject(ctx, tx, projectID); err != nil {
		return nil, err
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
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return nil, errors.Wrapf(err, "failed to insert sheet blobs")
	}

	q = qb.Q().Space(`
		INSERT INTO sheet_blob_ref (project, sha256)
		SELECT ?, unnest(CAST(? AS BYTEA[]))
		ON CONFLICT DO NOTHING
	`, projectID, sha256s)
	query, args, err = q.ToSQL()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build sql")
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return nil, errors.Wrapf(err, "failed to insert sheet blob refs")
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}
	return creates, nil
}
