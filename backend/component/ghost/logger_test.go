package ghost

import (
	"bytes"
	"context"
	"log/slog"
	"testing"

	"github.com/bytebase/bytebase/backend/common/log"

	"github.com/stretchr/testify/require"
)

func TestGhostLoggerUsesContextAttrs(t *testing.T) {
	var buf bytes.Buffer
	prev := slog.Default()
	t.Cleanup(func() { slog.SetDefault(prev) })
	slog.SetDefault(slog.New(log.NewContextHandler(slog.NewTextHandler(&buf, nil))))
	ctx := log.WithAttrs(context.Background(),
		slog.String("project", "db333"),
		slog.Int64("task_run_id", 9213),
	)

	newGhostLogger(ctx).Infof("Migrating %s.%s", "db_1", "tpri")

	output := buf.String()
	require.Contains(t, output, `project=db333`)
	require.Contains(t, output, `task_run_id=9213`)
	require.NotContains(t, output, `replica_id=`)
	require.Contains(t, output, `msg="Migrating db_1.tpri"`)
	require.NotContains(t, output, `!BADKEY`)
}
