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

// GenerateInstanceID creates guaranteed-unique instance identifier
// Format: {source}-{timestamp}-{random}
// Example: "bytebase-prod-20251029-150405-a1b2c3d4e5f6g7h8"
//
// Components:
//   - source: Hostname or BYTEBASE_INSTANCE_ID env var (truncated to 190 chars)
//   - timestamp: YYYYMMDD-HHMMSS UTC (15 chars)
//   - random: 16 alphanumeric chars (cryptographically secure)
//
// Total max length: 190 + 1 + 15 + 1 + 16 = 223 chars
func GenerateInstanceID() (string, error) {
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

	// Truncate source to guarantee total length < 255
	const maxSourceLen = 190
	if len(source) > maxSourceLen {
		// Hash excess to preserve uniqueness
		hash := sha256.Sum256([]byte(source))
		hashHex := hex.EncodeToString(hash[:4])
		source = source[:170] + "-" + hashHex
	}

	// Timestamp: YYYYMMDD-HHMMSS (15 chars)
	timestamp := time.Now().UTC().Format("20060102-150405")

	// Random: 16 alphanumeric chars (cryptographically secure)
	randomStr, err := common.RandomString(16)
	if err != nil {
		return "", errors.Wrap(err, "failed to generate random string")
	}

	// Combine: source-timestamp-random
	instanceID := fmt.Sprintf("%s-%s-%s", source, timestamp, randomStr)

	return instanceID, nil
}

// ValidateInstanceID checks if instance ID meets constraints
func ValidateInstanceID(id string) error {
	if id == "" {
		return errors.New("instance_id cannot be empty")
	}
	if len(id) > 255 {
		return errors.Errorf("instance_id too long: %d chars (max 255)", len(id))
	}
	return nil
}

// GenerateInstanceIDWithRetry generates instance ID and verifies uniqueness
// Retries on collision (extremely rare with 16-char random)
//
// IMPORTANT: Pass *sql.DB directly (not store.Store) to avoid circular dependency
//
// TODO(PR#2): Refactor to service layer pattern when implementing Logger
// - Add UNIQUE constraint on audit_log instance_id (or composite key)
// - Create audit/service.go for orchestration
// - Store should return ErrDuplicateID on constraint violation
// - Move retry logic to service layer (domain/data/service separation)
func GenerateInstanceIDWithRetry(db *sql.DB, maxAttempts int) (string, error) {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		instanceID, err := GenerateInstanceID()
		if err != nil {
			return "", errors.Wrap(err, "failed to generate instance_id")
		}

		// Verify uniqueness by checking JSONB payload
		// Note: protojson marshaling produces camelCase "instanceId"
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		var count int
		err = db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM audit_log
			 WHERE payload->>'instanceId' = $1`,
			instanceID).Scan(&count)
		cancel()

		if err != nil {
			return "", errors.Wrap(err, "failed to check instance_id uniqueness")
		}

		if count == 0 {
			return instanceID, nil
		}

		// Collision detected (extremely rare)
		slog.Warn("Instance ID collision detected, retrying",
			slog.String("instance_id", instanceID),
			slog.Int("attempt", attempt))
	}

	return "", errors.Errorf("failed to generate unique instance_id after %d attempts", maxAttempts)
}
