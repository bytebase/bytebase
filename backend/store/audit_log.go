package store

import (
	"context"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/pkg/errors"

	storepb "github.com/bytebase/bytebase/proto/generated-go/store"
)

type AuditLog struct {
}

func (s *Store) CreateAuditLog(ctx context.Context, payload *storepb.AuditLog) error {
	if !s.profile.DevelopmentAudit {
		return nil
	}
	query := `
		INSERT INTO audit_log (payload) VALUES ($1)
	`

	p, err := protojson.Marshal(payload)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal payload")
	}

	if _, err := s.db.db.ExecContext(ctx, query, p); err != nil {
		return errors.Wrapf(err, "failed to create audit log")
	}
	return nil
}
