package audit

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/bytebase/bytebase/backend/common"
)

// GenerateBytebaseID creates guaranteed-unique Bytebase deployment identifier
// Format: {source}-{timestamp}-{random}
// Example: "bytebase-prod-20251029-150405-a1b2c3d4e5f6g7h8"
//
// Components:
//   - source: Hostname or BYTEBASE_INSTANCE_ID env var (truncated to 190 chars)
//   - timestamp: YYYYMMDD-HHMMSS UTC (15 chars)
//   - random: 16 alphanumeric chars (cryptographically secure)
//
// Total max length: 190 + 1 + 15 + 1 + 16 = 223 chars
func GenerateBytebaseID() (string, error) {
	// Length constraints for Bytebase ID components
	const (
		maxTotalLen  = 255                                        // Database VARCHAR(255) limit
		timestampLen = 15                                         // YYYYMMDD-HHMMSS format
		randomLen    = 16                                         // Alphanumeric random string
		maxSourceLen = maxTotalLen - 2 - timestampLen - randomLen // 190 (2 dashes)
		hashLen      = 8                                          // 4 bytes -> 8 hex chars
		truncatedLen = maxSourceLen - 1 - hashLen                 // 181 (reserve space for -hash)
	)

	// Source priority: BYTEBASE_INSTANCE_ID > HOSTNAME > os.Hostname() > "bytebase"
	source := os.Getenv("BYTEBASE_INSTANCE_ID")
	if source == "" {
		source = os.Getenv("HOSTNAME")
	}
	if source == "" {
		hostname, _ := os.Hostname()
		if hostname != "" {
			source = hostname
		} else {
			source = "bytebase"
		}
	}

	// Truncate source to guarantee total length <= 255
	if len(source) > maxSourceLen {
		// Hash excess to preserve uniqueness
		hash := sha256.Sum256([]byte(source))
		hashHex := hex.EncodeToString(hash[:4])
		source = fmt.Sprintf("%s-%s", source[:truncatedLen], hashHex)
	}

	// Timestamp: YYYYMMDD-HHMMSS (15 chars)
	timestamp := time.Now().UTC().Format("20060102-150405")

	// Random: 16 alphanumeric chars (cryptographically secure)
	randomStr, err := common.RandomString(16)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate random string")
	}

	// Combine: source-timestamp-random
	bytebaseID := fmt.Sprintf("%s-%s-%s", source, timestamp, randomStr)

	return bytebaseID, nil
}

// ValidateBytebaseID checks if Bytebase ID meets constraints
func ValidateBytebaseID(id string) error {
	if id == "" {
		return errors.New("bytebase_id cannot be empty")
	}
	if len(id) > 255 {
		return errors.Errorf("bytebase_id too long: %d chars (max 255)", len(id))
	}
	return nil
}

// GenerateBytebaseIDWithRetry generates Bytebase ID and verifies uniqueness
// Retries on collision (extremely rare with 16-char random)
//
// IMPORTANT: Pass *sql.DB directly (not store.Store) to avoid circular dependency
//
// TODO(PR#2): Refactor to service layer pattern when implementing Logger
// - Add UNIQUE constraint on audit_log bytebase_id (or composite key)
// - Create audit/service.go for orchestration
// - Store should return ErrDuplicateID on constraint violation
// - Move retry logic to service layer (domain/data/service separation)
func GenerateBytebaseIDWithRetry(db *sql.DB, maxAttempts int) (string, error) {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		bytebaseID, err := GenerateBytebaseID()
		if err != nil {
			return "", errors.Wrap(err, "failed to generate bytebase_id")
		}

		// Verify uniqueness by checking JSONB payload
		// Note: protojson marshaling produces camelCase "bytebaseId"
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		var count int
		err = db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM audit_log
			 WHERE payload->>'bytebaseId' = $1`,
			bytebaseID).Scan(&count)
		cancel()

		if err != nil {
			return "", errors.Wrap(err, "failed to check bytebase_id uniqueness")
		}

		if count == 0 {
			return bytebaseID, nil
		}

		// Collision detected (extremely rare)
		slog.Warn("Bytebase ID collision detected, retrying",
			slog.String("bytebase_id", bytebaseID),
			slog.Int("attempt", attempt))
	}

	return "", errors.Errorf("failed to generate unique bytebase_id after %d attempts", maxAttempts)
}
