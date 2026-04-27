package v1

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"time"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/common/log"
	"github.com/bytebase/bytebase/backend/store"
)

// jitGrantMissDiag captures the bare-minimum forensic information about a JIT
// grant that *almost* matched a query but didn't. It deliberately excludes the
// SQL itself — operators get bytes and content hashes, never customer SQL in
// logs.
type jitGrantMissDiag struct {
	GrantID         string
	Expired         bool
	StoredBytes     int
	SubmittedBytes  int
	StoredSHA256    string
	SubmittedSHA256 string
}

// summarizeJITGrantMisses returns one diag entry per non-nil candidate. Pure
// function; no logging, no clock dependency. The caller passes `now` so tests
// stay deterministic.
func summarizeJITGrantMisses(candidates []*store.AccessGrantMessage, submitted string, now time.Time) []jitGrantMissDiag {
	if len(candidates) == 0 {
		return nil
	}
	submittedHash := sha256.Sum256([]byte(submitted))
	submittedHashHex := hex.EncodeToString(submittedHash[:])
	var out []jitGrantMissDiag
	for _, c := range candidates {
		if c == nil || c.Payload == nil {
			continue
		}
		storedHash := sha256.Sum256([]byte(c.Payload.Query))
		out = append(out, jitGrantMissDiag{
			GrantID:         c.ID,
			Expired:         c.ExpireTime != nil && c.ExpireTime.Before(now),
			StoredBytes:     len(c.Payload.Query),
			SubmittedBytes:  len(submitted),
			StoredSHA256:    hex.EncodeToString(storedHash[:]),
			SubmittedSHA256: submittedHashHex,
		})
	}
	return out
}

// logJITGrantMissDiagnostic is the operational entry point. It is gated on
// debug-level logging being enabled — in production (default INFO) this
// short-circuits before any extra DB call, which matters because for any
// JIT-enabled project, len(grants) == 0 in preCheckAccess is the *normal* path
// for queries that simply don't need a JIT grant. We must not pay a per-query
// cost there.
//
// When debug logging is on, the function runs one bounded extra ListAccessGrants
// scoped to ACTIVE grants targeting this exact database — i.e. only candidates
// that *would have matched* if not for the query/expiry comparison — and emits
// one slog.Debug per candidate with byte counts and SHA-256 of stored vs.
// submitted SQL. That's enough to localize whitespace/EOL drift in seconds
// without ever logging customer SQL.
func (s *SQLService) logJITGrantMissDiagnostic(
	ctx context.Context,
	databaseFullName string,
	projectID string,
	userEmail string,
	submitted string,
) {
	if !slog.Default().Enabled(ctx, slog.LevelDebug) {
		return
	}
	diagFilter := fmt.Sprintf(`status == "ACTIVE" && target == %q`, databaseFullName)
	diagFilterQ, err := store.GetListAccessGrantFilter(diagFilter)
	if err != nil {
		slog.Debug("jit grant miss diagnostic: failed to build filter", log.BBError(err))
		return
	}
	candidates, err := s.store.ListAccessGrants(ctx, &store.FindAccessGrantMessage{
		Workspace: common.GetWorkspaceIDFromContext(ctx),
		ProjectID: &projectID,
		Creator:   &userEmail,
		FilterQ:   diagFilterQ,
	})
	if err != nil {
		slog.Debug("jit grant miss diagnostic: failed to list candidates", log.BBError(err))
		return
	}
	for _, d := range summarizeJITGrantMisses(candidates, submitted, time.Now()) {
		slog.Debug("jit grant candidate did not match",
			slog.String("grant_id", d.GrantID),
			slog.String("database", databaseFullName),
			slog.Bool("expired", d.Expired),
			slog.Int("query_bytes_stored", d.StoredBytes),
			slog.Int("query_bytes_submitted", d.SubmittedBytes),
			slog.String("query_sha256_stored", d.StoredSHA256),
			slog.String("query_sha256_submitted", d.SubmittedSHA256),
		)
	}
}
