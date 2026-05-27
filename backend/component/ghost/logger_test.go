package ghost

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGhostLoggerUsesScopedLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil)).With(
		slog.String("project", "db333"),
		slog.Int64("task_run_id", 9213),
	)

	newGhostLogger(logger).Infof("Migrating %s.%s", "db_1", "tpri")

	output := buf.String()
	require.Contains(t, output, `project=db333`)
	require.Contains(t, output, `task_run_id=9213`)
	require.NotContains(t, output, `replica_id=`)
	require.Contains(t, output, `msg="Migrating db_1.tpri"`)
	require.NotContains(t, output, `!BADKEY`)
}
