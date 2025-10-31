package audit

import (
	"context"

	"github.com/bytebase/bytebase/backend/common"
	"github.com/bytebase/bytebase/backend/store"
)

// Service orchestrates audit logging operations
// This is the service/application layer that coordinates between domain logic and data layer
type Service struct {
	store *store.Store
}

// NewService creates a new audit service
func NewService(store *store.Store) *Service {
	return &Service{
		store: store,
	}
}

// GenerateUniqueBytebaseIDWithRetry generates a unique Bytebase ID
// The store layer handles collision retries internally (up to maxAttempts)
// No service-layer retry needed since collisions are not transient errors
func (s *Service) GenerateUniqueBytebaseIDWithRetry(ctx context.Context, maxAttempts int) (string, error) {
	return s.store.EnsureUniqueBytebaseID(ctx, GenerateBytebaseID, maxAttempts)
}

// LoadMaxSequenceWithRetry loads the maximum sequence number with retry
// Uses common.Retry for consistency with rest of codebase
func (s *Service) LoadMaxSequenceWithRetry(ctx context.Context, bytebaseID string) (int64, error) {
	var maxSeq int64
	err := common.Retry(ctx, func() error {
		var retryErr error
		maxSeq, retryErr = s.store.GetMaxAuditSequence(ctx, bytebaseID)
		return retryErr
	})
	return maxSeq, err
}
