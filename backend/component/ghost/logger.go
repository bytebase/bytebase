package ghost

import (
	"fmt"
	"log/slog"

	"github.com/openark/golib/log"
	"github.com/pkg/errors"
)

type ghostLogger struct{}

func newGhostLogger() *ghostLogger {
	return &ghostLogger{}
}

func (*ghostLogger) Debug(args ...any) {
	slog.Debug(fmt.Sprintf(args[0].(string), args[1:]))
}

func (*ghostLogger) Debugf(format string, args ...any) {
	slog.Debug(format, args...)
}

func (*ghostLogger) Info(args ...any) {
	slog.Info(fmt.Sprintf(args[0].(string), args[1:]))
}

func (*ghostLogger) Infof(format string, args ...any) {
	slog.Info(format, args...)
}

func (*ghostLogger) Warning(args ...any) error {
	slog.Warn(fmt.Sprintf(args[0].(string), args[1:]))
	return errors.Errorf(args[0].(string), args[1:])
}

func (*ghostLogger) Warningf(format string, args ...any) error {
	slog.Warn(format, args...)
	return errors.Errorf(format, args...)
}

func (*ghostLogger) Error(args ...any) error {
	slog.Error(fmt.Sprintf(args[0].(string), args[1:]))
	return errors.Errorf(args[0].(string), args[1:])
}

func (*ghostLogger) Errorf(format string, args ...any) error {
	slog.Error(format, args...)
	return errors.Errorf(format, args...)
}

func (*ghostLogger) Errore(err error) error {
	if err != nil {
		slog.Error(err.Error())
	}
	return err
}

func (*ghostLogger) Fatal(args ...any) error {
	slog.Error(fmt.Sprintf(args[0].(string), args[1:]))
	return errors.Errorf(args[0].(string), args[1:])
}

func (*ghostLogger) Fatalf(format string, args ...any) error {
	slog.Error(format, args...)
	return errors.Errorf(format, args...)
}

func (*ghostLogger) Fatale(err error) error {
	if err != nil {
		slog.Error(err.Error())
	}
	return err
}

func (*ghostLogger) SetLevel(_ log.LogLevel) {
}

func (*ghostLogger) SetPrintStackTrace(_ bool) {
}
