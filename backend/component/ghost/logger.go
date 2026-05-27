package ghost

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/openark/golib/log"
	"github.com/pkg/errors"
)

type ghostLogger struct {
	ctx context.Context
}

func newGhostLogger(ctx context.Context) *ghostLogger {
	if ctx == nil {
		ctx = context.Background()
	}
	return &ghostLogger{ctx: ctx}
}

func (l *ghostLogger) Debug(args ...any) {
	slog.DebugContext(l.ctx, fmt.Sprintf(args[0].(string), args[1:]...))
}

func (l *ghostLogger) Debugf(format string, args ...any) {
	slog.DebugContext(l.ctx, fmt.Sprintf(format, args...))
}

func (l *ghostLogger) Info(args ...any) {
	slog.InfoContext(l.ctx, fmt.Sprintf(args[0].(string), args[1:]...))
}

func (l *ghostLogger) Infof(format string, args ...any) {
	slog.InfoContext(l.ctx, fmt.Sprintf(format, args...))
}

func (l *ghostLogger) Warning(args ...any) error {
	slog.WarnContext(l.ctx, fmt.Sprintf(args[0].(string), args[1:]...))
	return errors.Errorf(args[0].(string), args[1:])
}

func (l *ghostLogger) Warningf(format string, args ...any) error {
	slog.WarnContext(l.ctx, fmt.Sprintf(format, args...))
	return errors.Errorf(format, args...)
}

func (l *ghostLogger) Error(args ...any) error {
	slog.ErrorContext(l.ctx, fmt.Sprintf(args[0].(string), args[1:]...))
	return errors.Errorf(args[0].(string), args[1:])
}

func (l *ghostLogger) Errorf(format string, args ...any) error {
	slog.ErrorContext(l.ctx, fmt.Sprintf(format, args...))
	return errors.Errorf(format, args...)
}

func (l *ghostLogger) Errore(err error) error {
	if err != nil {
		slog.ErrorContext(l.ctx, err.Error())
	}
	return err
}

func (l *ghostLogger) Fatal(args ...any) error {
	slog.ErrorContext(l.ctx, fmt.Sprintf(args[0].(string), args[1:]...))
	return errors.Errorf(args[0].(string), args[1:])
}

func (l *ghostLogger) Fatalf(format string, args ...any) error {
	slog.ErrorContext(l.ctx, fmt.Sprintf(format, args...))
	return errors.Errorf(format, args...)
}

func (l *ghostLogger) Fatale(err error) error {
	if err != nil {
		slog.ErrorContext(l.ctx, err.Error())
	}
	return err
}

func (*ghostLogger) SetLevel(_ log.LogLevel) {
}

func (*ghostLogger) SetPrintStackTrace(_ bool) {
}
