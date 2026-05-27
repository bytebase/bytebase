package ghost

import (
	"fmt"
	"log/slog"

	"github.com/openark/golib/log"
	"github.com/pkg/errors"
)

type ghostLogger struct {
	logger *slog.Logger
}

func newGhostLogger(logger *slog.Logger) *ghostLogger {
	if logger == nil {
		logger = slog.Default()
	}
	return &ghostLogger{logger: logger}
}

func (l *ghostLogger) Debug(args ...any) {
	l.logger.Debug(fmt.Sprintf(args[0].(string), args[1:]...))
}

func (l *ghostLogger) Debugf(format string, args ...any) {
	l.logger.Debug(fmt.Sprintf(format, args...))
}

func (l *ghostLogger) Info(args ...any) {
	l.logger.Info(fmt.Sprintf(args[0].(string), args[1:]...))
}

func (l *ghostLogger) Infof(format string, args ...any) {
	l.logger.Info(fmt.Sprintf(format, args...))
}

func (l *ghostLogger) Warning(args ...any) error {
	l.logger.Warn(fmt.Sprintf(args[0].(string), args[1:]...))
	return errors.Errorf(args[0].(string), args[1:])
}

func (l *ghostLogger) Warningf(format string, args ...any) error {
	l.logger.Warn(fmt.Sprintf(format, args...))
	return errors.Errorf(format, args...)
}

func (l *ghostLogger) Error(args ...any) error {
	l.logger.Error(fmt.Sprintf(args[0].(string), args[1:]...))
	return errors.Errorf(args[0].(string), args[1:])
}

func (l *ghostLogger) Errorf(format string, args ...any) error {
	l.logger.Error(fmt.Sprintf(format, args...))
	return errors.Errorf(format, args...)
}

func (l *ghostLogger) Errore(err error) error {
	if err != nil {
		l.logger.Error(err.Error())
	}
	return err
}

func (l *ghostLogger) Fatal(args ...any) error {
	l.logger.Error(fmt.Sprintf(args[0].(string), args[1:]...))
	return errors.Errorf(args[0].(string), args[1:])
}

func (l *ghostLogger) Fatalf(format string, args ...any) error {
	l.logger.Error(fmt.Sprintf(format, args...))
	return errors.Errorf(format, args...)
}

func (l *ghostLogger) Fatale(err error) error {
	if err != nil {
		l.logger.Error(err.Error())
	}
	return err
}

func (*ghostLogger) SetLevel(_ log.LogLevel) {
}

func (*ghostLogger) SetPrintStackTrace(_ bool) {
}
