package log //nolint:revive // intentional package name

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/bytebase/bytebase/backend/common/stacktrace"
)

// LogLevel is the default log severity level.
var LogLevel = new(slog.LevelVar)

// https://sourcegraph.com/github.com/uber-go/zap/-/blob/zapcore/entry.go?L117
// Replace is the default replace attribute.
var Replace = func(_ []string, a slog.Attr) slog.Attr {
	if a.Key == slog.SourceKey {
		if source, ok := a.Value.Any().(*slog.Source); ok {
			idx := strings.LastIndexByte(source.File, '/')
			if idx == -1 {
				return a
			}
			// Find the penultimate separator.
			idx = strings.LastIndexByte(source.File[:idx], '/')
			if idx == -1 {
				return a
			}
			source.File = source.File[idx+1:]
		}
	}
	return a
}

// Initializes the slog configuration.
func init() {
	LogLevel.Set(slog.LevelInfo)
}

func BBError(err error) slog.Attr {
	return slog.String("error", fmt.Sprintf("%+v", err))
}

func BBStack(key string) slog.Attr {
	stack := stacktrace.TakeStacktrace(20 /* n */, 3 /* skip */)
	return slog.Any(key, stack)
}
