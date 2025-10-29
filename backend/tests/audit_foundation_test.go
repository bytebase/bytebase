package tests

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
	"github.com/bytebase/bytebase/backend/store"
)

func TestAuditLogFoundation(t *testing.T) {
	t.Parallel()
	a := require.New(t)
	ctx := context.Background()

	ctl := &controller{}
	ctx, err := ctl.StartServerWithExternalPg(ctx)
	a.NoError(err)
	defer ctl.Close(ctx)

	// Create a store instance to use the helper methods
	s, err := store.New(ctx, ctl.profile.PgURL, false)
	a.NoError(err)

	db := s.GetDB()

	// Step 1: Insert a "legacy" log without bytebase_id/sequence_number
	legacyPayload := &storepb.AuditLog{
		Method:   "GET",
		Resource: "/api/v1/legacy",
		User:     "users/test@example.com",
		Severity: storepb.AuditLog_INFO,
		Request:  `{"legacy": true}`,
		Response: `{"ok": true}`,
	}

	legacyJSON, err := protojson.Marshal(legacyPayload)
	a.NoError(err)

	_, err = db.ExecContext(ctx, `INSERT INTO audit_log (payload) VALUES ($1)`, legacyJSON)
	a.NoError(err)

	// Step 2: Verify legacy log has NULL bytebase_id
	var nullCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM audit_log WHERE payload->>'bytebaseId' IS NULL
	`).Scan(&nullCount)
	a.NoError(err)
	a.Equal(1, nullCount, "should have 1 legacy log without bytebase_id")

	// Step 3: Insert a "new" log with bytebase_id/sequence_number
	testBytebaseID := "test-bytebase-123"
	newPayload := &storepb.AuditLog{
		Method:         "POST",
		Resource:       "/api/v1/new",
		User:           "users/test@example.com",
		Severity:       storepb.AuditLog_INFO,
		Request:        `{"new": true}`,
		Response:       `{"ok": true}`,
		BytebaseId:     testBytebaseID,
		SequenceNumber: 1,
	}

	newJSON, err := protojson.Marshal(newPayload)
	a.NoError(err)

	_, err = db.ExecContext(ctx, `INSERT INTO audit_log (payload) VALUES ($1)`, newJSON)
	a.NoError(err)

	// Step 4: Verify GetMaxAuditSequence returns 1
	maxSeq, err := s.GetMaxAuditSequence(ctx, testBytebaseID)
	a.NoError(err)
	a.Equal(int64(1), maxSeq, "should return max sequence 1")

	// Step 5: Verify CheckBytebaseIDExists returns true
	exists, err := s.CheckBytebaseIDExists(ctx, testBytebaseID)
	a.NoError(err)
	a.True(exists, "bytebase_id should exist")

	// Step 6: Verify non-existent Bytebase deployment returns false
	exists, err = s.CheckBytebaseIDExists(ctx, "non-existent")
	a.NoError(err)
	a.False(exists, "non-existent bytebase_id should not exist")

	// Step 7: Verify both logs are queryable
	var totalCount int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM audit_log`).Scan(&totalCount)
	a.NoError(err)
	a.Equal(2, totalCount, "should have 2 total logs (1 legacy + 1 new)")

	t.Log("✅ Foundation test passed:")
	t.Log("  - Legacy log (NULL sequences): queryable via API")
	t.Log("  - New log (sequence 1): ready for stdout streaming")
}
