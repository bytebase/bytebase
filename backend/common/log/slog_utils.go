package log

import (
	"log/slog"
	"os"
	"runtime/debug"
	"strings"
)

var GLogLevel *slog.LevelVar

// Initializes the slog configuration.
func init() {
	GLogLevel = new(slog.LevelVar)

	// https://sourcegraph.com/github.com/uber-go/zap/-/blob/zapcore/entry.go?L117
	replace := func(groups []string, a slog.Attr) slog.Attr {
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

	textHandler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{AddSource: true, Level: GLogLevel, ReplaceAttr: replace})
	slog.SetDefault(slog.New(textHandler))
}

func BBError(err error) slog.Attr {
	var value string
	if err == nil {
		value = "no-op"
	} else {
		value = err.Error()
	}
	return slog.String("error", value)
}

func BBStack(key string) slog.Attr {
	return slog.Any(key, debug.Stack())
}

func BBStrings(key string, ss []string) slog.Attr {
	return slog.Any(key, ss)
}
