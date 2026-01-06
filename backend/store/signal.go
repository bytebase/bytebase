package store

import (
	"context"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"

	storepb "github.com/bytebase/bytebase/backend/generated-go/store"
)

// SignalChannel is the PostgreSQL NOTIFY channel for HA coordination.
const SignalChannel = "bytebase_signal"

// SendSignal sends a notification to the bytebase_signal channel.
func (s *Store) SendSignal(ctx context.Context, signalType storepb.Signal_Type, uid int32) error {
	payload, err := protojson.Marshal(&storepb.Signal{
		Type: signalType,
		Uid:  uid,
	})
	if err != nil {
		return errors.Wrap(err, "failed to marshal signal payload")
	}
	_, err = s.GetDB().ExecContext(ctx, "SELECT pg_notify($1, $2)", SignalChannel, string(payload))
	return errors.Wrap(err, "failed to send signal")
}
