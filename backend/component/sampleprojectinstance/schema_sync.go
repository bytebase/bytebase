package sampleprojectinstance

import (
	"context"

	"github.com/bytebase/bytebase/backend/runner/schemasync"
	"github.com/bytebase/bytebase/backend/store"
)

// SyncerSchemaSync adapts the ordinary schema syncer to the synchronous
// readiness requirement for sample Project Instances.
type SyncerSchemaSync struct {
	syncer *schemasync.Syncer
}

// NewSyncerSchemaSync creates a schema-sync adapter.
func NewSyncerSchemaSync(syncer *schemasync.Syncer) *SyncerSchemaSync {
	return &SyncerSchemaSync{syncer: syncer}
}

// SyncInstance discovers databases synchronously and returns only databases
// newly discovered during this exact sync.
func (s *SyncerSchemaSync) SyncInstance(
	ctx context.Context,
	instance *store.InstanceMessage,
) (*store.InstanceMessage, []*store.DatabaseMessage, error) {
	updated, _, databases, err := s.syncer.SyncInstance(ctx, instance)
	if err != nil {
		return nil, nil, err
	}
	return updated, databases, nil
}

// SyncDatabasesAsync resumes normal asynchronous schema synchronization.
func (s *SyncerSchemaSync) SyncDatabasesAsync(databases []*store.DatabaseMessage) {
	s.syncer.SyncDatabasesAsync(databases)
}
